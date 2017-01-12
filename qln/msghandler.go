package qln

import (
	"fmt"
	"log"
	"net"

	"github.com/mit-dci/lit/lnutil"
)

// handles stuff that comes in over the wire.  Not user-initiated.
func (nd *LitNode) OmniHandler() {
	for {
		routedMsg := <-nd.OmniIn // blocks here

		// TEXT MESSAGE.  SIMPLE
		if routedMsg.MsgType == lnutil.MSGID_TEXTCHAT { //it's text
			nd.UserMessageBox <- fmt.Sprintf(
				"msg from %d: %s", routedMsg.PeerIdx, routedMsg.Data)
			continue
		}
		// POINT REQUEST
		if routedMsg.MsgType == lnutil.MSGID_POINTREQ {
			fmt.Printf("Got point request from %x\n", routedMsg.PeerIdx)
			nd.PointReqHandler(routedMsg)
			continue
		}
		// POINT RESPONSE
		if routedMsg.MsgType == lnutil.MSGID_POINTRESP {
			fmt.Printf("Got point response from %x\n", routedMsg.PeerIdx)
			err := nd.PointRespHandler(routedMsg)
			if err != nil {
				log.Printf(err.Error())
			}
			continue
		}
		// CHANNEL DESCRIPTION
		if routedMsg.MsgType == lnutil.MSGID_CHANDESC {
			fmt.Printf("Got channel description from %x\n", routedMsg.PeerIdx)
			nd.QChanDescHandler(routedMsg)
			continue
		}
		// CHANNEL ACKNOWLEDGE
		if routedMsg.MsgType == lnutil.MSGID_CHANACK {
			fmt.Printf("Got channel acknowledgement from %x\n", routedMsg.PeerIdx)
			nd.QChanAckHandler(routedMsg)
			continue
		}
		// HERE'S YOUR CHANNEL
		if routedMsg.MsgType == lnutil.MSGID_SIGPROOF {
			fmt.Printf("Got channel proof from %x\n", routedMsg.PeerIdx)
			nd.SigProofHandler(routedMsg)
			continue
		}
		// CLOSE REQ
		if routedMsg.MsgType == lnutil.MSGID_CLOSEREQ {
			fmt.Printf("Got close request from %x\n", routedMsg.PeerIdx)
			nd.CloseReqHandler(routedMsg)
			continue
		}
		// CLOSE RESP
		//		if msgid == uspv.MSGID_CLOSERESP {
		//			fmt.Printf("Got close response from %x\n", from)
		//			CloseRespHandler(from, msg[1:])
		//			continue
		//		}

		// PUSH
		// just put 'go' in front.  then it's concurrent.
		if routedMsg.MsgType == lnutil.MSGID_DELTASIG {
			fmt.Printf("Got DELTASIG from %x\n", routedMsg.PeerIdx)
			go func() {
				err := nd.DeltaSigHandler(routedMsg)
				if err != nil {
					fmt.Printf(err.Error())
				}
			}()
			continue
		}
		// SIGNATURE AND REVOCATION
		if routedMsg.MsgType == lnutil.MSGID_SIGREV {
			fmt.Printf("Got SIGREV from %x\n", routedMsg.PeerIdx)
			go func() {
				err := nd.SigRevHandler(routedMsg)
				if err != nil {
					fmt.Printf(err.Error())
				}
			}()
			continue
		}
		// REVOCATION
		if routedMsg.MsgType == lnutil.MSGID_REV {
			fmt.Printf("Got REV from %x\n", routedMsg.PeerIdx)
			// breaks if concurrent!  Maybe fix that for a little speedup.
			nd.REVHandler(routedMsg)
			continue
		}

		// messages to hand to the watchtower all start with 0xa_
		// don't strip the first byte before handing it over
		if routedMsg.MsgType&0xf0 == 0xa0 {
			if !nd.Tower.Accepting {
				fmt.Printf("Error: Got tower msg from %x but tower disabled\n",
					routedMsg.PeerIdx)
				continue
			}
			err := nd.Tower.HandleMessage(routedMsg)
			if err != nil {
				fmt.Printf(err.Error())
			}
			continue
		}

		fmt.Printf("Unknown message id byte %x &f0", routedMsg.MsgType)
		continue
	}
}

// Every lndc has one of these running
// it listens for incoming messages on the lndc and hands it over
// to the OmniHandler via omnichan
func (nd *LitNode) LNDCReader(l net.Conn, peerIdx uint32) error {
	for {
		msg := make([]byte, 65535)
		//	fmt.Printf("read message from %x\n", l.RemoteLNId)
		n, err := l.Read(msg)
		if err != nil {
			fmt.Printf("read error with %d: %s\n",
				peerIdx, err.Error())
			return l.Close()
		}
		msg = msg[:n]
		routedMsg := new(lnutil.LitMsg)
		routedMsg.PeerIdx = peerIdx
		routedMsg.MsgType = msg[0]
		routedMsg.Data = msg[1:]

		nd.OmniIn <- routedMsg
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
					privBase := nd.BaseWallet.GetPriv(portxo.KeyGen)
					portxo.PrivKey = lnutil.CombinePrivKeyAndSubtract(
						privBase, elkScalar[:])
				}
				// make this concurrent to avoid circular locking
				go nd.BaseWallet.ExportUtxo(&portxo)
			}
		}
	}
}
