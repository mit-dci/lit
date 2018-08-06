package qln

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"

	"github.com/adiabat/bech32"
	"github.com/btcsuite/fastsha256"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/lnutil"
)

func (nd *LitNode) PayMultihop(dstLNAdr string, coinType uint32, amount int64) (bool, error) {
	var targetAdr [20]byte
	_, adr, err := bech32.Decode(dstLNAdr)
	if err != nil {
		return false, err
	}

	wal, ok := nd.SubWallet[coinType]
	if !ok {
		return false, fmt.Errorf("not connected to cointype %d", coinType)
	}

	fee := wal.Fee() * 1000

	if amount < consts.MinOutput+fee {
		return false, fmt.Errorf("cannot send %d because it's less than minOutput + fee: %d", amount, consts.MinOutput+fee)
	}

	//Connect to the node
	if _, err := nd.FindPeerIndexByAddress(dstLNAdr); err != nil {
		err = nd.DialPeer(dstLNAdr)
		if err != nil {
			return false, fmt.Errorf("error connected to destination node for multihop: %s", err.Error())
		}
	}

	copy(targetAdr[:], adr)
	log.Printf("Finding route to %s", dstLNAdr)
	path, err := nd.FindPath(targetAdr, coinType, amount, fee)
	if err != nil {
		return false, err
	}
	log.Printf("Done route to %s", dstLNAdr)

	idx, err := nd.FindPeerIndexByAddress(dstLNAdr)
	if err != nil {
		return false, err
	}

	inFlight := new(InFlightMultihop)
	inFlight.Path = path
	inFlight.Amt = amount
	inFlight.Cointype = coinType

	nd.MultihopMutex.Lock()
	nd.InProgMultihop = append(nd.InProgMultihop, inFlight)
	nd.MultihopMutex.Unlock()

	log.Printf("Sending payment request to %s", dstLNAdr)
	msg := lnutil.NewMultihopPaymentRequestMsg(idx, coinType)
	nd.OmniOut <- msg
	log.Printf("Done sending payment request to %s", dstLNAdr)
	return true, nil
}

func (nd *LitNode) MultihopPaymentRequestHandler(msg lnutil.MultihopPaymentRequestMsg) error {
	// Generate private preimage and send ack with the hash
	fmt.Printf("Received multihop payment request from peer %d\n", msg.Peer())
	inFlight := new(InFlightMultihop)
	var pkh [20]byte

	id, _ := nd.GetPubHostFromPeerIdx(msg.Peer())
	idHash := fastsha256.Sum256(id[:])
	copy(pkh[:], idHash[:20])
	inFlight.Path = [][20]byte{pkh}

	inFlight.Cointype = msg.Cointype
	rand.Read(inFlight.PreImage[:])
	hash := fastsha256.Sum256(inFlight.PreImage[:])

	inFlight.HHash = hash

	nd.MultihopMutex.Lock()
	nd.InProgMultihop = append(nd.InProgMultihop, inFlight)
	nd.MultihopMutex.Unlock()

	outMsg := lnutil.NewMultihopPaymentAckMsg(msg.Peer(), hash)
	nd.OmniOut <- outMsg
	return nil
}

func (nd *LitNode) MultihopPaymentAckHandler(msg lnutil.MultihopPaymentAckMsg) error {
	fmt.Printf("Received multihop payment ack from peer %d, hash %x\n", msg.Peer(), msg.HHash)

	nd.MultihopMutex.Lock()
	defer nd.MultihopMutex.Unlock()
	for idx, mh := range nd.InProgMultihop {
		var nullHash [32]byte
		if !mh.Succeeded && bytes.Equal(nullHash[:], mh.HHash[:]) {
			targetNode := mh.Path[len(mh.Path)-1]
			targetIdx, err := nd.FindPeerIndexByAddress(bech32.Encode("ln", targetNode[:]))
			if err != nil {
				return fmt.Errorf("not connected to destination peer")
			}
			if msg.Peer() == targetIdx {
				fmt.Printf("Found the right pending multihop. Sending setup msg to first hop\n")
				// found the right one. Set this up
				firstHop := mh.Path[1]
				firstHopIdx, err := nd.FindPeerIndexByAddress(bech32.Encode("ln", firstHop[:]))
				if err != nil {
					return fmt.Errorf("not connected to first hop in route")
				}

				nd.RemoteMtx.Lock()
				var qc *Qchan
				for _, ch := range nd.RemoteCons[firstHopIdx].QCs {
					if ch.Coin() == mh.Cointype && ch.State.MyAmt-consts.MinOutput-ch.State.Fee >= mh.Amt && !ch.CloseData.Closed && !ch.State.Failed {
						qc = ch
						break
					}
				}

				if qc == nil {
					nd.RemoteMtx.Unlock()
					return fmt.Errorf("could not find suitable channel to route payment")
				}

				nd.RemoteMtx.Unlock()

				nd.InProgMultihop[idx].HHash = msg.HHash

				// Calculate what initial locktime we need
				wal, ok := nd.SubWallet[mh.Cointype]
				if !ok {
					return fmt.Errorf("not connected to wallet for cointype %d", mh.Cointype)
				}

				height := wal.CurrentHeight()

				// Allow 5 blocks of leeway per hop in case people's wallets are out of sync
				locktime := height + int32(len(mh.Path)*(consts.DefaultLockTime+5))

				// This handler needs to return before OfferHTLC can work
				go func() {
					log.Printf("offering HTLC with RHash: %x", msg.HHash)
					err = nd.OfferHTLC(qc, uint32(mh.Amt), msg.HHash, uint32(locktime), [32]byte{})
					if err != nil {
						log.Printf("error offering HTLC: %s", err.Error())
						return
					}

					var data [32]byte
					outMsg := lnutil.NewMultihopPaymentSetupMsg(firstHopIdx, mh.Amt, mh.Cointype, msg.HHash, mh.Path, data)
					fmt.Printf("Sending multihoppaymentsetup to peer %d\n", firstHopIdx)
					nd.OmniOut <- outMsg
				}()

				break
			}
		}
	}
	return nil
}

func (nd *LitNode) MultihopPaymentSetupHandler(msg lnutil.MultihopPaymentSetupMsg) error {
	fmt.Printf("Received multihop payment setup from peer %d, hash %x\n", msg.Peer(), msg.HHash)

	var nullBytes [16]byte
	nd.MultihopMutex.Lock()
	defer nd.MultihopMutex.Unlock()
	for _, mh := range nd.InProgMultihop {
		hash := fastsha256.Sum256(mh.PreImage[:])
		if !bytes.Equal(mh.PreImage[:], nullBytes[:]) && bytes.Equal(msg.HHash[:], hash[:]) {
			// We already know this. If we have a Preimage, then we're the receiving
			// end and we should send a settlement message to the
			// predecessor
			go func() {
				_, err := nd.ClaimHTLC(mh.PreImage)
				if err != nil {
					log.Printf("error claiming HTLC: %s", err.Error())
				}
			}()

			return nil
		}
	}

	// Check there is a corresponding incoming HTLC
	HTLCs, chans, err := nd.FindHTLCsByHash(msg.HHash)
	if err != nil {
		return fmt.Errorf("error finding HTLCs: %s", err.Error())
	}

	var found bool
	var locktime uint32
	for idx, h := range HTLCs {
		if h.Incoming && !h.Cleared && !h.Clearing && !h.ClearedOnChain && h.Amt == msg.Amount && chans[idx].Coin() == msg.Cointype {
			found = true
			locktime = h.Locktime
			break
		}

		// We already have an outgoing HTLC with this hash
		if !h.Incoming && !h.Cleared && !h.ClearedOnChain {
			return fmt.Errorf("we already have an uncleared offered HTLC with RHash: %x", msg.HHash)
		}
	}

	if !found {
		return fmt.Errorf("no corresponding incoming HTLC found for multihop payment with RHash: %x", msg.HHash)
	}

	wal, ok := nd.SubWallet[msg.Cointype]
	if !ok {
		return fmt.Errorf("not connected to wallet for cointype %d", msg.Cointype)
	}

	height := wal.CurrentHeight()
	if locktime-consts.DefaultLockTime < uint32(height+consts.DefaultLockTime) {
		return fmt.Errorf("locktime of preceeding hop is too close for comfort: %d, height: %d", locktime-consts.DefaultLockTime, height)
	}

	inFlight := new(InFlightMultihop)
	inFlight.Path = msg.NodeRoute
	inFlight.Amt = msg.Amount
	inFlight.HHash = msg.HHash
	inFlight.Cointype = msg.Cointype
	nd.InProgMultihop = append(nd.InProgMultihop, inFlight)

	// Forward
	var pkh [20]byte
	id := nd.IdKey().PubKey().SerializeCompressed()
	idHash := fastsha256.Sum256(id[:])
	copy(pkh[:], idHash[:20])
	var sendToPkh [20]byte
	for i, node := range inFlight.Path {
		if bytes.Equal(pkh[:], node[:]) {
			sendToPkh = inFlight.Path[i+1]
		}
	}

	lnAdr := bech32.Encode("ln", sendToPkh[:])

	//Connect to the node
	if _, err := nd.FindPeerIndexByAddress(lnAdr); err != nil {
		err = nd.DialPeer(lnAdr)
		if err != nil {
			return fmt.Errorf("error connecting to node for multihop: %s", err.Error())
		}
	}

	sendToIdx, err := nd.FindPeerIndexByAddress(lnAdr)
	if err != nil {
		return fmt.Errorf("not connected to peer in route")
	}

	nd.RemoteMtx.Lock()
	var qc *Qchan
	for _, ch := range nd.RemoteCons[sendToIdx].QCs {
		if ch.Coin() == inFlight.Cointype && ch.State.MyAmt-consts.MinOutput > msg.Amount && !ch.CloseData.Closed && !ch.State.Failed {
			qc = ch
			break
		}
	}

	if qc == nil {
		nd.RemoteMtx.Unlock()
		return fmt.Errorf("could not find suitable channel to route payment")
	}

	nd.RemoteMtx.Unlock()

	// This handler needs to return so run this in a goroutine
	go func() {
		log.Printf("offering HTLC with RHash: %x", msg.HHash)
		err = nd.OfferHTLC(qc, uint32(msg.Amount), msg.HHash, locktime-consts.DefaultLockTime, [32]byte{})
		if err != nil {
			log.Printf("error offering HTLC: %s", err.Error())
			return
		}

		msg.PeerIdx = sendToIdx
		nd.OmniOut <- msg
	}()

	return nil
}
