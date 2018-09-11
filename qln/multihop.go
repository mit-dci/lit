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

func (nd *LitNode) PayMultihop(dstLNAdr string, originCoinType uint32, destCoinType uint32, amount int64) (bool, error) {
	var targetAdr [20]byte
	_, adr, err := bech32.Decode(dstLNAdr)
	if err != nil {
		return false, err
	}

	wal, ok := nd.SubWallet[originCoinType]
	if !ok {
		return false, fmt.Errorf("not connected to cointype %d", originCoinType)
	}

	fee := wal.Fee() * 1000

	if amount < consts.MinOutput+fee {
		return false, fmt.Errorf("cannot send %d because it's less than minOutput + fee: %d", amount, consts.MinOutput+fee)
	}

	// Connect to the node
	if _, err := nd.FindPeerIndexByAddress(dstLNAdr); err != nil {
		err = nd.DialPeer(dstLNAdr)
		if err != nil {
			return false, fmt.Errorf("error connected to destination node for multihop: %s", err.Error())
		}
	}

	copy(targetAdr[:], adr)
	log.Printf("Finding route to %s", dstLNAdr)
	path, err := nd.FindPath(targetAdr, destCoinType, originCoinType, amount)
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
	nd.MultihopMutex.Lock()
	nd.InProgMultihop = append(nd.InProgMultihop, inFlight)
	nd.MultihopMutex.Unlock()

	log.Printf("Sending payment request to %s", dstLNAdr)
	msg := lnutil.NewMultihopPaymentRequestMsg(idx, destCoinType)
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
	inFlight.Path = []lnutil.RouteHop{{pkh, msg.Cointype}}

	rand.Read(inFlight.PreImage[:])
	hash := fastsha256.Sum256(inFlight.PreImage[:])

	inFlight.HHash = hash

	nd.MultihopMutex.Lock()
	nd.InProgMultihop = append(nd.InProgMultihop, inFlight)
	err := nd.SaveMultihopPayment(inFlight)
	if err != nil {
		nd.MultihopMutex.Unlock()
		return err
	}
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
			targetIdx, err := nd.FindPeerIndexByAddress(bech32.Encode("ln", targetNode.Node[:]))
			if err != nil {
				return fmt.Errorf("not connected to destination peer")
			}
			if msg.Peer() == targetIdx {
				fmt.Printf("Found the right pending multihop. Sending setup msg to first hop\n")
				// found the right one. Set this up
				firstHop := mh.Path[1]
				ourHop := mh.Path[0]
				firstHopIdx, err := nd.FindPeerIndexByAddress(bech32.Encode("ln", firstHop.Node[:]))
				if err != nil {
					return fmt.Errorf("not connected to first hop in route")
				}

				nd.RemoteMtx.Lock()
				var qc *Qchan
				for _, ch := range nd.RemoteCons[firstHopIdx].QCs {
					if ch.Coin() == ourHop.CoinType && ch.State.MyAmt-consts.MinOutput-ch.State.Fee >= mh.Amt && !ch.CloseData.Closed && !ch.State.Failed {
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
				err = nd.SaveMultihopPayment(nd.InProgMultihop[idx])
				if err != nil {
					return err
				}

				// Calculate what initial locktime we need
				wal, ok := nd.SubWallet[ourHop.CoinType]
				if !ok {
					return fmt.Errorf("not connected to wallet for cointype %d", ourHop.CoinType)
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

					// Set the dirty flag on each of the nodes' channels we used
					nd.ChannelMapMtx.Lock()
					for _, hop := range nd.InProgMultihop[idx].Path {
						for i, channel := range nd.ChannelMap[hop.Node] {
							if channel.Link.CoinType == hop.CoinType {
								nd.ChannelMap[hop.Node][i].Dirty = true
								break
							}
						}
					}
					nd.ChannelMapMtx.Unlock()

					var data [32]byte
					outMsg := lnutil.NewMultihopPaymentSetupMsg(firstHopIdx, msg.HHash, mh.Path, data)
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

	inFlight := new(InFlightMultihop)
	inFlight.Path = msg.NodeRoute
	inFlight.HHash = msg.HHash

	// Forward
	var pkh [20]byte
	id := nd.IdKey().PubKey().SerializeCompressed()
	idHash := fastsha256.Sum256(id[:])
	copy(pkh[:], idHash[:20])
	var nextHop, ourHop, incomingHop *lnutil.RouteHop
	for i, node := range inFlight.Path {
		if bytes.Equal(pkh[:], node.Node[:]) {
			if i == 0 {
				return fmt.Errorf("path is invalid")
			}
			if i+1 < len(inFlight.Path) {
				nextHop = &inFlight.Path[i+1]
			}
			ourHop = &inFlight.Path[i]
			incomingHop = &inFlight.Path[i-1]
			break
		}
	}

	// Check there is a corresponding incoming HTLC
	HTLCs, chans, err := nd.FindHTLCsByHash(msg.HHash)
	if err != nil {
		return fmt.Errorf("error finding HTLCs: %s", err.Error())
	}

	var found bool
	var prevHTLC *HTLC
	for idx, h := range HTLCs {
		if h.Incoming && !h.Cleared && !h.Clearing && !h.ClearedOnChain && chans[idx].Coin() == incomingHop.CoinType {
			found = true
			prevHTLC = &h
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

	var nullBytes [16]byte
	nd.MultihopMutex.Lock()
	defer nd.MultihopMutex.Unlock()
	for _, mh := range nd.InProgMultihop {
		hash := fastsha256.Sum256(mh.PreImage[:])

		if !bytes.Equal(mh.PreImage[:], nullBytes[:]) && bytes.Equal(msg.HHash[:], hash[:]) && mh.Path[len(mh.Path)-1].CoinType == incomingHop.CoinType {
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

	if nextHop == nil {
		return fmt.Errorf("route is invalid")
	}

	wal, ok := nd.SubWallet[incomingHop.CoinType]
	if !ok {
		return fmt.Errorf("not connected to wallet for cointype %d", incomingHop.CoinType)
	}

	height := wal.CurrentHeight()
	if prevHTLC.Locktime-consts.DefaultLockTime < uint32(height+consts.DefaultLockTime) {
		return fmt.Errorf("locktime of preceeding hop is too close for comfort: %d, height: %d", prevHTLC.Locktime-consts.DefaultLockTime, height)
	}

	wal, ok = nd.SubWallet[ourHop.CoinType]
	if !ok {
		return fmt.Errorf("not connected to wallet for cointype %d", ourHop.CoinType)
	}

	fee := wal.Fee() * 1000

	newLocktime := ((((prevHTLC.Locktime - uint32(height)) / consts.DefaultLockTime) - 1) * consts.DefaultLockTime) + uint32(wal.CurrentHeight())

	nd.InProgMultihop = append(nd.InProgMultihop, inFlight)

	err = nd.SaveMultihopPayment(inFlight)
	if err != nil {
		return err
	}

	lnAdr := bech32.Encode("ln", nextHop.Node[:])

	// Connect to the node
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

	amtRqd := prevHTLC.Amt

	// do we need to exchange?
	// is the last hop coin type the same as this one?
	if incomingHop.CoinType != ourHop.CoinType {
		// we need to exchange, but is it possible?
		var rd *lnutil.RateDesc
		var rates []lnutil.RateDesc
		nd.ChannelMapMtx.Lock()
		defer nd.ChannelMapMtx.Unlock()
		for _, link := range nd.ChannelMap[pkh] {
			if link.Link.CoinType == ourHop.CoinType {
				rates = link.Link.Rates
				break
			}
		}

		for _, rate := range rates {
			if rate.CoinType == incomingHop.CoinType && rate.Rate > 0 {
				rd = &rate
				break
			}
		}

		// it's not possible to exchange these two coin types
		if rd == nil {
			return fmt.Errorf("can't exchange %d for %d via us", incomingHop.CoinType, ourHop.CoinType)
		}

		// required capacity is last hop amt * rate
		if rd.Reciprocal {
			// prior hop coin type is worth less than this one
			amtRqd /= rd.Rate
		} else {
			// prior hop coin type is worth more than this one
			amtRqd *= rd.Rate
		}
	}

	if amtRqd < consts.MinOutput+fee {
		// exchanging to this point has pushed the amount too low
		return fmt.Errorf("exchanging %d for %d via us pushes the amount too low: %d", incomingHop.CoinType, ourHop.CoinType, amtRqd)
	}

	nd.RemoteMtx.Lock()
	var qc *Qchan
	for _, ch := range nd.RemoteCons[sendToIdx].QCs {
		if ch.Coin() == ourHop.CoinType && ch.State.MyAmt-consts.MinOutput-fee >= amtRqd && !ch.CloseData.Closed && !ch.State.Failed {
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
		err = nd.OfferHTLC(qc, uint32(amtRqd), msg.HHash, newLocktime, [32]byte{})
		if err != nil {
			log.Printf("error offering HTLC: %s", err.Error())
			return
		}

		// Set the dirty flag on the nodes' channels in this route so we
		// don't attempt to use them for routing before we get an link update
		nd.ChannelMapMtx.Lock()
		for _, hop := range msg.NodeRoute {
			if _, ok := nd.ChannelMap[hop.Node]; ok {
				for idx, channel := range nd.ChannelMap[hop.Node] {
					if channel.Link.CoinType == hop.CoinType {
						nd.ChannelMap[hop.Node][idx].Dirty = true
						break
					}
				}
			}
		}
		nd.ChannelMapMtx.Unlock()

		msg.PeerIdx = sendToIdx
		msg.NodeRoute = msg.NodeRoute[1:]
		nd.OmniOut <- msg
	}()

	return nil
}
