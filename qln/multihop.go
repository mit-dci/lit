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

	copy(targetAdr[:], adr)
	path, err := nd.FindPath(targetAdr, coinType, amount)

	if err != nil {
		return false, err
	}

	inFlight := new(InFlightMultihop)
	inFlight.Path = path
	inFlight.Amt = amount
	nd.InProgMultihop = append(nd.InProgMultihop, inFlight)

	//Connect to the node
	nd.DialPeer(dstLNAdr)
	idx, err := nd.FindPeerIndexByAddress(dstLNAdr)
	if err != nil {
		return false, err
	}

	msg := lnutil.NewMultihopPaymentRequestMsg(idx)
	nd.OmniOut <- msg
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
	rand.Read(inFlight.PreImage[:])
	nd.InProgMultihop = append(nd.InProgMultihop, inFlight)

	hash := fastsha256.Sum256(inFlight.PreImage[:])
	outMsg := lnutil.NewMultihopPaymentAckMsg(msg.Peer(), hash)
	nd.OmniOut <- outMsg
	return nil
}

func (nd *LitNode) MultihopPaymentAckHandler(msg lnutil.MultihopPaymentAckMsg) error {
	fmt.Printf("Received multihop payment ack from peer %d, hash %x\n", msg.Peer(), msg.HHash)

	for _, mh := range nd.InProgMultihop {
		targetNode := mh.Path[len(mh.Path)-1]
		targetIdx, _ := nd.FindPeerIndexByAddress(bech32.Encode("ln", targetNode[:]))
		if msg.Peer() == targetIdx {
			fmt.Printf("Found the right pending multihop. Sending setup msg to first hop\n")
			// found the right one. Set this up
			firstHop := mh.Path[1]
			firstHopIdx, _ := nd.FindPeerIndexByAddress(bech32.Encode("ln", firstHop[:]))

			nd.RemoteMtx.Lock()
			var qc *Qchan
			for _, ch := range nd.RemoteCons[firstHopIdx].QCs {
				// TODO: should specify the exact channel on the route, not just the peer
				if ch.Coin() == 1 && ch.State.MyAmt-consts.MinOutput > mh.Amt && !ch.CloseData.Closed {
					qc = ch
					break
				}
			}

			if qc == nil {
				return fmt.Errorf("could not find suitable channel to route payment")
			}

			nd.RemoteMtx.Unlock()
			log.Printf("offering HTLC")
			err := nd.OfferHTLC(qc, uint32(mh.Amt), mh.HHash, 100, [32]byte{})
			if err != nil {
				return err
			}

			var data [32]byte
			outMsg := lnutil.NewMultihopPaymentSetupMsg(firstHopIdx, mh.Amt, msg.HHash, mh.Path, data)
			fmt.Printf("Sending multihoppaymentsetup to peer %d\n", firstHopIdx)
			nd.OmniOut <- outMsg
		}
	}
	return nil
}

func (nd *LitNode) MultihopPaymentSetupHandler(msg lnutil.MultihopPaymentSetupMsg) error {
	fmt.Printf("Received multihop payment setup from peer %d, hash %x\n", msg.Peer(), msg.HHash)

	var nullBytes [16]byte
	for _, mh := range nd.InProgMultihop {
		hash := fastsha256.Sum256(mh.PreImage[:])
		if mh.PreImage != nullBytes && bytes.Equal(msg.HHash[:], hash[:]) {
			// We already know this. If we have a Preimage, then we're the receiving
			// end and we should send a settlement message to the
			// predecessor
			go func() {
				_, err := nd.ClaimHTLC(mh.PreImage)
				if err != nil {
					log.Printf("error claiming HTLC: %s", err.Error())
				}

				outMsg := lnutil.NewMultihopPaymentSettleMsg(msg.Peer(), mh.PreImage)
				nd.OmniOut <- outMsg
			}()

			return nil
		}
	}

	inFlight := new(InFlightMultihop)
	inFlight.Path = msg.NodeRoute
	inFlight.Amt = msg.Amount
	inFlight.HHash = msg.HHash
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

	sendToIdx, _ := nd.FindPeerIndexByAddress(bech32.Encode("ln", sendToPkh[:]))

	nd.RemoteMtx.Lock()
	var qc *Qchan
	for _, ch := range nd.RemoteCons[sendToIdx].QCs {
		// TODO: should specify the exact channel on the route, not just the peer
		if ch.Coin() == 1 && ch.State.MyAmt-consts.MinOutput > msg.Amount && !ch.CloseData.Closed {
			qc = ch
			break
		}
	}

	if qc == nil {
		return fmt.Errorf("could not find suitable channel to route payment")
	}

	nd.RemoteMtx.Unlock()
	err := nd.OfferHTLC(qc, uint32(msg.Amount), msg.HHash, 100, [32]byte{})
	if err != nil {
		return err
	}

	msg.PeerIdx = sendToIdx
	nd.OmniOut <- msg
	return nil
}

func (nd *LitNode) MultihopPaymentSettleHandler(msg lnutil.MultihopPaymentSettleMsg) error {
	fmt.Printf("Received multihop payment settle from peer %d\n", msg.Peer())
	found := false
	inFlight := new(InFlightMultihop)

	for _, mh := range nd.InProgMultihop {
		hash := fastsha256.Sum256(msg.PreImage[:])
		if bytes.Equal(hash[:], mh.HHash[:]) {
			inFlight = mh
			found = true
		}
	}

	if !found {
		return fmt.Errorf("Unmatched settle message received")
	}

	// Forward
	var pkh [20]byte
	id := nd.IdKey().PubKey().SerializeCompressed()
	idHash := fastsha256.Sum256(id[:])
	copy(pkh[:], idHash[:20])
	var sendToPkh [20]byte
	for i, node := range inFlight.Path {
		if bytes.Equal(pkh[:], node[:]) {
			sendToPkh = inFlight.Path[i-1]
		}
	}

	sendToIdx, _ := nd.FindPeerIndexByAddress(bech32.Encode("ln", sendToPkh[:]))
	msg.PeerIdx = sendToIdx
	nd.OmniOut <- msg

	return nil
}
