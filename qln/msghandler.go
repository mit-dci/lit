package qln

import (
	"bytes"
	"fmt"

	"github.com/mit-dci/lit/logging"

	"github.com/mit-dci/lit/btcutil/txscript"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/wire"
)

func (nd *LitNode) registerHandlers() {

	mp := nd.PeerMan.GetMessageProcessor()
	hf := makeNeoOmniHandler(nd)

	// I used the following command to generate these calls below:
	// grep -E '^.MSGID_[A-Z_]+ += ' lnutil/msglib.go | awk '{ print $1 }' | while read m; do echo "mp.DefineMessage(lnutil.$m, makeNeoOmniParser(lnutil.$m), hf)" ; done

	mp.DefineMessage(lnutil.MSGID_TEXTCHAT, makeNeoOmniParser(lnutil.MSGID_TEXTCHAT), hf)
	mp.DefineMessage(lnutil.MSGID_POINTREQ, makeNeoOmniParser(lnutil.MSGID_POINTREQ), hf)
	mp.DefineMessage(lnutil.MSGID_POINTRESP, makeNeoOmniParser(lnutil.MSGID_POINTRESP), hf)
	mp.DefineMessage(lnutil.MSGID_CHANDESC, makeNeoOmniParser(lnutil.MSGID_CHANDESC), hf)
	mp.DefineMessage(lnutil.MSGID_CHANACK, makeNeoOmniParser(lnutil.MSGID_CHANACK), hf)
	mp.DefineMessage(lnutil.MSGID_SIGPROOF, makeNeoOmniParser(lnutil.MSGID_SIGPROOF), hf)
	mp.DefineMessage(lnutil.MSGID_CLOSEREQ, makeNeoOmniParser(lnutil.MSGID_CLOSEREQ), hf)
	mp.DefineMessage(lnutil.MSGID_CLOSERESP, makeNeoOmniParser(lnutil.MSGID_CLOSERESP), hf)
	mp.DefineMessage(lnutil.MSGID_DELTASIG, makeNeoOmniParser(lnutil.MSGID_DELTASIG), hf)
	mp.DefineMessage(lnutil.MSGID_SIGREV, makeNeoOmniParser(lnutil.MSGID_SIGREV), hf)
	mp.DefineMessage(lnutil.MSGID_GAPSIGREV, makeNeoOmniParser(lnutil.MSGID_GAPSIGREV), hf)
	mp.DefineMessage(lnutil.MSGID_REV, makeNeoOmniParser(lnutil.MSGID_REV), hf)
	mp.DefineMessage(lnutil.MSGID_HASHSIG, makeNeoOmniParser(lnutil.MSGID_HASHSIG), hf)
	mp.DefineMessage(lnutil.MSGID_PREIMAGESIG, makeNeoOmniParser(lnutil.MSGID_PREIMAGESIG), hf)
	mp.DefineMessage(lnutil.MSGID_FWDMSG, makeNeoOmniParser(lnutil.MSGID_FWDMSG), hf)
	mp.DefineMessage(lnutil.MSGID_FWDAUTHREQ, makeNeoOmniParser(lnutil.MSGID_FWDAUTHREQ), hf)
	mp.DefineMessage(lnutil.MSGID_SELFPUSH, makeNeoOmniParser(lnutil.MSGID_SELFPUSH), hf)
	mp.DefineMessage(lnutil.MSGID_WATCH_DESC, makeNeoOmniParser(lnutil.MSGID_WATCH_DESC), hf)
	mp.DefineMessage(lnutil.MSGID_WATCH_STATEMSG, makeNeoOmniParser(lnutil.MSGID_WATCH_STATEMSG), hf)
	mp.DefineMessage(lnutil.MSGID_WATCH_DELETE, makeNeoOmniParser(lnutil.MSGID_WATCH_DELETE), hf)
	mp.DefineMessage(lnutil.MSGID_LINK_DESC, makeNeoOmniParser(lnutil.MSGID_LINK_DESC), hf)
	mp.DefineMessage(lnutil.MSGID_DLC_OFFER, makeNeoOmniParser(lnutil.MSGID_DLC_OFFER), hf)
	mp.DefineMessage(lnutil.MSGID_DLC_ACCEPTOFFER, makeNeoOmniParser(lnutil.MSGID_DLC_ACCEPTOFFER), hf)
	mp.DefineMessage(lnutil.MSGID_DLC_DECLINEOFFER, makeNeoOmniParser(lnutil.MSGID_DLC_DECLINEOFFER), hf)
	mp.DefineMessage(lnutil.MSGID_DLC_CONTRACTACK, makeNeoOmniParser(lnutil.MSGID_DLC_CONTRACTACK), hf)
	mp.DefineMessage(lnutil.MSGID_DLC_CONTRACTFUNDINGSIGS, makeNeoOmniParser(lnutil.MSGID_DLC_CONTRACTFUNDINGSIGS), hf)
	mp.DefineMessage(lnutil.MSGID_DLC_SIGPROOF, makeNeoOmniParser(lnutil.MSGID_DLC_SIGPROOF), hf)
	mp.DefineMessage(lnutil.MSGID_DLC_NEGOTIATE, makeNeoOmniParser(lnutil.MSGID_DLC_NEGOTIATE), hf)
	mp.DefineMessage(lnutil.MSGID_DLC_ACCEPTNEGOTIATE, makeNeoOmniParser(lnutil.MSGID_DLC_ACCEPTNEGOTIATE), hf)
	mp.DefineMessage(lnutil.MSGID_DLC_DECLINENEGOTIATE, makeNeoOmniParser(lnutil.MSGID_DLC_DECLINENEGOTIATE), hf)
	mp.DefineMessage(lnutil.MSGID_DUALFUNDINGREQ, makeNeoOmniParser(lnutil.MSGID_DUALFUNDINGREQ), hf)
	mp.DefineMessage(lnutil.MSGID_DUALFUNDINGACCEPT, makeNeoOmniParser(lnutil.MSGID_DUALFUNDINGACCEPT), hf)
	mp.DefineMessage(lnutil.MSGID_DUALFUNDINGDECL, makeNeoOmniParser(lnutil.MSGID_DUALFUNDINGDECL), hf)
	mp.DefineMessage(lnutil.MSGID_DUALFUNDINGCHANACK, makeNeoOmniParser(lnutil.MSGID_DUALFUNDINGCHANACK), hf)
	mp.DefineMessage(lnutil.MSGID_REMOTE_RPCREQUEST, makeNeoOmniParser(lnutil.MSGID_REMOTE_RPCREQUEST), hf)
	mp.DefineMessage(lnutil.MSGID_REMOTE_RPCRESPONSE, makeNeoOmniParser(lnutil.MSGID_REMOTE_RPCRESPONSE), hf)
	mp.DefineMessage(lnutil.MSGID_PAY_REQ, makeNeoOmniParser(lnutil.MSGID_PAY_REQ), hf)
	mp.DefineMessage(lnutil.MSGID_PAY_ACK, makeNeoOmniParser(lnutil.MSGID_PAY_ACK), hf)
	mp.DefineMessage(lnutil.MSGID_PAY_SETUP, makeNeoOmniParser(lnutil.MSGID_PAY_SETUP), hf)

}

// handles stuff that comes in over the wire.  Not user-initiated.
func (nd *LitNode) PeerHandler(msg lnutil.LitMsg, q *Qchan, peer *RemotePeer) error {
	logging.Infof("Message from %d type %x", msg.Peer(), msg.MsgType())
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
		if msg.MsgType() == lnutil.MSGID_WATCH_DESC {
			return nd.Tower.NewChannel(msg.(lnutil.WatchDescMsg))
		}
		if msg.MsgType() == lnutil.MSGID_WATCH_STATEMSG {
			return nd.Tower.UpdateChannel(msg.(lnutil.WatchStateMsg))
		}
		if msg.MsgType() == lnutil.MSGID_WATCH_DELETE {
			return nd.Tower.DeleteChannel(msg.(lnutil.WatchDelMsg))
		}

	case 0x70: // Routing messages
		if msg.MsgType() == lnutil.MSGID_LINK_DESC {
			nd.LinkMsgHandler(msg.(lnutil.LinkMsg))
		}
		if msg.MsgType() == lnutil.MSGID_PAY_REQ {
			return nd.MultihopPaymentRequestHandler(msg.(lnutil.MultihopPaymentRequestMsg))
		}
		if msg.MsgType() == lnutil.MSGID_PAY_ACK {
			return nd.MultihopPaymentAckHandler(msg.(lnutil.MultihopPaymentAckMsg))
		}
		if msg.MsgType() == lnutil.MSGID_PAY_SETUP {
			return nd.MultihopPaymentSetupHandler(msg.(lnutil.MultihopPaymentSetupMsg))
		}

	case 0xA0: // Dual Funding messages
		return nd.DualFundingHandler(msg, peer)

	case 0x90: // Discreet log contract messages
		if msg.MsgType() == lnutil.MSGID_DLC_OFFER {
			nd.DlcOfferHandler(msg.(lnutil.DlcOfferMsg), peer)
		}
		if msg.MsgType() == lnutil.MSGID_DLC_ACCEPTOFFER {
			return nd.DlcAcceptHandler(msg.(lnutil.DlcOfferAcceptMsg), peer)
		}
		if msg.MsgType() == lnutil.MSGID_DLC_DECLINEOFFER {
			nd.DlcDeclineHandler(msg.(lnutil.DlcOfferDeclineMsg), peer)
		}
		if msg.MsgType() == lnutil.MSGID_DLC_CONTRACTACK {
			nd.DlcContractAckHandler(msg.(lnutil.DlcContractAckMsg), peer)
		}
		if msg.MsgType() == lnutil.MSGID_DLC_CONTRACTFUNDINGSIGS {
			nd.DlcFundingSigsHandler(
				msg.(lnutil.DlcContractFundingSigsMsg), peer)
		}
		if msg.MsgType() == lnutil.MSGID_DLC_SIGPROOF {
			nd.DlcSigProofHandler(msg.(lnutil.DlcContractSigProofMsg), peer)
		}
		if msg.MsgType() == lnutil.MSGID_DLC_NEGOTIATE {
			nd.DlcNegotiateContractHandler(msg.(lnutil.DlcContractNegotiateMsg), peer)
		}

		if msg.MsgType() == lnutil.MSGID_DLC_ACCEPTNEGOTIATE {
			nd.DlcAcceptNegotiateAck(msg.(lnutil.DlcContractAcceptNegotiateMsg), peer)
		}
		
		if msg.MsgType() == lnutil.MSGID_DLC_DECLINENEGOTIATE {
			nd.DlcDeclineNegotiateAck(msg.(lnutil.DlcContractDeclineNegotiateMsg), peer)
		}			

	case 0xB0: // remote control
		if msg.MsgType() == lnutil.MSGID_REMOTE_RPCREQUEST {
			nd.RemoteControlRequestHandler(msg.(lnutil.RemoteControlRpcRequestMsg), peer)
		}
		if msg.MsgType() == lnutil.MSGID_REMOTE_RPCRESPONSE {
			nd.RemoteControlResponseHandler(msg.(lnutil.RemoteControlRpcResponseMsg), peer)
		}
	default:
		return fmt.Errorf("Unknown message id byte %x &f0", msg.MsgType())

	}
	return nil
}

func (nd *LitNode) PopulateQchanMap(peer *RemotePeer) error {
	allQs, err := nd.GetAllQchans()
	if err != nil {
		return err
	}
	// initialize map
	nd.RemoteMtx.Lock()
	peer.QCs = make(map[uint32]*Qchan)
	// populate from all channels (inefficient)
	for i, q := range allQs {
		if q.Peer() == peer.Idx {
			peer.QCs[q.Idx()] = allQs[i]
		}
	}
	nd.RemoteMtx.Unlock()
	return nil
}

func (nd *LitNode) ChannelHandler(msg lnutil.LitMsg, peer *RemotePeer) error {
	if nd.InProgDual.PeerIdx != 0 { // a dual funding is in progress
		nd.DualFundingHandler(msg, peer)
		return nil
	}

	switch message := msg.(type) {
	case lnutil.PointReqMsg: // POINT REQUEST
		logging.Infof("Got point request from %x\n", message.Peer())
		nd.PointReqHandler(message)
		return nil

	case lnutil.PointRespMsg: // POINT RESPONSE
		logging.Infof("Got point response from %x\n", msg.Peer())
		return nd.PointRespHandler(message)

	case lnutil.ChanDescMsg: // CHANNEL DESCRIPTION
		logging.Infof("Got channel description from %x\n", msg.Peer())

		return nd.QChanDescHandler(message)

	case lnutil.ChanAckMsg: // CHANNEL ACKNOWLEDGE
		logging.Infof("Got channel acknowledgement from %x\n", msg.Peer())

		nd.QChanAckHandler(message, peer)
		return nil

	case lnutil.SigProofMsg: // HERE'S YOUR CHANNEL
		logging.Infof("Got channel proof from %x\n", msg.Peer())
		nd.SigProofHandler(message, peer)
		return nil

	default:
		return fmt.Errorf("Unknown message type %x", msg.MsgType())
	}

}

func (nd *LitNode) DualFundingHandler(msg lnutil.LitMsg, peer *RemotePeer) error {
	switch message := msg.(type) {
	case lnutil.DualFundingReqMsg: // DUAL FUNDING REQUEST
		logging.Infof("Got dual funding request from %x\n", message.Peer())
		nd.DualFundingReqHandler(message)
		return nil

	case lnutil.DualFundingAcceptMsg: // DUAL FUNDING ACCEPT
		logging.Infof("Got dual funding acceptance from %x\n", msg.Peer())
		nd.DualFundingAcceptHandler(message)
		return nil

	case lnutil.DualFundingDeclMsg: // DUAL FUNDING DECLINE
		logging.Infof("Got dual funding decline from %x\n", msg.Peer())
		nd.DualFundingDeclHandler(message)
		return nil

	case lnutil.ChanDescMsg: // CHANNEL DESCRIPTION
		logging.Infof("Got (dual funding) channel description from %x\n", msg.Peer())
		nd.DualFundChanDescHandler(message)
		return nil

	case lnutil.DualFundingChanAckMsg: // CHANNEL ACKNOWLEDGE
		logging.Infof("Got (dual funding) channel acknowledgement from %x\n", msg.Peer())

		nd.DualFundChanAckHandler(message, peer)
		return nil

	case lnutil.SigProofMsg: // HERE'S YOUR CHANNEL
		logging.Infof("Got (dual funding) channel proof from %x\n", msg.Peer())
		nd.DualFundSigProofHandler(message, peer)
		return nil

	default:
		return fmt.Errorf("Unknown message type %x", msg.MsgType())
	}

}

func (nd *LitNode) CloseHandler(msg lnutil.LitMsg) error {
	switch message := msg.(type) { // CLOSE REQ

	case lnutil.CloseReqMsg:
		logging.Infof("Got close request from %x\n", msg.Peer())
		nd.CloseReqHandler(message)
		return nil

	/* - not yet implemented
	case lnutil.MSGID_CLOSERESP: // CLOSE RESP
		logging.Infof("Got close response from %x\n", from)
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
	q.ChanMtx.Lock()
	defer q.ChanMtx.Unlock()
	switch message := routedMsg.(type) {
	case lnutil.DeltaSigMsg:
		logging.Infof("Got DELTASIG from %x\n", routedMsg.Peer())
		return nd.DeltaSigHandler(message, q)

	case lnutil.SigRevMsg: // SIGNATURE AND REVOCATION
		logging.Infof("Got SIGREV from %x\n", routedMsg.Peer())
		return nd.SigRevHandler(message, q)

	case lnutil.GapSigRevMsg: // GAP SIGNATURE AND REVOCATION
		logging.Infof("Got GapSigRev from %x\n", routedMsg.Peer())
		return nd.GapSigRevHandler(message, q)

	case lnutil.RevMsg: // REVOCATION
		logging.Infof("Got REV from %x\n", routedMsg.Peer())
		return nd.RevHandler(message, q)

	case lnutil.HashSigMsg: // Offer HTLC
		logging.Infof("Got HashSig from %d", routedMsg.Peer())
		return nd.HashSigHandler(message, q)

	case lnutil.PreimageSigMsg: // Clear HTLC
		logging.Infof("Got PreimageSig from %d", routedMsg.Peer())
		return nd.PreimageSigHandler(message, q)

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
			logging.Errorf("ln db error: %s", err.Error())
			continue
		}
		var theQ *Qchan
		for _, q := range qcs {
			if lnutil.OutPointsEqual(q.Op, curOPEvent.Op) {
				theQ = q
			}
		}

		var theC *lnutil.DlcContract

		if theQ == nil {
			// Check if this is a contract output
			contracts, err := nd.DlcManager.ListContracts()
			if err != nil {
				logging.Errorf("contract db error: %s\n", err.Error())
				continue
			}
			for _, c := range contracts {
				if lnutil.OutPointsEqual(c.FundingOutpoint, curOPEvent.Op) {
					theC = c
				}
			}
		}

		if theC != nil {
			err := nd.HandleContractOPEvent(theC, &curOPEvent)
			if err != nil {
				logging.Errorf("HandleContractOPEvent error: %s\n", err.Error())
			}
			continue
		}

		if theQ == nil && curOPEvent.Tx != nil {
			// Check if this is a HTLC output we're watching
			h, _, err := nd.GetHTLC(&curOPEvent.Op)
			if err != nil {
				logging.Errorf("Error Getting HTLC OPHash: %s\n", err.Error())
			}
			if h.Idx == 0 && h.Amt == 0 { // empty HTLC, so none found
				continue
			}

			logging.Infof("Got OP event for HTLC output %s [Incoming: %t]\n", curOPEvent.Op.String(), h.Incoming)
			// Check the witness stack for a preimage
			for _, txi := range curOPEvent.Tx.TxIn {

				var preimage [16]byte
				preimageFound := false
				if len(txi.Witness) == 5 && len(txi.Witness[0]) == 0 && len(txi.Witness[3]) == 16 {
					// Success transaction from their break TX, multisig. Preimage is fourth on the witness stack.
					copy(preimage[:], txi.Witness[3])
					preimageFound = true
				}
				if len(txi.Witness) == 3 && len(txi.Witness[1]) == 16 {
					// Success transaction from their break TX, multisig. Preimage is fourth on the witness stack.
					copy(preimage[:], txi.Witness[1])
					preimageFound = true
				}

				if preimageFound {
					logging.Infof("Found preimage [%x] in this TX, looking for HTLCs i have that are claimable with that\n", preimage)
					// try claiming it!
					nd.ClaimHTLC(preimage)
				}
			}

			continue
		}

		// end if no associated channel
		if theQ == nil {
			logging.Infof("OPEvent %s doesn't match any channel\n",
				curOPEvent.Op.String())
			continue
		}

		// confirmation event
		if curOPEvent.Tx == nil {
			logging.Infof("OP %s Confirmation event\n", curOPEvent.Op.String())
			theQ.Height = curOPEvent.Height
			err = nd.SaveQchanUtxoData(theQ)
			if err != nil {
				logging.Errorf("SaveQchanUtxoData error: %s", err.Error())
				continue
			}
			// spend event (note: happens twice!)

			if theQ.Height > 0 {
				logging.Debugf("Second time this is confirmed, send out real confirm event")

				// TODO: abstract important channel things into a channel manager type of thing
				peerIdx := theQ.Peer()
				peer := nd.PeerMan.GetPeerByIdx(int32(peerIdx))
				if peer == nil {
					logging.Errorf("Please use errors in peermanager rather than just returning could be nil or 0 or something else")
				} else {
					confirmEvent := ChannelStateUpdateEvent{
						Action:   "opconfirm",
						ChanIdx:  theQ.Idx(),
						State:    theQ.State,
						TheirPub: peer.GetPubkey(),
						CoinType: theQ.Coin(),
					}

					if succeed, err := nd.Events.Publish(confirmEvent); err != nil {
						logging.Errorf("ConfirmHandler publish err %s", err)
						return
					} else if !succeed {
						logging.Errorf("ConfirmHandler publish did not succeed")
						return
					}
				}
			}

		} else {
			logging.Infof("OP %s Spend event\n", curOPEvent.Op.String())
			// mark channel as closed
			theQ.CloseData.Closed = true
			theQ.CloseData.CloseTxid = curOPEvent.Tx.TxHash()
			theQ.CloseData.CloseHeight = curOPEvent.Height
			err = nd.SaveQchanUtxoData(theQ)
			if err != nil {
				logging.Errorf("SaveQchanUtxoData error: %s", err.Error())
				continue
			}

			// detect close tx outs.
			txos, err := theQ.GetCloseTxos(curOPEvent.Tx)
			if err != nil {
				logging.Errorf("GetCloseTxos error: %s", err.Error())
				continue
			}

			// if you have seq=1 txos, modify the privkey...
			// pretty ugly as we need the private key to do that.
			for _, ptxo := range txos {
				if ptxo.Seq == 1 { // revoked key
					// GetCloseTxos returns a porTxo with the elk scalar in the
					// privkey field.  It isn't just added though; it needs to
					// be combined with the private key in a way porTxo isn't
					// aware of, so derive and subtract that here.
					var elkScalar [32]byte
					// swap out elkscalar, leaving privkey empty
					elkScalar, ptxo.KeyGen.PrivKey =
						ptxo.KeyGen.PrivKey, elkScalar

					privBase, err := nd.SubWallet[theQ.Coin()].GetPriv(ptxo.KeyGen)
					if err != nil {
						continue // or return?
					}

					ptxo.PrivKey = lnutil.CombinePrivKeyAndSubtract(
						privBase, elkScalar[:])
				}
				// make this concurrent to avoid circular locking
				go func(porTxo portxo.PorTxo) {
					nd.SubWallet[theQ.Coin()].ExportUtxo(&porTxo)
				}(ptxo)
			}

			// Fetch the indexes of HTLC outputs, and then register them to be watched
			// We can monitor this for spends from an HTLC output that contains a preimage
			// and then use that preimage to claim any HTLCs we have outstanding.
			_, htlcIdxes, err := theQ.GetHtlcTxos(curOPEvent.Tx, false)
			if err != nil {
				logging.Errorf("GetHtlcTxos error: %s", err.Error())
				continue
			}
			_, htlcOurIdxes, err := theQ.GetHtlcTxos(curOPEvent.Tx, true)
			if err != nil {
				logging.Errorf("GetHtlcTxos error: %s", err.Error())
				continue
			}
			htlcIdxes = append(htlcIdxes, htlcOurIdxes...)
			txHash := curOPEvent.Tx.TxHash()
			for _, i := range htlcIdxes {
				op := wire.NewOutPoint(&txHash, i)
				logging.Infof("Watching for spends from [%s] (HTLC)\n", op.String())
				nd.SubWallet[theQ.Coin()].WatchThis(*op)
			}
		}
	}
}

func (nd *LitNode) HeightEventHandler(HeightEventChan chan lnutil.HeightEvent) {
	for {
		event := <-HeightEventChan
		txs, err := nd.ClaimHTLCTimeouts(event.CoinType, event.Height)
		if err != nil {
			logging.Errorf("Error while claiming HTLC timeouts for coin %d at height %d : %s\n", event.CoinType, event.Height, err.Error())
		} else {
			for _, tx := range txs {
				logging.Infof("Claimed timeout HTLC using TXID %x\n", tx)
			}
		}
	}
}

func (nd *LitNode) HandleContractOPEvent(c *lnutil.DlcContract,
	opEvent *lnutil.OutPointEvent) error {

	logging.Infof("Received OPEvent for contract %d!\n", c.Idx)
	if opEvent.Tx != nil {
		wal, ok := nd.SubWallet[c.CoinType]
		if !ok {
			return fmt.Errorf("Could not find associated wallet"+
				" for type %d", c.CoinType)
		}

		nd.OpEventTx = opEvent.Tx
		
		pkhIsMine := false
		pkhIdx := uint32(0)
		value := int64(0)
		myPKHPkSript := lnutil.DirectWPKHScriptFromPKH(c.OurPayoutPKH)
		for i, out := range opEvent.Tx.TxOut {

			if bytes.Equal(myPKHPkSript, out.PkScript) {
				pkhIdx = uint32(i)
				pkhIsMine = true
				value = out.Value

			}
		}

		if pkhIsMine {
			c.Status = lnutil.ContractStatusSettling
			err := nd.DlcManager.SaveContract(c)
			if err != nil {
				logging.Errorf("HandleContractOPEvent SaveContract err %s\n", err.Error())
				return err
			}

			// We need to claim this.
			txClaim := wire.NewMsgTx()
			txClaim.Version = 2

			settleOutpoint := wire.OutPoint{Hash: opEvent.Tx.TxHash(), Index: pkhIdx}
			txClaim.AddTxIn(wire.NewTxIn(&settleOutpoint, nil, nil))

			addr, err := wal.NewAdr()
			if err != nil {
				return err
			}

			// Here the transaction size is always the same
			// n := 8 + VarIntSerializeSize(uint64(len(msg.TxIn))) +
			// 	VarIntSerializeSize(uint64(len(msg.TxOut)))
			// n = 10
			// Plus Single input 41
			// Plus Single output 31
			// Plus 2 for all wittness transactions
			// Plus Witness Data 108

			// TxSize = 4 + 4 + 1 + 1 + 2 + 108 + 41 + 31 = 192
			// Vsize = ((192 - 108 - 2) * 3 + 192) / 4 = 109,5

			vsize := uint32(110)
			fee := vsize * c.FeePerByte			

			txClaim.AddTxOut(wire.NewTxOut(value-int64(fee), lnutil.DirectWPKHScriptFromPKH(addr))) 

			var kg portxo.KeyGen
			kg.Depth = 5
			kg.Step[0] = 44 | 1<<31
			kg.Step[1] = c.CoinType | 1<<31
			kg.Step[2] = UseContractPayoutPKH
			kg.Step[3] = c.PeerIdx | 1<<31
			kg.Step[4] = uint32(c.Idx) | 1<<31
			priv, _ := wal.GetPriv(kg)

			// make hash cache
			hCache := txscript.NewTxSigHashes(txClaim)

			// generate sig
			txClaim.TxIn[0].Witness, err = txscript.WitnessScript(txClaim, hCache, 0, value, myPKHPkSript, txscript.SigHashAll, priv, true)

			if err != nil {
				return err
			}
			wal.DirectSendTx(txClaim)

			c.Status = lnutil.ContractStatusClosed
			err = nd.DlcManager.SaveContract(c)
			if err != nil {
				return err
			}
		}

	}
	return nil
}
