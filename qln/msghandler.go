package qln

import (
	"fmt"

	"github.com/mit-dci/lit/lnutil"
)

// handles stuff that comes in over the wire.  Not user-initiated.
func (nd *LitNode) PeerHandler(msg lnutil.LitMsg, q *Qchan, peer *RemotePeer) error {
	switch msg.MsgType() & 0xf0 {
	case 0x00: // TEXT MESSAGE.  SIMPLE
		chat, ok := msg.(lnutil.ChatMsg)
		if !ok {
			return fmt.Errorf("can't cast to chat message")
		}
		nd.UserMessageBox <- fmt.Sprintf(
			"\nmsg from %s: %s", lnutil.White(msg.Peer()), lnutil.Green(chat.Text))
		return nil // no error

	case 0x10: //Making Channel, or using
		return nd.ChannelHandler(msg, peer)

	case 0x20: //Closing
		return nd.CloseHandler(msg)

	case 0x30: //PushPull
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
		return nd.SelfPushHandler(msg)
	*/

	case 0x60: //Tower Messages
		//if !nd.Tower.Accepting {
		//	return fmt.Errorf("Error: Got tower msg from %x but tower disabled\n",
		//		msg.Peer())
		//}
		if msg.MsgType() == lnutil.MSGID_WATCH_DESC {
			nd.Tower.NewChannel(msg.(lnutil.WatchDescMsg))
		}
		if msg.MsgType() == lnutil.MSGID_WATCH_STATEMSG {
			nd.Tower.UpdateChannel(msg.(lnutil.WatchStateMsg))
		}
		if msg.MsgType() == lnutil.MSGID_WATCH_DELETE {
			nd.Tower.DeleteChannel(msg.(lnutil.WatchDelMsg))
		}

	case 0x70: // Routing messages
		if msg.MsgType() == lnutil.MSGID_LINK_DESC {
			nd.LinkMsgHandler(msg.(lnutil.LinkMsg))
		}

	case 0x80: // Dual Funding messages
		return nd.DualFundingHandler(msg, peer)

	default:
		return fmt.Errorf("Unknown message id byte %x &f0", msg.MsgType())

	}
	return nil
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
	// iterate through all this peer's channels to extract outpoints
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

		fmt.Printf("decrypted message is %x\n", msg)

		var routedMsg lnutil.LitMsg
		routedMsg, err = lnutil.LitMsgFromBytes(msg, peer.Idx)
		if err != nil {
			fmt.Printf("decoding message error with %d: %s\n", peer.Idx, err.Error())
			return err
		}

		fmt.Printf("peerIdx is %d\n", routedMsg.Peer())
		fmt.Printf("routed bytes %x\n", routedMsg.Bytes())

		fmt.Printf("message type %x\n", routedMsg.MsgType())

		var chanIdx uint32
		chanIdx = 0
		if len(msg) > 38 {
			copy(opArr[:], msg[1:37])
			chanCheck, ok := peer.OpMap[opArr]
			if ok {
				chanIdx = chanCheck
			}
		}

		fmt.Printf("chanIdx is %x\n", chanIdx)

		if chanIdx != 0 {
			err = nd.PeerHandler(routedMsg, peer.QCs[chanIdx], peer)
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

func (nd *LitNode) ChannelHandler(msg lnutil.LitMsg, peer *RemotePeer) error {
	if nd.InProgDual.PeerIdx != 0 { // a dual funding is in progress
		nd.DualFundingHandler(msg, peer)
		return nil
	}

	switch message := msg.(type) {
	case lnutil.PointReqMsg: // POINT REQUEST
		fmt.Printf("Got point request from %x\n", message.Peer())
		nd.PointReqHandler(message)
		return nil

	case lnutil.PointRespMsg: // POINT RESPONSE
		fmt.Printf("Got point response from %x\n", msg.Peer())
		return nd.PointRespHandler(message)

	case lnutil.ChanDescMsg: // CHANNEL DESCRIPTION
		fmt.Printf("Got channel description from %x\n", msg.Peer())
		nd.QChanDescHandler(message)
		return nil

	case lnutil.ChanAckMsg: // CHANNEL ACKNOWLEDGE
		fmt.Printf("Got channel acknowledgement from %x\n", msg.Peer())

		nd.QChanAckHandler(message, peer)
		return nil

	case lnutil.SigProofMsg: // HERE'S YOUR CHANNEL
		fmt.Printf("Got channel proof from %x\n", msg.Peer())
		nd.SigProofHandler(message, peer)
		return nil

	default:
		return fmt.Errorf("Unknown message type %x", msg.MsgType())
	}

}

func (nd *LitNode) DualFundingHandler(msg lnutil.LitMsg, peer *RemotePeer) error {
	switch message := msg.(type) {
	case lnutil.DualFundingReqMsg: // DUAL FUNDING REQUEST
		fmt.Printf("Got dual funding request from %x\n", message.Peer())
		nd.DualFundingReqHandler(message)
		return nil

	case lnutil.DualFundingAcceptMsg: // DUAL FUNDING ACCEPT
		fmt.Printf("Got dual funding acceptance from %x\n", msg.Peer())
		nd.DualFundingAcceptHandler(message)
		return nil

	case lnutil.DualFundingDeclMsg: // DUAL FUNDING DECLINE
		fmt.Printf("Got dual funding decline from %x\n", msg.Peer())
		nd.DualFundingDeclHandler(message)
		return nil

	case lnutil.ChanDescMsg: // CHANNEL DESCRIPTION
		fmt.Printf("Got (dual funding) channel description from %x\n", msg.Peer())
		nd.DualFundChanDescHandler(message)
		return nil

	case lnutil.DualFundingChanAckMsg: // CHANNEL ACKNOWLEDGE
		fmt.Printf("Got (dual funding) channel acknowledgement from %x\n", msg.Peer())

		nd.DualFundChanAckHandler(message, peer)
		return nil

	case lnutil.SigProofMsg: // HERE'S YOUR CHANNEL
		fmt.Printf("Got (dual funding) channel proof from %x\n", msg.Peer())
		nd.DualFundSigProofHandler(message, peer)
		return nil

	default:
		return fmt.Errorf("Unknown message type %x", msg.MsgType())
	}

}

func (nd *LitNode) CloseHandler(msg lnutil.LitMsg) error {
	switch message := msg.(type) { // CLOSE REQ

	case lnutil.CloseReqMsg:
		fmt.Printf("Got close request from %x\n", msg.Peer())
		nd.CloseReqHandler(message)
		return nil

	/* - not yet implemented
	case lnutil.MSGID_CLOSERESP: // CLOSE RESP
		fmt.Printf("Got close response from %x\n", from)
		nd.CloseRespHandler(from, msg[1:])
		continue
		return nil
	*/
	default:
		return fmt.Errorf("Unknown message type %x", msg.MsgType())
	}

}

// need a go routine for each qchan.

func (nd *LitNode) PushPullHandler(routedMsg lnutil.LitMsg, q *Qchan) error {
	switch message := routedMsg.(type) {
	case lnutil.DeltaSigMsg:
		fmt.Printf("Got DELTASIG from %x\n", routedMsg.Peer())
		return nd.DeltaSigHandler(message, q)

	case lnutil.SigRevMsg: // SIGNATURE AND REVOCATION
		fmt.Printf("Got SIGREV from %x\n", routedMsg.Peer())
		return nd.SigRevHandler(message, q)

	case lnutil.GapSigRevMsg: // GAP SIGNATURE AND REVOCATION
		fmt.Printf("Got GapSigRev from %x\n", routedMsg.Peer())
		return nd.GapSigRevHandler(message, q)

	case lnutil.RevMsg: // REVOCATION
		fmt.Printf("Got REV from %x\n", routedMsg.Peer())
		return nd.RevHandler(message, q)

	default:
		return fmt.Errorf("Unknown message type %x", routedMsg.MsgType())

	}

}

func (nd *LitNode) FWDHandler(msg lnutil.LitMsg) error { // not yet implemented
	switch message := msg.(type) {
	default:
		return fmt.Errorf("Unknown message type %x", message.MsgType())
	}
}

func (nd *LitNode) SelfPushHandler(msg lnutil.LitMsg) error { // not yet implemented
	switch message := msg.(type) {
	default:
		return fmt.Errorf("Unknown message type %x", message.MsgType())
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
					elkScalar, portxo.KeyGen.PrivKey =
						portxo.KeyGen.PrivKey, elkScalar

					// TODO make sure this doesn't crash on nil wallet
					privBase := nd.SubWallet[theQ.Coin()].GetPriv(portxo.KeyGen)

					portxo.PrivKey = lnutil.CombinePrivKeyAndSubtract(
						privBase, elkScalar[:])
				}
				// make this concurrent to avoid circular locking
				go nd.SubWallet[theQ.Coin()].ExportUtxo(&portxo)
			}
		}
	}
}
