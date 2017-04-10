package qln

import (
	"fmt"
	"log"

	"github.com/mit-dci/lit/lnutil"
)

// handles stuff that comes in over the wire.  Not user-initiated.
func (nd *LitNode) PeerHandler(msg lnutil.LitMsg, q *Qchan, peer *RemotePeer) error {
	switch msg.MsgType() & 0xf0 { // in progress
	case 0x00: // TEXT MESSAGE.  SIMPLE
		nd.UserMessageBox <- fmt.Sprintf(
			"\nmsg from %s: %s", lnutil.White(msg.Peer()), lnutil.Green(string(msg.Bytes())))
		return nil

	case 0x10:
		return nd.ChannelHandler(msg)

	case 0x20:
		return nd.CloseHandler(msg)

	case 0x30:
		if q == nil {
			return fmt.Errorf("pushpull message but no matching channel")
		}
		return nd.PushPullHandler(msg, q)

	/* not yet implemented
	case 0x40:
		return nd.FWDHandler(msg)
	*/
	/* not yet implemented
	case 0x50:
		return nd.SelfPush(msg)
	*/

	case 0x60:
		if !nd.Tower.Accepting {
			return fmt.Errorf("Error: Got tower msg from %x but tower disabled\n",
				msg.Peer())
		}
		return nd.Tower.HandleMessage(msg)

	default:
		return fmt.Errorf("Unknown message id byte %x &f0", msg.MsgType())

	}

}

// Every lndc has one of these running
// it listens for incoming messages on the lndc and hands it over
// to the OmniHandler via omnichan
func (nd *LitNode) LNDCReader(peer *RemotePeer) error {
	// this is a new peer connection; load all channels for this peer

	// no concurrency risk since we just got this map
	// have this as a separate func to drop extra channels from mem
	err := nd.PopulateQchanMap(peer)
	if err != nil {
		return err
	}
	var opArr [36]byte
	// make a local map of outpoints to channel indexes
	peer.OpMap = make(map[[36]byte]uint32)
	// inerate through all this peer's channels to extract outpoints
	for _, q := range peer.QCs {
		opArr = lnutil.OutPointToBytes(q.Op)
		peer.OpMap[opArr] = q.Idx()
	}

	for {
		msg := make([]byte, 65535)
		//	fmt.Printf("read message from %x\n", l.RemoteLNId)
		n, err := peer.Con.Read(msg)
		if err != nil {
			fmt.Printf("read error with %d: %s\n", peer.Idx, err.Error())
			nd.RemoteMtx.Lock()
			delete(nd.RemoteCons, peer.Idx)
			nd.RemoteMtx.Unlock()
			return peer.Con.Close()
		}
		msg = msg[:n]
		routedMsg := new(lnutil.LitMsg)
		// if message is long enough, try to set channel index of message
		if len(msg) > 38 {
			copy(opArr[:], msg[1:37])
			chanIdx, ok := peer.OpMap[opArr]
			if ok {
				routedMsg.ChanIdx = chanIdx
			}
		}
		routedMsg.PeerIdx = peer.Idx
		routedMsg.MsgType = msg[0]
		routedMsg.Data = msg[1:]
		if routedMsg.ChanIdx != 0 {
			err = nd.PeerHandler(routedMsg, peer.QCs[routedMsg.ChanIdx], peer)
		} else {
			err = nd.PeerHandler(routedMsg, nil, peer)
		}
		if err != nil {
			fmt.Printf("PeerHandler error with %d: %s\n", peer.Idx, err.Error())
		}
	}
}

func (nd *LitNode) PopulateQchanMap(peer *RemotePeer) error {
	allQs, err := nd.GetAllQchans()
	if err != nil {
		return err
	}
	// initialize map
	peer.QCs = make(map[uint32]*Qchan)
	// populate from all channels (inefficient)
	for i, q := range allQs {
		if q.Peer() == peer.Idx {
			peer.QCs[q.Idx()] = allQs[i]
		}
	}
	return nil
}

func (nd *LitNode) ChannelHandler(msg lnutil.LitMsg) error {
	switch msg.MsgType() {
	case lnutil.MSGID_POINTREQ: // POINT REQUEST
		fmt.Printf("Got point request from %x\n", msg.Peer())
		prm, ok := msg.(lnutil.PointReqMsg)
		if !ok {
			return fmt.Errorf("didn't work")
		}

		nd.PointReqHandler(prm)
		return nil

	case lnutil.MSGID_POINTRESP: // POINT RESPONSE
		fmt.Printf("Got point response from %x\n", msg.PeerIdx())
		err := nd.PointRespHandler(lnutil.PointRespMsg(msg))
		if err != nil {
			log.Printf(err.Error())
		}
		return nil

	case lnutil.MSGID_CHANDESC: // CHANNEL DESCRIPTION
		fmt.Printf("Got channel description from %x\n", msg.PeerIdx())
		nd.QChanDescHandler(lnutil.ChanDescMsg(msg))
		return nil

	case lnutil.MSGID_CHANACK: // CHANNEL ACKNOWLEDGE
		fmt.Printf("Got channel acknowledgement from %x\n", msg.PeerIdx())
		nd.QChanAckHandler(lnutil.ChanAckMsg(msg), peer)
		return nil

	case lnutil.MSGID_SIGPROOF: // HERE'S YOUR CHANNEL
		fmt.Printf("Got channel proof from %x\n", msg.PeerIdx())
		nd.SigProofHandler(lnutil.SigProofMsg(msg), peer)
		return nil

	default:
		return fmt.Errorf("Unknown message type %x", routedMsg.MsgType())
	}

}

func (nd *LitNode) CloseHandler(msg lnutil.LitMsg) error {
	switch msg.MsgType() { // CLOSE REQ

	case lnutil.MSGID_CLOSEREQ:
		fmt.Printf("Got close request from %x\n", msg.PeerIdx())
		nd.CloseReqHandler(lnutil.CloseReqMsg(msg))
		return nil

	/* - not yet implemented
	case lnutil.MSGID_CLOSERESP: // CLOSE RESP
		fmt.Printf("Got close response from %x\n", from)
		nd.CloseRespHandler(from, msg[1:])
		continue
		return nil
	*/
	default:
		return fmt.Errorf("Unknown message type %x", routedMsg.MsgType())
	}

}

// need a go routine for each qchan.

func (nd *LitNode) PushPullHandler(routedMsg lnutil.LitMsg, q *Qchan) error {
	switch routedMsg.MsgType() {
	case lnutil.MSGID_DELTASIG:
		fmt.Printf("Got DELTASIG from %x\n", routedMsg.PeerIdx())
		return nd.DeltaSigHandler(lnutil.DeltaSigMsg(routedMsg), q)

	case lnutil.MSGID_SIGREV: // SIGNATURE AND REVOCATION
		fmt.Printf("Got SIGREV from %x\n", routedMsg.PeerIdx())
		return nd.SigRevHandler(lnutil.SigRevMsg(routedMsg), q)

	case lnutil.MSGID_GAPSIGREV: // GAP SIGNATURE AND REVOCATION
		fmt.Printf("Got GapSigRev from %x\n", routedMsg.PeerIdx())
		return nd.GapSigRevHandler(lnutil.GapSigRevMsg(routedMsg), q)

	case lnutil.MSGID_REV: // REVOCATION
		fmt.Printf("Got REV from %x\n", routedMsg.PeerIdx())
		return nd.RevHandler(lnutil.RevMsg(routedMsg), q)

	default:
		return fmt.Errorf("Unknown message type %x", routedMsg.MsgType())

	}

}

func (nd *LitNode) FWDHandler(msg lnutil.LitMsg) error { // not yet implemented
	switch msg.MsgType() {
	default:
		return fmt.Errorf("Unknown message type %x", routedMsg.MsgType())
	}
}

func (nd *LitNode) SelfPush(msg lnutil.LitMsg) error { // not yet implemented
	switch msg.MsgType() {
	default:
		return fmt.Errorf("Unknown message type %x", routedMsg.MsgType())
	}
}

// OPEventHandler gets outpoint events from the base wallet,
// and modifies the ln node db to reflect confirmations.  Can also respond
// with exporting txos to the base wallet, or penalty txs.
func (nd *LitNode) OPEventHandler(OPEventChan chan lnutil.OutPointEvent) {
	for {
		curOPEvent := <-OPEventChan
		// get all channels each time.  This is very inefficient!
		qcs, err := nd.GetAllQchans()
		if err != nil {
			fmt.Printf("ln db error: %s", err.Error())
			continue
		}
		var theQ *Qchan
		for _, q := range qcs {
			if lnutil.OutPointsEqual(q.Op, curOPEvent.Op) {
				theQ = q
			}
		}
		// end if no associated channel
		if theQ == nil {
			fmt.Printf("OPEvent %s doesn't match any channel\n",
				curOPEvent.Op.String())
			continue
		}

		// confirmation event
		if curOPEvent.Tx == nil {
			fmt.Printf("OP %s Confirmation event\n", curOPEvent.Op.String())
			theQ.Height = curOPEvent.Height
			err = nd.SaveQchanUtxoData(theQ)
			if err != nil {
				fmt.Printf("SaveQchanUtxoData error: %s", err.Error())
				continue
			}
			// spend event (note: happens twice!)
		} else {
			fmt.Printf("OP %s Spend event\n", curOPEvent.Op.String())
			// mark channel as closed
			theQ.CloseData.Closed = true
			theQ.CloseData.CloseTxid = curOPEvent.Tx.TxHash()
			theQ.CloseData.CloseHeight = curOPEvent.Height
			err = nd.SaveQchanUtxoData(theQ)
			if err != nil {
				fmt.Printf("SaveQchanUtxoData error: %s", err.Error())
				continue
			}

			// detect close tx outs.
			txos, err := theQ.GetCloseTxos(curOPEvent.Tx)
			if err != nil {
				fmt.Printf("GetCloseTxos error: %s", err.Error())
				continue
			}
			// if you have seq=1 txos, modify the privkey...
			// pretty ugly as we need the private key to do that.
			for _, portxo := range txos {
				if portxo.Seq == 1 { // revoked key
					// GetCloseTxos returns a porTxo with the elk scalar in the
					// privkey field.  It isn't just added though; it needs to
					// be combined with the private key in a way porTxo isn't
					// aware of, so derive and subtract that here.
					var elkScalar [32]byte
					// swap out elkscalar, leaving privkey empty
					elkScalar, portxo.KeyGen.PrivKey = portxo.KeyGen.PrivKey, elkScalar
					privBase := nd.SubWallet.GetPriv(portxo.KeyGen)
					portxo.PrivKey = lnutil.CombinePrivKeyAndSubtract(
						privBase, elkScalar[:])
				}
				// make this concurrent to avoid circular locking
				go nd.SubWallet.ExportUtxo(&portxo)
			}
		}
	}
}
