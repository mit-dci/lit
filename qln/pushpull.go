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

pusher <- puller
SigRev: A signature and revocation of previous state

pusher -> puller
Rev: revocation

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

/*

2 options for dealing with push collision:
sequential and concurrent.
sequential has a deterministic priority which selects who to continue
the go-ahead node completes the push, then waits for the other node to push.

DeltaSig collision handling:

Send a DeltaSig.  Delta < 0.
Receive a DeltaSig with Delta < 0; need to send a GapSigRev
COLLISION: Set the collision flag (delta-(1<<30))
update amount with increment from received deltaSig
verify received signature & save to disk, update state number
*your delta value stays the same*
Send GapSigRev: revocation of previous state, and sig for next state
Receive GapSigRev
Clear collision flag
set delta = -delta (turns positive)
Update amount,  verity received signature & save to disk, update state number
Send Rev for previous state
Receive Rev for previous state


*/

// SendNextMsg determines what message needs to be sent next
// based on the channel state.  It then calls the appropriate function.
func (nd *LitNode) SendNextMsg(qc *Qchan) error {

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

// PushChannel initiates a state update by sending an DeltaSig
func (nd LitNode) PushChannel(qc *Qchan, amt uint32) error {
	// first, see if channel is busy, error if so, lock if not
	// lock this channel
	nd.PushClearMutex.Lock()
	if nd.PushClear[qc.Idx()] != nil {
		return fmt.Errorf("channel %d busy", qc.Idx())
	}
	nd.PushClear[qc.Idx()] = make(chan bool, 1)
	nd.PushClearMutex.Unlock()
	// reload from disk here, after unlock
	err := nd.ReloadQchan(qc)
	if err != nil {
		return err
	}

	// don't try to update state until all prior updates have cleared
	// may want to change this later, but requires other changes.
	if qc.State.Delta != 0 {
		return fmt.Errorf("channel update in progress, cannot push")
	}

	if amt == 0 {
		return fmt.Errorf("have to send non-zero amount")
	}

	if amt >= 1<<30 {
		return fmt.Errorf("max send 1G sat (1073741823)")
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
	err = nd.SaveQchanState(qc)
	if err != nil {
		return err
	}
	err = nd.SendDeltaSig(qc)
	if err != nil {
		return err
	}

	<-nd.PushClear[qc.Idx()]
	nd.PushClearMutex.Lock()
	delete(nd.PushClear, qc.Idx())
	nd.PushClearMutex.Unlock()

	return nil
}

// SendDeltaSig initiates a push, sending the amount to be pushed and the new sig.
func (nd *LitNode) SendDeltaSig(q *Qchan) error {
	// increment state number, update balance, go to next elkpoint
	q.State.StateIdx++
	q.State.MyAmt += int64(q.State.Delta)
	q.State.ElkPoint = q.State.NextElkPoint
	q.State.NextElkPoint = q.State.N2ElkPoint
	// N2Elk is now invalid

	// make the signature to send over
	sig, err := nd.SignState(q)
	if err != nil {
		return err
	}

	opArr := lnutil.OutPointToBytes(q.Op)

	var msg []byte

	// DeltaSig is op (36), Delta (4),  sig (64)
	// total length 104
	msg = append(msg, opArr[:]...)
	msg = append(msg, lnutil.I32tB(-q.State.Delta)...)
	msg = append(msg, sig[:]...)

	outMsg := new(lnutil.LitMsg)
	outMsg.MsgType = lnutil.MSGID_DELTASIG
	outMsg.PeerIdx = q.Peer()
	outMsg.Data = msg
	nd.OmniOut <- outMsg

	return err
}

// DeltaSigHandler takes in a DeltaSig and responds with an SigRev (normally)
// or a GapSigRev (if there's a collision)
// Leaves the channel either expecting a Rev (normally) or a GapSigRev (collision)
func (nd *LitNode) DeltaSigHandler(lm *lnutil.LitMsg) error {

	if len(lm.Data) < 104 || len(lm.Data) > 104 {
		return fmt.Errorf("got %d byte DeltaSig, expect 104", len(lm.Data))
	}

	var opArr [36]byte
	var incomingDelta uint32
	var incomingSig [64]byte
	// deserialize DeltaSig
	copy(opArr[:], lm.Data[:36])
	incomingDelta = lnutil.BtU32(lm.Data[36:40])
	copy(incomingSig[:], lm.Data[40:])

	// see if there is an actual channel
	// load qchan & state from DB
	qc, err := nd.GetQchan(opArr)
	if err != nil {
		return fmt.Errorf("DeltaSigHandler GetQchan err %s", err.Error())
	}

	if qc.CloseData.Closed {
		return fmt.Errorf("DeltaSigHandler err: %d, %d is closed.",
			qc.Peer(), qc.Idx())
	}

	// detect if channel is already locked, and lock if not
	nd.PushClearMutex.Lock()
	if nd.PushClear[qc.Idx()] == nil {
		nd.PushClear[qc.Idx()] = make(chan bool, 1)
	} else {
		// this means there was a collision
		// collision; incoming delta saved as collision value,
		// existing (negative) delta value retained.
		qc.State.Collision = int32(incomingDelta)
		fmt.Printf("delta sig COLLISION (%d)\n", qc.State.Collision)
	}
	nd.PushClearMutex.Unlock()

	if qc.State.Delta > 0 {
		return fmt.Errorf("DeltaSigHandler err: chan %d is delta %d, expect rev",
			qc.Idx(), qc.State.Delta)
	}

	if qc.State.Collision == 0 {
		// no collision, incoming (positive) delta saved.
		qc.State.Delta = int32(incomingDelta)
	}

	// they have to actually send you money
	if incomingDelta < 1 {
		return fmt.Errorf("DeltaSigHandler err: delta %d", incomingDelta)
	}

	// check if this push would lower counterparty balance below minBal
	if int64(incomingDelta) > (qc.Value-qc.State.MyAmt)+minBal {
		return fmt.Errorf("DeltaSigHandler err: delta %d but they have %d, minBal %d",
			incomingDelta, qc.Value-qc.State.MyAmt, minBal)
	}

	// update to the next state to verify
	qc.State.StateIdx++
	// regardless of collision, raise amt
	qc.State.MyAmt += int64(incomingDelta)

	// verify sig for the next state. only save if this works
	err = qc.VerifySig(incomingSig)
	if err != nil {
		return fmt.Errorf("DeltaSigHandler err %s", err.Error())
	}

	// (seems odd, but everything so far we still do in case of collision, so
	// only check here.  If it's a collision, set, save, send gapSigRev

	// save channel with new state, new sig, and positive delta set
	// and maybe collision; still haven't checked
	err = nd.SaveQchanState(qc)
	if err != nil {
		return fmt.Errorf("DeltaSigHandler SaveQchanState err %s", err.Error())
	}

	if qc.State.Collision != 0 {
		err = nd.SendGapSigRev(qc)
		if err != nil {
			return fmt.Errorf("DeltaSigHandler SendGapSigRev err %s", err.Error())
		}
	} else { // saved to db, now proceed to create & sign their tx
		err = nd.SendSigRev(qc)
		if err != nil {
			return fmt.Errorf("DeltaSigHandler SendSigRev err %s", err.Error())
		}
	}
	return nil
}

// SendGapSigRev is different; it signs for state+1 and revokes state-1
func (nd *LitNode) SendGapSigRev(q *Qchan) error {
	// state should already be set to the "gap" state; generate signature for n+1
	// the signature generation is similar to normal sigrev signing
	// in these "send_whatever" methods we don't modify and save to disk

	// state has been incremented in DeltaSigHandler so n is the gap state
	// revoke n-1
	elk, err := q.ElkSnd.AtIndex(q.State.StateIdx - 1)
	if err != nil {
		return err
	}

	// send elkpoint for n+2
	n2ElkPoint, err := q.N2ElkPointForThem()
	if err != nil {
		return err
	}

	// go up to n+1 elkpoint for the signing
	q.State.ElkPoint = q.State.N2ElkPoint
	// state is already incremented from DeltaSigHandler, increment again for n+1
	// (note that we've moved n here.)
	q.State.StateIdx++
	// amt is delta (negative) plus current amt (collision already added in)
	q.State.MyAmt += int64(q.State.Delta)

	// sign state n+1
	sig, err := nd.SignState(q)
	if err != nil {
		return err
	}

	// send
	// GapSigRev is op (36), sig (64), ElkHash (32), NextElkPoint (33)
	// total length 165
	opArr := lnutil.OutPointToBytes(q.Op)

	var msg []byte

	// SigRev is op (36), sig (64), ElkHash (32), NextElkPoint (33)
	// total length 165
	msg = append(msg, opArr[:]...)
	msg = append(msg, sig[:]...)
	msg = append(msg, elk[:]...)
	msg = append(msg, n2ElkPoint[:]...)

	outMsg := new(lnutil.LitMsg)
	outMsg.MsgType = lnutil.MSGID_GAPSIGREV
	outMsg.PeerIdx = q.KeyGen.Step[3] & 0x7fffffff
	outMsg.Data = msg
	nd.OmniOut <- outMsg

	return nil
}

// SendSigRev sends an SigRev message based on channel info
func (nd *LitNode) SendSigRev(q *Qchan) error {

	// revoke n-1
	elk, err := q.ElkSnd.AtIndex(q.State.StateIdx - 1)
	if err != nil {
		return err
	}

	// state number and balance has already been updated if the incoming sig worked.
	// go to next elkpoint for signing
	// note that we have to keep the old elkpoint on disk for when the rev comes in
	q.State.ElkPoint = q.State.NextElkPoint
	q.State.NextElkPoint = q.State.N2ElkPoint
	// n2elk invalid here

	sig, err := nd.SignState(q)
	if err != nil {
		return err
	}

	// send commitment elkrem point for next round of messages
	n2ElkPoint, err := q.N2ElkPointForThem()
	if err != nil {
		return err
	}

	opArr := lnutil.OutPointToBytes(q.Op)

	var msg []byte

	// SigRev is op (36), sig (64), ElkHash (32), NextElkPoint (33)
	// total length 165
	msg = append(msg, opArr[:]...)
	msg = append(msg, sig[:]...)
	msg = append(msg, elk[:]...)
	msg = append(msg, n2ElkPoint[:]...)

	outMsg := new(lnutil.LitMsg)
	outMsg.MsgType = lnutil.MSGID_SIGREV
	outMsg.PeerIdx = q.KeyGen.Step[3] & 0x7fffffff
	outMsg.Data = msg
	nd.OmniOut <- outMsg

	return nil
}

// GapSigRevHandler takes in a GapSigRev, responds with a Rev, and
// leaves the channel in a state expecting a Rev.
func (nd *LitNode) GapSigRevHandler(lm *lnutil.LitMsg) error {
	if len(lm.Data) < 165 || len(lm.Data) > 165 {
		return fmt.Errorf("got %d byte GAPSIGREV, expect 165", len(lm.Data))
	}

	var opArr [36]byte
	var sig [64]byte
	var n2elkPoint [33]byte
	// deserialize GapSigRev
	copy(opArr[:], lm.Data[:36])
	copy(sig[:], lm.Data[36:100])
	revElk, _ := chainhash.NewHash(lm.Data[100:132])
	copy(n2elkPoint[:], lm.Data[132:])

	// load qchan & state from DB
	q, err := nd.GetQchan(opArr)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
	}

	// check if we're supposed to get a GapSigRev now. Collision should be set
	if q.State.Collision == 0 {
		return fmt.Errorf(
			"chan %d got GapSigRev but collision = 0, delta = %d",
			q.Idx(), q.State.Delta)
	}

	// stash previous amount here for watchtower sig creation
	// will do this later
	//	prevAmt := qc.State.MyAmt
	q.State.MyAmt += int64(q.State.Delta)
	q.State.Delta = q.State.Collision
	q.State.Collision = 0

	// go up to n+2 elkpoint for the signing
	q.State.ElkPoint = q.State.N2ElkPoint

	// verify elkrem and save it in ram
	// Doesn't work.  Off by one...
	err = q.AdvanceElkrem(revElk, n2elkPoint)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
		// ! non-recoverable error, need to close the channel here.
	}

	// state is already incremented from DeltaSigHandler, increment again for n+2
	// (note that we've moved n here.)
	q.State.StateIdx++

	// verify the sig
	err = q.VerifySig(sig)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
	}

	err = nd.SaveQchanState(q)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
	}
	err = nd.SendREV(q)
	if err != nil {
		return fmt.Errorf("SIGREVHandler err %s", err.Error())
	}

	return nil
}

// SIGREVHandler takes in an SIGREV and responds with a REV (if everything goes OK)
// Leaves the channel in a clear / rest state.
func (nd *LitNode) SigRevHandler(lm *lnutil.LitMsg) error {

	if len(lm.Data) < 165 || len(lm.Data) > 165 {
		return fmt.Errorf("got %d byte SIGREV, expect 165", len(lm.Data))
	}

	var opArr [36]byte
	var sig [64]byte
	var n2elkPoint [33]byte
	// deserialize SIGREV
	copy(opArr[:], lm.Data[:36])
	copy(sig[:], lm.Data[36:100])
	revElk, _ := chainhash.NewHash(lm.Data[100:132])
	copy(n2elkPoint[:], lm.Data[132:])

	// load qchan & state from DB
	qc, err := nd.GetQchan(opArr)
	if err != nil {
		return fmt.Errorf("SIGREVHandler err %s", err.Error())
	}

	// check if we're supposed to get a SigRev now. Delta should be negative
	if qc.State.Delta >= 0 {
		return fmt.Errorf("SIGREVHandler err: chan %d unexpected SigRev, delta %d",
			qc.Idx(), qc.State.Delta)
	}

	if qc.State.Collision != 0 {
		return fmt.Errorf("chan %d got SigRev, expect GapSigRev delta %d col %d",
			qc.Idx(), qc.State.Delta, qc.State.Collision)
	}

	// stash previous amount here for watchtower sig creation
	prevAmt := qc.State.MyAmt

	qc.State.StateIdx++
	qc.State.MyAmt += int64(qc.State.Delta)
	qc.State.Delta = 0

	// first verify sig.
	// (if elkrem ingest fails later, at least we close out with a bit more money)
	err = qc.VerifySig(sig)
	if err != nil {
		return fmt.Errorf("SIGREVHandler err %s", err.Error())
	}

	// verify elkrem and save it in ram
	err = qc.AdvanceElkrem(revElk, n2elkPoint)
	if err != nil {
		return fmt.Errorf("SIGREVHandler err %s", err.Error())
		// ! non-recoverable error, need to close the channel here.
	}
	// if the elkrem failed but sig didn't... we should update the DB to reflect
	// that and try to close with the incremented amount, why not.
	// TODO Implement that later though.

	// all verified; Save finished state to DB, puller is pretty much done.
	err = nd.SaveQchanState(qc)
	if err != nil {
		return fmt.Errorf("SIGREVHandler err %s", err.Error())
	}

	fmt.Printf("SIGREV OK, state %d, will send REV\n", qc.State.StateIdx)
	err = nd.SendREV(qc)
	if err != nil {
		return fmt.Errorf("SIGREVHandler err %s", err.Error())
	}

	// now that we've saved & sent everything, before ending the function, we
	// go BACK to create a txid/sig pair for watchtower.  This feels like a kindof
	// weird way to do it.  Maybe there's a better way.

	qc.State.StateIdx--
	qc.State.MyAmt = prevAmt

	/*
		err = nd.BuildJusticeSig(qc)
		if err != nil {
			fmt.Printf("SIGREVHandler err %s", err.Error())
			return
		}
	*/

	// I'm done updating this channel
	nd.PushClear[qc.Idx()] <- true
	// that will delete this channel from the map

	return nil
}

// SendREV sends a REV message based on channel info
func (nd *LitNode) SendREV(q *Qchan) error {
	// revoke previous already built state
	elk, err := q.ElkSnd.AtIndex(q.State.StateIdx - 1)
	if err != nil {
		return err
	}
	// send commitment elkrem point for next round of messages
	n2ElkPoint, err := q.N2ElkPointForThem()
	if err != nil {
		return err
	}

	opArr := lnutil.OutPointToBytes(q.Op)

	var msg []byte
	// REV is op (36), elk hash (32), n2 elk point (33)
	// total length 101
	msg = append(msg, opArr[:]...)
	msg = append(msg, elk[:]...)
	msg = append(msg, n2ElkPoint[:]...)

	outMsg := new(lnutil.LitMsg)
	outMsg.MsgType = lnutil.MSGID_REV
	outMsg.PeerIdx = q.Peer()
	outMsg.Data = msg
	nd.OmniOut <- outMsg

	return err
}

// REVHandler takes in an REV and clears the state's prev HAKD.  This is the
// final message in the state update process and there is no response.
// Leaves the channel in a clear / rest state.
func (nd *LitNode) REVHandler(lm *lnutil.LitMsg) error {
	if len(lm.Data) != 101 {
		return fmt.Errorf("got %d byte REV, expect 101", len(lm.Data))
	}
	var opArr [36]byte
	var n2elkPoint [33]byte
	// deserialize SigRev
	copy(opArr[:], lm.Data[:36])
	revElk, _ := chainhash.NewHash(lm.Data[36:68])
	copy(n2elkPoint[:], lm.Data[68:])

	// load qchan & state from DB
	qc, err := nd.GetQchan(opArr)
	if err != nil {
		return fmt.Errorf("REVHandler err %s", err.Error())
	}

	// check if there's nothing for them to revoke
	if qc.State.Delta == 0 {
		return fmt.Errorf("got REV message with hash %s, but nothing to revoke\n",
			revElk.String())
	}

	// verify elkrem
	err = qc.AdvanceElkrem(revElk, n2elkPoint)
	if err != nil {
		fmt.Printf(" ! non-recoverable error, need to close the channel here.\n")
		return fmt.Errorf("REVHandler err %s", err.Error())
	}
	prevAmt := qc.State.MyAmt - int64(qc.State.Delta)
	qc.State.Delta = 0

	// save to DB (new elkrem & point, delta zeroed)
	err = nd.SaveQchanState(qc)
	if err != nil {
		return fmt.Errorf("REVHandler err %s", err.Error())
	}

	// after saving cleared updated state, go back to previous state and build
	// the justice signature
	qc.State.StateIdx--      // back one state
	qc.State.MyAmt = prevAmt // use stashed previous state amount

	/*
		err = nd.BuildJusticeSig(qc)
		if err != nil {
			fmt.Printf("REVHandler err %s", err.Error())
			return
		}
	*/

	// if there was a collision & we started a push, this should clear it
	nd.PushClear[qc.Idx()] <- true

	// if this was received only, this will clear it
	nd.PushClearMutex.Lock()
	delete(nd.PushClear, qc.Idx())
	nd.PushClearMutex.Unlock()

	fmt.Printf("REV OK, state %d all clear.\n", qc.State.StateIdx)
	return nil
}
