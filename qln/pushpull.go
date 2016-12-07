package qln

import (
	"fmt"

	"github.com/mit-dci/lit/lnutil"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const minBal = 10000 // channels have to have 10K sat in them; can make variable later.

// Grab the coins that are rightfully yours! Plus some more.
// For right now, spend all outputs from channel close.
//func Grab(args []string) error {
//	return SCon.GrabAll()
//}

/*

3 messages

pusher -> puller
DeltaSig: how much is being sent, and a signature for that state

puller -> pusher
SigRev: A signature and revocation of previous state

pusher -> puller
Rev: A revocation

Every revocation contains the elkrem hash being revoked, and the next elkpoint.

SendNextMsg logic:

Message to send: channel state (sanity check)

DeltaSig:
delta < 0
you must be pushing.

SigRev:
delta > 0
you must be pulling.

Rev:
delta == 0
you must be done.

(note that puller also sends a (useless) rev once they've received the rev and
have their delta set to 0)

Note that when there's nothing to send, it'll send a REV message,
revoking the previous state which has already been revoked.

We could distinguish by writing to the db that we've sent the REV message...
but that doesn't seem that useful because we don't know if they got it so
we might have to send it again anyway.
*/

// SendNextMsg determines what message needs to be sent next
// based on the channel state.  It then calls the appropriate function.
func (nd *LnNode) SendNextMsg(qc *Qchan) error {

	// DeltaSig
	if qc.State.Delta < 0 {
		return nd.SendDeltaSig(qc)
	}

	// SigRev
	if qc.State.Delta > 0 {
		return nd.SendSigRev(qc)
	}

	// Rev
	return nd.SendREV(qc)
}

// PushChannel initiates a state update by sending an RTS
func (nd LnNode) PushChannel(qc *Qchan, amt uint32) error {

	// don't try to update state until all prior updates have cleared
	// may want to change this later, but requires other changes.
	if qc.State.Delta != 0 {
		return fmt.Errorf("channel update in progress, cannot push")
	}
	// check if this push would lower my balance below minBal
	if int64(amt)+minBal > qc.State.MyAmt {
		return fmt.Errorf("want to push %d but %d available, %d minBal",
			amt, qc.State.MyAmt, minBal)
	}
	// check if this push is sufficient to get them above minBal (only needed at
	// state 1)
	// if qc.State.StateIdx < 2 && int64(amt)+(qc.Value-qc.State.MyAmt) < minBal {
	if int64(amt)+(qc.Value-qc.State.MyAmt) < minBal {
		return fmt.Errorf("pushing %d insufficient; counterparty minBal %d",
			amt, minBal)
	}

	qc.State.Delta = int32(-amt)
	// save to db with ONLY delta changed
	err := nd.SaveQchanState(qc)
	if err != nil {
		return err
	}
	return nd.SendDeltaSig(qc)
}

// SendDeltaSig initiates a push, sending the amount to be pushed and the new sig.
func (nd *LnNode) SendDeltaSig(q *Qchan) error {
	// increment state number, update balance, go to next elkpoint
	q.State.StateIdx++
	q.State.MyAmt += int64(q.State.Delta)
	q.State.ElkPoint = q.State.NextElkPoint

	// make the signature to send over
	sig, err := nd.SignState(q)
	if err != nil {
		return err
	}
	opArr := lnutil.OutPointToBytes(q.Op)
	// DeltaSig is op (36), Delta (4),  sig (64)
	// total length 104
	msg := []byte{MSGID_DELTASIG}
	msg = append(msg, opArr[:]...)
	msg = append(msg, lnutil.I32tB(-q.State.Delta)...)
	msg = append(msg, sig[:]...)
	_, err = nd.RemoteCon.Write(msg)
	return err
}

// DeltaSigHandler takes in a DeltaSig and responds with an SigRev (if everything goes OK)
func (nd *LnNode) DeltaSigHandler(from [16]byte, DeltaSigBytes []byte) {

	if len(DeltaSigBytes) < 104 || len(DeltaSigBytes) > 104 {
		fmt.Printf("got %d byte DeltaSig, expect 104", len(DeltaSigBytes))
		return
	}

	var opArr [36]byte
	var incomingDelta uint32
	var incomingSig [64]byte
	// deserialize DeltaSig
	copy(opArr[:], DeltaSigBytes[:36])
	incomingDelta = lnutil.BtU32(DeltaSigBytes[36:40])
	copy(incomingSig[:], DeltaSigBytes[40:])

	// find who we're talkikng to
	var peerArr [33]byte
	copy(peerArr[:], nd.RemoteCon.RemotePub.SerializeCompressed())
	// load qchan & state from DB
	qc, err := nd.GetQchan(peerArr, opArr)
	if err != nil {
		fmt.Printf("DeltaSigHandler GetQchan err %s", err.Error())
		return
	}
	if qc.CloseData.Closed {
		fmt.Printf("DeltaSigHandler err: %d, %d is closed.",
			qc.KeyGen.Step[3], qc.KeyGen.Step[4])
		return
	}
	if qc.State.Delta != 0 {
		fmt.Printf("DeltaSigHandler err: %d, %d is in progress, delta %d",
			qc.KeyGen.Step[3], qc.KeyGen.Step[4], qc.State.Delta)
	}

	// they have to actually send you money
	if incomingDelta < 1 {
		fmt.Printf("DeltaSigHandler err: delta %d", incomingDelta)
		return
	}

	// check if this push would lower counterparty balance below minBal
	if int64(incomingDelta) > (qc.Value-qc.State.MyAmt)+minBal {
		fmt.Printf("DeltaSigHandler err: RTS delta %d but they have %d, minBal %d",
			incomingDelta, qc.Value-qc.State.MyAmt, minBal)
		return
	}

	if peerArr != qc.PeerId {
		fmt.Printf("DeltaSigHandler err: peer %x trying to modify peer %x's channel\n",
			peerArr, qc.PeerId)
		return
	}

	// update to the next state to verify
	qc.State.Delta = int32(incomingDelta)
	qc.State.StateIdx++
	qc.State.MyAmt += int64(incomingDelta)

	// verify sig for the next state. only save if this works
	err = qc.VerifySig(incomingSig)
	if err != nil {
		fmt.Printf("DeltaSigHandler err %s", err.Error())
		return
	}

	// save channel with new state, new sig, and positive delta set
	err = nd.SaveQchanState(qc)
	if err != nil {
		fmt.Printf("DeltaSigHandler SaveQchanState err %s", err.Error())
		return
	}
	// saved to db, now proceed to create & sign their tx
	err = nd.SendSigRev(qc)
	if err != nil {
		fmt.Printf("DeltaSigHandler SendACKSIG err %s", err.Error())
		return
	}

}

// SendSigRev sends an SigRev message based on channel info
func (nd *LnNode) SendSigRev(q *Qchan) error {
	// state number and balance has already been updated if the incoming sig worked.
	// go to next elkpoint for signing
	q.State.ElkPoint = q.State.NextElkPoint

	sig, err := nd.SignState(q)
	if err != nil {
		return err
	}

	// revoke previous already built state
	elk, err := q.ElkSnd.AtIndex(q.State.StateIdx - 1)
	if err != nil {
		return err
	}
	// send commitment elkrem point for next round of messages
	NextElkPoint, err := q.NextElkPointForThem()
	if err != nil {
		return err
	}

	opArr := lnutil.OutPointToBytes(q.Op)
	// SigRev is op (36), sig (64), ElkHash (32), NextElkPoint (33)
	// total length 165
	msg := []byte{MSGID_SIGREV}
	msg = append(msg, opArr[:]...)
	msg = append(msg, sig[:]...)
	msg = append(msg, elk[:]...)
	msg = append(msg, NextElkPoint[:]...)

	_, err = nd.RemoteCon.Write(msg)
	return err
}

// SIGREVHandler takes in an SIGREV and responds with a REV (if everything goes OK)
func (nd *LnNode) SigRevHandler(from [16]byte, SigRevBytes []byte) {

	if len(SigRevBytes) < 165 || len(SigRevBytes) > 165 {
		fmt.Printf("got %d byte SIGREV, expect 165", len(SigRevBytes))
		return
	}

	var opArr [36]byte
	var sig [64]byte
	var nextElkPoint [33]byte
	// deserialize SIGREV
	copy(opArr[:], SigRevBytes[:36])
	copy(sig[:], SigRevBytes[36:100])
	revElk, _ := chainhash.NewHash(SigRevBytes[100:132])
	copy(nextElkPoint[:], SigRevBytes[132:])

	// find who we're talkikng to
	var peerArr [33]byte
	copy(peerArr[:], nd.RemoteCon.RemotePub.SerializeCompressed())
	// load qchan & state from DB
	qc, err := nd.GetQchan(peerArr, opArr)
	if err != nil {
		fmt.Printf("SIGREVHandler err %s", err.Error())
		return
	}
	if peerArr != qc.PeerId {
		fmt.Printf("SIGREVHandler err: peer %x trying to modify peer %x's channel\n",
			peerArr, qc.PeerId)
		fmt.Printf("This can't happen now, but joseph wants this check here ",
			"in case the code changes later and we forget.\n")
		return
	}

	// stash previous amount here for watchtower sig creation
	prevAmt := qc.State.MyAmt

	qc.State.StateIdx++
	qc.State.MyAmt += int64(qc.State.Delta)
	qc.State.Delta = 0
	// go to next elkpoint for sig verification.  If it doesn't work we'll crash
	// without overwriting the old elkpoint
	//	qc.State.ElkPoint = qc.State.NextElkPoint

	// first verify sig.
	// (if elkrem ingest fails later, at least we close out with a bit more money)
	err = qc.VerifySig(sig)
	if err != nil {
		fmt.Printf("SIGREVHandler err %s", err.Error())
		return
	}

	// verify elkrem and save it in ram
	err = qc.AdvanceElkrem(revElk, nextElkPoint)
	if err != nil {
		fmt.Printf("SIGREVHandler err %s", err.Error())
		fmt.Printf(" ! non-recoverable error, need to close the channel here.\n")
		return
	}
	// if the elkrem failed but sig didn't... we should update the DB to reflect
	// that and try to close with the incremented amount, why not.
	// TODO Implement that later though.

	// all verified; Save finished state to DB, puller is pretty much done.
	err = nd.SaveQchanState(qc)
	if err != nil {
		fmt.Printf("SIGREVHandler err %s", err.Error())
		return
	}

	fmt.Printf("SIGREV OK, state %d, will send REV\n", qc.State.StateIdx)
	err = nd.SendREV(qc)
	if err != nil {
		fmt.Printf("SIGREVHandler err %s", err.Error())
		return
	}

	// now that we've saved & sent everything, before ending the function, we
	// go BACK to create a txid/sig pair for watchtower.  This feels like a kindof
	// weird way to do it.  Maybe there's a better way.

	qc.State.StateIdx--
	qc.State.MyAmt = prevAmt

	err = nd.BuildWatchTxidSig(qc)
	if err != nil {
		fmt.Printf("SIGREVHandler err %s", err.Error())
		return
	}

	return
}

// SendREV sends a REV message based on channel info
func (nd *LnNode) SendREV(q *Qchan) error {
	// revoke previous already built state
	elk, err := q.ElkSnd.AtIndex(q.State.StateIdx - 1)
	if err != nil {
		return err
	}
	// send commitment elkrem point for next round of messages
	nextElkPoint, err := q.NextElkPointForThem()
	if err != nil {
		return err
	}

	opArr := lnutil.OutPointToBytes(q.Op)
	// REV is op (36), elk hash (32), next elk point (33)
	// total length 101
	msg := []byte{MSGID_REV}
	msg = append(msg, opArr[:]...)
	msg = append(msg, elk[:]...)
	msg = append(msg, nextElkPoint[:]...)

	_, err = nd.RemoteCon.Write(msg)
	return err
}

// REVHandler takes in an REV and clears the state's prev HAKD.  This is the
// final message in the state update process and there is no response.
func (nd *LnNode) REVHandler(from [16]byte, revBytes []byte) {
	if len(revBytes) != 101 {
		fmt.Printf("got %d byte REV, expect 101", len(revBytes))
		return
	}
	var opArr [36]byte
	var nextElkPoint [33]byte
	// deserialize SigRev
	copy(opArr[:], revBytes[:36])
	revElk, _ := chainhash.NewHash(revBytes[36:68])
	copy(nextElkPoint[:], revBytes[68:])

	// find who we're talkikng to
	var peerArr [33]byte
	copy(peerArr[:], nd.RemoteCon.RemotePub.SerializeCompressed())
	// load qchan & state from DB
	qc, err := nd.GetQchan(peerArr, opArr)
	if err != nil {
		fmt.Printf("REVHandler err %s", err.Error())
		return
	}
	if peerArr != qc.PeerId {
		fmt.Printf("REVHandler err: peer %x trying to modify peer %x's channel\n",
			peerArr, qc.PeerId)
		fmt.Printf("This can't happen now, but joseph wants this check here ",
			"in case the code changes later and we forget.\n")
		return
	}

	// check if there's nothing for them to revoke
	if qc.State.Delta == 0 {
		fmt.Printf("got REV message with hash %s, but nothing to revoke\n",
			revElk.String())
	}

	// verify elkrem
	err = qc.AdvanceElkrem(revElk, nextElkPoint)
	if err != nil {
		fmt.Printf("REVHandler err %s", err.Error())
		fmt.Printf(" ! non-recoverable error, need to close the channel here.\n")
		return
	}
	qc.State.Delta = 0

	// save to DB (new elkrem & point, delta zeroed)
	err = nd.SaveQchanState(qc)
	if err != nil {
		fmt.Printf("REVHandler err %s", err.Error())
		return
	}
	fmt.Printf("REV OK, state %d all clear.\n", qc.State.StateIdx)
	return
}
