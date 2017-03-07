package qln

import (
	"fmt"
	"log"

	"github.com/mit-dci/lit/lnutil"
)

// handles stuff that comes in over the wire.  Not user-initiated.
func (nd *LitNode) PeerHandler(msg *lnutil.LitMsg, q *Qchan, peer *RemotePeer) error {

	// TEXT MESSAGE.  SIMPLE
	if msg.MsgType == lnutil.MSGID_TEXTCHAT { //it's text
		nd.UserMessageBox <- fmt.Sprintf(
			"\nmsg from %s: %s", lnutil.White(msg.PeerIdx), lnutil.Green(string(msg.Data[:])))
		return nil
	}
	// POINT REQUEST
	if msg.MsgType == lnutil.MSGID_POINTREQ {
		fmt.Printf("Got point request from %x\n", msg.PeerIdx)
		nd.PointReqHandler(msg)
		return nil
	}
	// POINT RESPONSE
	if msg.MsgType == lnutil.MSGID_POINTRESP {
		fmt.Printf("Got point response from %x\n", msg.PeerIdx)
		err := nd.PointRespHandler(msg)
		if err != nil {
			log.Printf(err.Error())
		}
		return nil
	}
	// CHANNEL DESCRIPTION
	if msg.MsgType == lnutil.MSGID_CHANDESC {
		fmt.Printf("Got channel description from %x\n", msg.PeerIdx)
		nd.QChanDescHandler(msg)
		return nil
	}
	// CHANNEL ACKNOWLEDGE
	if msg.MsgType == lnutil.MSGID_CHANACK {
		fmt.Printf("Got channel acknowledgement from %x\n", msg.PeerIdx)
		nd.QChanAckHandler(msg, peer)
		return nil
	}
	// HERE'S YOUR CHANNEL
	if msg.MsgType == lnutil.MSGID_SIGPROOF {
		fmt.Printf("Got channel proof from %x\n", msg.PeerIdx)
		nd.SigProofHandler(msg, peer)
		return nil
	}
	// CLOSE REQ
	if msg.MsgType == lnutil.MSGID_CLOSEREQ {
		fmt.Printf("Got close request from %x\n", msg.PeerIdx)
		nd.CloseReqHandler(msg)
		return nil
	}
	// CLOSE RESP
	//		if msgid == uspv.MSGID_CLOSERESP {
	//			fmt.Printf("Got close response from %x\n", from)
	//			CloseRespHandler(from, msg[1:])
	//			continue
	//		}

	// PUSH type messages are 0x7?, and get their own helper function
	if msg.MsgType&0xf0 == 0x70 {
		if q == nil {
			return fmt.Errorf("pushpull message but no matching channel")
		}
		return nd.PushPullHandler(msg, q)
	}

	// messages to hand to the watchtower all start with 0xa_
	// don't strip the first byte before handing it over
	if msg.MsgType&0xf0 == 0xa0 {
		if !nd.Tower.Accepting {
			return fmt.Errorf("Error: Got tower msg from %x but tower disabled\n",
				msg.PeerIdx)
		}
		return nd.Tower.HandleMessage(msg)
	}

	return fmt.Errorf("Unknown message id byte %x &f0", msg.MsgType)
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

// need a go routine for each qchan.

func (nd *LitNode) PushPullHandler(routedMsg *lnutil.LitMsg, q *Qchan) error {

	if routedMsg.MsgType == lnutil.MSGID_DELTASIG {
		fmt.Printf("Got DELTASIG from %x\n", routedMsg.PeerIdx)
		return nd.DeltaSigHandler(routedMsg, q)
	}

	// SIGNATURE AND REVOCATION
	if routedMsg.MsgType == lnutil.MSGID_SIGREV {
		fmt.Printf("Got SIGREV from %x\n", routedMsg.PeerIdx)
		return nd.SigRevHandler(routedMsg, q)
	}

	// GAP SIGNATURE AND REVOCATION
	if routedMsg.MsgType == lnutil.MSGID_GAPSIGREV {
		fmt.Printf("Got GapSigRev from %x\n", routedMsg.PeerIdx)
		return nd.GapSigRevHandler(routedMsg, q)
	}

	// REVOCATION
	if routedMsg.MsgType == lnutil.MSGID_REV {
		fmt.Printf("Got REV from %x\n", routedMsg.PeerIdx)
		return nd.RevHandler(routedMsg, q)
	}

	return fmt.Errorf("Unknown message type %x", routedMsg.MsgType)
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
