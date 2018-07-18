package qln

import (
	"errors"
	"fmt"
	"log"

	"github.com/mit-dci/lit/wire"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/lnutil"
)

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
COLLISION: Set the collision flag (delta-(130))
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

// example message struct
type SigRevMsg struct {
	Op    wire.OutPoint
	Delta int32
	Sig   [64]byte
}

// example serialization method
func (m *SigRevMsg) Bytes() []byte {
	var b []byte

	// DeltaSig is op (36), Delta (4),  sig (64)
	// total length 104

	opbytes := lnutil.OutPointToBytes(m.Op)
	b = append(b, opbytes[:]...)
	b = append(b, lnutil.I32tB(m.Delta)...)
	b = append(b, m.Sig[:]...)

	return b
}

// example deserialization method
func SigRevFromBytes(b []byte) (*SigRevMsg, error) {
	if len(b) != 104 {
		return nil, fmt.Errorf("%d bytes, need 104", len(b))
	}

	m := new(SigRevMsg)

	var opArr [36]byte
	copy(opArr[:], b[:36])
	op := lnutil.OutPointFromBytes(opArr)
	m.Op = *op

	m.Delta = lnutil.BtI32(b[36:40])

	copy(m.Sig[:], b[40:])

	return m, nil

}

// SendNextMsg determines what message needs to be sent next
// based on the channel state.  It then calls the appropriate function.
func (nd *LitNode) ReSendMsg(qc *Qchan) error {

	// DeltaSig
	if qc.State.Delta < 0 {
		log.Printf("Sending previously sent DeltaSig\n")
		return nd.SendDeltaSig(qc)
	}

	// SigRev
	if qc.State.Delta > 0 {
		log.Printf("Sending previously sent SigRev\n")
		return nd.SendSigRev(qc)
	}

	// Rev
	return nd.SendREV(qc)
}

// PushChannel initiates a state update by sending a DeltaSig
func (nd *LitNode) PushChannel(qc *Qchan, amt uint32, data [32]byte) error {
	// sanity checks
	if amt >= 1<<30 {
		return fmt.Errorf("max send 1G sat (1073741823)")
	}
	if amt == 0 {
		return fmt.Errorf("have to send non-zero amount")
	}

	// see if channel is busy
	// lock this channel
	cts := false
	for !cts {
		qc.ChanMtx.Lock()
		select {
		case <-qc.ClearToSend:
			cts = true
		default:
			qc.ChanMtx.Unlock()
		}
	}
	// ClearToSend is now empty

	// reload from disk here, after unlock
	err := nd.ReloadQchanState(qc)
	if err != nil {
		// don't clear to send here; something is wrong with the channel
		qc.ChanMtx.Unlock()
		return err
	}

	// check that channel is confirmed, if non-test coin
	wal, ok := nd.SubWallet[qc.Coin()]
	if !ok {
		qc.ClearToSend <- true
		qc.ChanMtx.Unlock()
		return fmt.Errorf("Not connected to coin type %d\n", qc.Coin())
	}

	if !wal.Params().TestCoin && qc.Height < 100 {
		qc.ClearToSend <- true
		qc.ChanMtx.Unlock()
		return fmt.Errorf(
			"height %d; must wait min 1 conf for non-test coin\n", qc.Height)
	}

	// perform minOutput checks after reload
	myNewOutputSize := (qc.State.MyAmt - int64(amt)) - qc.State.Fee
	theirNewOutputSize := qc.Value - (qc.State.MyAmt - int64(amt)) - qc.State.Fee

	// check if this push would lower my balance below minBal
	if myNewOutputSize < consts.MinOutput {
		qc.ClearToSend <- true
		qc.ChanMtx.Unlock()
		return fmt.Errorf("want to push %s but %s available after %s fee and %s",
			lnutil.SatoshiColor(int64(amt)),
			lnutil.SatoshiColor(qc.State.MyAmt-qc.State.Fee-consts.MinOutput),
			lnutil.SatoshiColor(qc.State.Fee),
			lnutil.SatoshiColor(consts.MinOutput))
	}
	// check if this push is sufficient to get them above minBal
	if theirNewOutputSize < consts.MinOutput {
		qc.ClearToSend <- true
		qc.ChanMtx.Unlock()
		return fmt.Errorf(
			"pushing %s insufficient; counterparty bal %s fee %s consts.MinOutput %s",
			lnutil.SatoshiColor(int64(amt)),
			lnutil.SatoshiColor(qc.Value-qc.State.MyAmt),
			lnutil.SatoshiColor(qc.State.Fee),
			lnutil.SatoshiColor(consts.MinOutput))
	}

	// if we got here, but channel is not in rest state, try to fix it.
	if qc.State.Delta != 0 {
		err = nd.ReSendMsg(qc)
		if err != nil {
			qc.ClearToSend <- true
			qc.ChanMtx.Unlock()
			return err
		}
		qc.ChanMtx.Unlock()
		return fmt.Errorf("Didn't send.  Recovered though, so try again!")
	}

	qc.State.Data = data
	log.Printf("Sending message %x", data)

	qc.State.Delta = int32(-amt)

	if qc.State.Delta == 0 {
		qc.ChanMtx.Unlock()
		return errors.New("PushChannel: Delta cannot be zero")
	}

	// save to db with ONLY delta changed
	err = nd.SaveQchanState(qc)
	if err != nil {
		// don't clear to send here; something is wrong with the channel
		qc.ChanMtx.Unlock()
		return err
	}
	// move unlock to here so that delta is saved before

	log.Printf("PushChannel: Sending DeltaSig")

	err = nd.SendDeltaSig(qc)
	if err != nil {
		qc.ChanMtx.Unlock()
		// don't clear; something is wrong with the network
		return err
	}

	log.Printf("PushChannel: Done: sent DeltaSig")

	log.Printf("got pre CTS... \n")
	// block until clear to send is full again
	qc.ChanMtx.Unlock()

	cts = false
	for !cts {
		qc.ChanMtx.Lock()
		select {
		case <-qc.ClearToSend:
			cts = true
		default:
			qc.ChanMtx.Unlock()
		}
	}

	log.Printf("got post CTS... \n")
	// since we cleared with that statement, fill it again before returning
	qc.ClearToSend <- true
	qc.ChanMtx.Unlock()

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

	if q.State.Delta == 0 {
		return errors.New("Delta cannot be zero")
	}

	outMsg := lnutil.NewDeltaSigMsg(q.Peer(), q.Op, -q.State.Delta, sig, q.State.Data)

	log.Printf("Sending DeltaSig: %v", outMsg)

	nd.OmniOut <- outMsg

	return nil
}

// DeltaSigHandler takes in a DeltaSig and responds with a SigRev (normally)
// or a GapSigRev (if there's a collision)
// Leaves the channel either expecting a Rev (normally) or a GapSigRev (collision)
func (nd *LitNode) DeltaSigHandler(msg lnutil.DeltaSigMsg, qc *Qchan) error {
	log.Printf("Got DeltaSig: %v", msg)

	var collision bool
	//incomingDelta := uint32(math.Abs(float64(msg.Delta))) //int32 (may be negative, but should not be)
	incomingDelta := msg.Delta

	// we should be clear to send when we get a deltaSig
	select {
	case <-qc.ClearToSend:
	// keep going, normal
	default:
		// collision
		collision = true
	}

	log.Printf("COLLISION is (%t)\n", collision)

	// load state from disk
	err := nd.ReloadQchanState(qc)
	if err != nil {
		return fmt.Errorf("DeltaSigHandler ReloadQchan err %s", err.Error())
	}

	// TODO we should send a response that the channel is closed.
	// or offer to double spend with a cooperative close?
	// or update the remote node on closed channel status when connecting
	// TODO should disallow 'break' command when connected to the other node
	// or merge 'break' and 'close' UI so that it breaks when it can't
	// connect, and closes when it can.
	if qc.CloseData.Closed {
		return fmt.Errorf("DeltaSigHandler err: %d, %d is closed.",
			qc.Peer(), qc.Idx())
	}

	if collision {
		// incoming delta saved as collision value,
		// existing (negative) delta value retained.
		qc.State.Collision = int32(incomingDelta)
		log.Printf("delta sig COLLISION (%d)\n", qc.State.Collision)
	}

	// detect if channel is already locked, and lock if not
	//	nd.PushClearMutex.Lock()
	//	if nd.PushClear[qc.Idx()] == nil {
	//		nd.PushClear[qc.Idx()] = make(chan bool, 1)
	//	} else {
	// this means there was a collision
	// reload from disk; collision may have happened after disk read
	//		err := nd.ReloadQchan(qc)
	//		if err != nil {
	//			return fmt.Errorf("DeltaSigHandler err %s", err.Error())
	//		}

	//	}

	if qc.State.Delta > 0 {
		log.Printf(
			"DeltaSigHandler err: chan %d delta %d, expect rev, send empty rev",
			qc.Idx(), qc.State.Delta)

		return nd.SendREV(qc)
	}

	if !collision {
		// no collision, incoming (positive) delta saved.
		qc.State.Delta = int32(incomingDelta)
	}

	// they have to actually send you money
	if incomingDelta < 1 {
		return fmt.Errorf("DeltaSigHandler err: delta %d", incomingDelta)
	}

	// perform consts.MinOutput check
	theirNewOutputSize :=
		qc.Value - (qc.State.MyAmt + int64(incomingDelta)) - qc.State.Fee

	// check if this push is takes them below minimum output size
	if theirNewOutputSize < consts.MinOutput {
		qc.ClearToSend <- true
		return fmt.Errorf(
			"pushing %s reduces them too low; counterparty bal %s fee %s consts.MinOutput %s",
			lnutil.SatoshiColor(int64(incomingDelta)),
			lnutil.SatoshiColor(qc.Value-qc.State.MyAmt),
			lnutil.SatoshiColor(qc.State.Fee),
			lnutil.SatoshiColor(consts.MinOutput))
	}

	// update to the next state to verify
	qc.State.StateIdx++
	// regardless of collision, raise amt
	qc.State.MyAmt += int64(incomingDelta)

	log.Printf("Got message %x", msg.Data)
	qc.State.Data = msg.Data

	// verify sig for the next state. only save if this works
	err = qc.VerifySig(msg.Signature)
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

	// go up to n+2 elkpoint for the signing
	q.State.ElkPoint = q.State.N2ElkPoint
	// state is already incremented from DeltaSigHandler, increment *again* for n+1
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

	outMsg := lnutil.NewGapSigRev(q.KeyGen.Step[3]&0x7fffffff, q.Op, sig, *elk, n2ElkPoint)

	log.Printf("Sending GapSigRev: %v", outMsg)

	nd.OmniOut <- outMsg

	return nil
}

// SendSigRev sends a SigRev message based on channel info
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
	// q.State.NextElkPoint = q.State.N2ElkPoint // not needed
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

	outMsg := lnutil.NewSigRev(q.KeyGen.Step[3]&0x7fffffff, q.Op, sig, *elk, n2ElkPoint)

	log.Printf("Sending SigRev: %v", outMsg)

	nd.OmniOut <- outMsg
	return nil
}

// GapSigRevHandler takes in a GapSigRev, responds with a Rev, and
// leaves the channel in a state expecting a Rev.
func (nd *LitNode) GapSigRevHandler(msg lnutil.GapSigRevMsg, q *Qchan) error {
	log.Printf("Got GapSigRev: %v", msg)

	// load qchan & state from DB
	err := nd.ReloadQchanState(q)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
	}

	// check if we're supposed to get a GapSigRev now. Collision should be set
	if q.State.Collision == 0 {
		return fmt.Errorf(
			"chan %d got GapSigRev but collision = 0, delta = %d",
			q.Idx(), q.State.Delta)
	}

	// stash for justice tx
	prevAmt := q.State.MyAmt - int64(q.State.Collision) // myAmt before collision

	q.State.MyAmt += int64(q.State.Delta) // delta should be negative
	q.State.Delta = q.State.Collision     // now delta is positive
	q.State.Collision = 0

	// verify elkrem and save it in ram
	err = q.AdvanceElkrem(&msg.Elk, msg.N2ElkPoint)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
		// ! non-recoverable error, need to close the channel here.
	}

	// go up to n+1 elkpoint for the sig verification
	stashElkPoint := q.State.ElkPoint
	q.State.ElkPoint = q.State.NextElkPoint

	// state is already incremented from DeltaSigHandler, increment again for n+2
	// (note that we've moved n here.)
	q.State.StateIdx++

	// verify the sig
	err = q.VerifySig(msg.Signature)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
	}
	// go back to sequential elkpoints
	q.State.ElkPoint = stashElkPoint

	err = nd.SaveQchanState(q)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
	}
	err = nd.SendREV(q)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
	}

	// for justice, have to create signature for n-2.  Remember the n-2 amount

	q.State.StateIdx -= 2
	q.State.MyAmt = prevAmt

	err = nd.BuildJusticeSig(q)
	if err != nil {
		log.Printf("GapSigRevHandler BuildJusticeSig err %s", err.Error())
	}

	return nil
}

// SIGREVHandler takes in a SIGREV and responds with a REV (if everything goes OK)
// Leaves the channel in a clear / rest state.
func (nd *LitNode) SigRevHandler(msg lnutil.SigRevMsg, qc *Qchan) error {
	log.Printf("Got SigRev: %v", msg)

	// load qchan & state from DB
	err := nd.ReloadQchanState(qc)
	if err != nil {
		return fmt.Errorf("SIGREVHandler err %s", err.Error())
	}

	// check if we're supposed to get a SigRev now. Delta should be negative
	if qc.State.Delta > 0 {
		return fmt.Errorf("SIGREVHandler err: chan %d got SigRev, expect Rev. delta %d",
			qc.Idx(), qc.State.Delta)
	}

	if qc.State.Delta == 0 {
		// re-send last rev; they probably didn't get it
		return nd.SendREV(qc)
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
	err = qc.VerifySig(msg.Signature)
	if err != nil {
		return fmt.Errorf("SIGREVHandler err %s", err.Error())
	}

	// verify elkrem and save it in ram
	err = qc.AdvanceElkrem(&msg.Elk, msg.N2ElkPoint)
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

	log.Printf("SIGREV OK, state %d, will send REV\n", qc.State.StateIdx)
	err = nd.SendREV(qc)
	if err != nil {
		return fmt.Errorf("SIGREVHandler err %s", err.Error())
	}

	// now that we've saved & sent everything, before ending the function, we
	// go BACK to create a txid/sig pair for watchtower.  This feels like a kindof
	// weird way to do it.  Maybe there's a better way.

	qc.State.StateIdx--
	qc.State.MyAmt = prevAmt

	err = nd.BuildJusticeSig(qc)
	if err != nil {
		log.Printf("SigRevHandler BuildJusticeSig err %s", err.Error())
	}

	// done updating channel, no new messages expected.  Set clear to send
	qc.ClearToSend <- true

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

	outMsg := lnutil.NewRevMsg(q.Peer(), q.Op, *elk, n2ElkPoint)

	log.Printf("Sending Rev: %v", outMsg)

	nd.OmniOut <- outMsg

	return err
}

// REVHandler takes in a REV and clears the state's prev HAKD.  This is the
// final message in the state update process and there is no response.
// Leaves the channel in a clear / rest state.
func (nd *LitNode) RevHandler(msg lnutil.RevMsg, qc *Qchan) error {
	log.Printf("Got Rev: %v", msg)

	// load qchan & state from DB
	err := nd.ReloadQchanState(qc)
	if err != nil {
		return fmt.Errorf("REVHandler err %s", err.Error())
	}

	// check if there's nothing for them to revoke
	if qc.State.Delta == 0 {
		return fmt.Errorf("got REV, expected deltaSig, ignoring.")
	}
	// maybe this is an unexpected rev, asking us for a rev repeat
	if qc.State.Delta < 0 {
		log.Printf("got Rev, expected SigRev.  Re-sending last REV.\n")
		return nd.SendREV(qc)
	}

	// verify elkrem
	err = qc.AdvanceElkrem(&msg.Elk, msg.N2ElkPoint)
	if err != nil {
		log.Printf(" ! non-recoverable error, need to close the channel here.\n")
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
	err = nd.BuildJusticeSig(qc)
	if err != nil {
		log.Printf("RevHandler BuildJusticeSig err %s", err.Error())
	}

	// got rev, assert clear to send
	qc.ClearToSend <- true

	log.Printf("REV OK, state %d all clear.\n", qc.State.StateIdx)
	return nil
}
