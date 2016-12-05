package qln

import (
	"fmt"
	"log"
	"net"

	"github.com/mit-dci/lit/lnutil"
)

// handles stuff that comes in over the wire.  Not user-initiated.
func (nd *LnNode) OmniHandler() {
	var from [16]byte
	for {
		newdata := <-nd.OmniChan // blocks here
		if len(newdata) < 17 {
			fmt.Printf("got too short a message")
			continue
		}
		copy(from[:], newdata[:16])
		msg := newdata[16:]
		msgid := msg[0]

		// TEXT MESSAGE.  SIMPLE
		if msgid == MSGID_TEXTCHAT { //it's text
			fmt.Printf("text from %x: %s\n", from, msg[1:])
			continue
		}
		// POINT REQUEST
		if msgid == MSGID_POINTREQ {
			fmt.Printf("Got point request from %x\n", from)
			nd.PointReqHandler(from, msg[1:])
			continue
		}
		// POINT RESPONSE
		if msgid == MSGID_POINTRESP {
			fmt.Printf("Got point response from %x\n", from)
			err := nd.PointRespHandler(from, msg[1:])
			if err != nil {
				log.Printf(err.Error())
			}
			continue
		}
		// CHANNEL DESCRIPTION
		if msgid == MSGID_CHANDESC {
			fmt.Printf("Got channel description from %x\n", from)
			nd.QChanDescHandler(from, msg[1:])
			continue
		}
		// CHANNEL ACKNOWLEDGE
		if msgid == MSGID_CHANACK {
			fmt.Printf("Got channel acknowledgement from %x\n", from)
			nd.QChanAckHandler(from, msg[1:])
			continue
		}
		// HERE'S YOUR CHANNEL
		if msgid == MSGID_SIGPROOF {
			fmt.Printf("Got channel proof from %x\n", from)
			nd.SigProofHandler(from, msg[1:])
			continue
		}
		// CLOSE REQ
		if msgid == MSGID_CLOSEREQ {
			fmt.Printf("Got close request from %x\n", from)
			nd.CloseReqHandler(from, msg[1:])
			continue
		}
		// CLOSE RESP
		//		if msgid == uspv.MSGID_CLOSERESP {
		//			fmt.Printf("Got close response from %x\n", from)
		//			CloseRespHandler(from, msg[1:])
		//			continue
		//		}
		// REQUEST TO SEND
		if msgid == MSGID_RTS {
			fmt.Printf("Got RTS from %x\n", from)
			nd.RTSHandler(from, msg[1:])
			continue
		}
		// CHANNEL UPDATE ACKNOWLEDGE AND SIGNATURE
		if msgid == MSGID_ACKSIG {
			fmt.Printf("Got ACKSIG from %x\n", from)
			nd.ACKSIGHandler(from, msg[1:])
			continue
		}
		// SIGNATURE AND REVOCATION
		if msgid == MSGID_SIGREV {
			fmt.Printf("Got SIGREV from %x\n", from)
			nd.SIGREVHandler(from, msg[1:])
			continue
		}
		// REVOCATION
		if msgid == MSGID_REVOKE {
			fmt.Printf("Got REVOKE from %x\n", from)
			nd.REVHandler(from, msg[1:])
			continue
		}
		fmt.Printf("Unknown message id byte %x", msgid)
		continue
	}
}

// Every lndc has one of these running
// it listens for incoming messages on the lndc and hands it over
// to the OmniHandler via omnichan
func (nd *LnNode) LNDCReceiver(l net.Conn, id [16]byte) error {
	// first store peer in DB if not yet known
	_, err := nd.NewPeer(nd.RemoteCon.RemotePub)
	if err != nil {
		return err
	}
	for {
		msg := make([]byte, 65535)
		//	fmt.Printf("read message from %x\n", l.RemoteLNId)
		n, err := l.Read(msg)
		if err != nil {
			fmt.Printf("read error with %x: %s\n",
				id, err.Error())
			//			delete(CnMap, id)
			return l.Close()
		}
		msg = msg[:n]
		msg = append(id[:], msg...)
		//		fmt.Printf("incoming msg %x\n", msg)
		nd.OmniChan <- msg
	}
}

// OPEventHandler gets outpoint events from the base wallet,
// and modifies the ln node db to reflect confirmations.  Can also respond
// with exporting txos to the base wallet, or penalty txs.
func (nd *LnNode) OPEventHandler() {
	OPEventChan := nd.BaseWallet.LetMeKnow()
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
