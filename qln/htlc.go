package qln

import (
	"fmt"
	"log"

	"github.com/btcsuite/fastsha256"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

func (nd *LitNode) OfferHTLC(qc *Qchan, amt uint32, RHash [32]byte, locktime uint32, data [32]byte) error {
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

	value := qc.Value

	for _, h := range qc.State.HTLCs {
		value -= h.Amt
	}

	myNewOutputSize := qc.State.MyAmt - qc.State.Fee - int64(amt)
	theirNewOutputSize := value - myNewOutputSize - int64(amt)

	// check if this push would lower my balance below minBal
	if myNewOutputSize < consts.MinOutput {
		qc.ClearToSend <- true
		qc.ChanMtx.Unlock()
		return fmt.Errorf("want to push %s but %s available after %s fee and %s consts.MinOutput",
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
	if qc.State.Delta != 0 || qc.State.InProgHTLC != nil {
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

	qc.State.InProgHTLC = new(HTLC)
	qc.State.InProgHTLC.Idx = qc.State.HTLCIdx
	qc.State.InProgHTLC.Incoming = false
	qc.State.InProgHTLC.Amt = int64(amt)
	qc.State.InProgHTLC.RHash = RHash
	qc.State.InProgHTLC.Locktime = locktime
	qc.State.InProgHTLC.TheirHTLCBase = qc.State.NextHTLCBase

	qc.State.InProgHTLC.KeyGen.Depth = 5
	qc.State.InProgHTLC.KeyGen.Step[0] = 44 | 1<<31
	qc.State.InProgHTLC.KeyGen.Step[1] = qc.Coin() | 1<<31
	qc.State.InProgHTLC.KeyGen.Step[2] = UseHTLCBase
	qc.State.InProgHTLC.KeyGen.Step[3] = qc.State.HTLCIdx | 1<<31
	qc.State.InProgHTLC.KeyGen.Step[4] = qc.Idx() | 1<<31

	qc.State.InProgHTLC.MyHTLCBase, _ = nd.GetUsePub(qc.State.InProgHTLC.KeyGen,
		UseHTLCBase)

	// save to db with ONLY InProgHTLC changed
	err = nd.SaveQchanState(qc)
	if err != nil {
		// don't clear to send here; something is wrong with the channel
		qc.ChanMtx.Unlock()
		return err
	}

	log.Printf("OfferHTLC: Sending HashSig")

	err = nd.SendHashSig(qc)
	if err != nil {
		qc.ChanMtx.Unlock()
		return err
	}

	log.Printf("OfferHTLC: Done: sent HashSig")

	log.Printf("got pre CTS...")
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

	log.Printf("got post CTS...")
	// since we cleared with that statement, fill it again before returning
	qc.ClearToSend <- true
	qc.ChanMtx.Unlock()

	return nil
}

func (nd *LitNode) SendHashSig(q *Qchan) error {
	q.State.StateIdx++

	q.State.MyAmt -= int64(q.State.InProgHTLC.Amt)

	q.State.ElkPoint = q.State.NextElkPoint
	q.State.NextElkPoint = q.State.N2ElkPoint

	// make the signature to send over
	commitmentSig, HTLCSigs, err := nd.SignState(q)
	if err != nil {
		return err
	}

	q.State.NextHTLCBase = q.State.N2HTLCBase

	outMsg := lnutil.NewHashSigMsg(q.Peer(), q.Op, q.State.InProgHTLC.Amt, q.State.InProgHTLC.RHash, commitmentSig, HTLCSigs, q.State.Data)

	log.Printf("Sending HashSig with %d HTLC sigs", len(HTLCSigs))

	nd.OmniOut <- outMsg

	return nil
}

func (nd *LitNode) HashSigHandler(msg lnutil.HashSigMsg, qc *Qchan) error {
	log.Printf("Got HashSig: %v", msg)

	var collision bool

	// we should be clear to send when we get a hashSig
	select {
	case <-qc.ClearToSend:
	// keep going, normal
	default:
		// collision
		collision = true
	}

	fmt.Printf("COLLISION is (%t)\n", collision)

	// load state from disk
	err := nd.ReloadQchanState(qc)
	if err != nil {
		return fmt.Errorf("HashSigHandler ReloadQchan err %s", err.Error())
	}

	// TODO we should send a response that the channel is closed.
	// or offer to double spend with a cooperative close?
	// or update the remote node on closed channel status when connecting
	// TODO should disallow 'break' command when connected to the other node
	// or merge 'break' and 'close' UI so that it breaks when it can't
	// connect, and closes when it can.
	if qc.CloseData.Closed {
		return fmt.Errorf("HashSigHandler err: %d, %d is closed.",
			qc.Peer(), qc.Idx())
	}

	if collision {
		// TODO: handle collisions
	}

	if qc.State.Delta > 0 {
		fmt.Printf(
			"DeltaSigHandler err: chan %d delta %d, expect rev, send empty rev",
			qc.Idx(), qc.State.Delta)

		return nd.SendREV(qc)
	}

	if !collision {
		// TODO: handle non-collision
	}

	// they have to actually send you money
	if msg.Amt < consts.MinOutput {
		return fmt.Errorf("HashSigHandler err: HTLC amount %d less than minOutput", msg.Amt)
	}

	// perform consts.MinOutput check
	myNewOutputSize := qc.State.MyAmt - qc.State.Fee
	theirNewOutputSize := qc.Value - myNewOutputSize - int64(msg.Amt)

	for _, h := range qc.State.HTLCs {
		theirNewOutputSize -= h.Amt
	}

	// check if this push is takes them below minimum output size
	if theirNewOutputSize < consts.MinOutput {
		qc.ClearToSend <- true
		return fmt.Errorf(
			"pushing %s reduces them too low; counterparty bal %s fee %s consts.MinOutput %s",
			lnutil.SatoshiColor(int64(msg.Amt)),
			lnutil.SatoshiColor(qc.Value-qc.State.MyAmt),
			lnutil.SatoshiColor(qc.State.Fee),
			lnutil.SatoshiColor(consts.MinOutput))
	}

	// update to the next state to verify
	qc.State.StateIdx++

	qc.State.InProgHTLC = new(HTLC)
	qc.State.InProgHTLC.Idx = qc.State.HTLCIdx
	qc.State.InProgHTLC.Incoming = true
	qc.State.InProgHTLC.Amt = int64(msg.Amt)
	qc.State.InProgHTLC.RHash = msg.RHash

	// TODO: make this customisable?
	qc.State.InProgHTLC.Locktime = consts.DefaultLockTime
	qc.State.InProgHTLC.TheirHTLCBase = qc.State.NextHTLCBase

	qc.State.InProgHTLC.KeyGen.Depth = 5
	qc.State.InProgHTLC.KeyGen.Step[0] = 44 | 1<<31
	qc.State.InProgHTLC.KeyGen.Step[1] = qc.Coin() | 1<<31
	qc.State.InProgHTLC.KeyGen.Step[2] = UseHTLCBase
	qc.State.InProgHTLC.KeyGen.Step[3] = qc.State.HTLCIdx | 1<<31
	qc.State.InProgHTLC.KeyGen.Step[4] = qc.Idx() | 1<<31

	qc.State.InProgHTLC.MyHTLCBase, _ = nd.GetUsePub(qc.State.InProgHTLC.KeyGen,
		UseHTLCBase)

	fmt.Printf("Got message %x", msg.Data)
	qc.State.Data = msg.Data

	// verify sig for the next state. only save if this works

	// TODO: There are more signatures required
	err = qc.VerifySigs(msg.CommitmentSignature, msg.HTLCSigs)
	if err != nil {
		return fmt.Errorf("HashSigHandler err %s", err.Error())
	}

	// (seems odd, but everything so far we still do in case of collision, so
	// only check here.  If it's a collision, set, save, send gapSigRev

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = qc.Coin() | 1<<31
	kg.Step[2] = UseHTLCBase
	kg.Step[3] = qc.State.HTLCIdx + 2 | 1<<31
	kg.Step[4] = qc.Idx() | 1<<31

	qc.State.MyNextHTLCBase = qc.State.MyN2HTLCBase

	qc.State.MyN2HTLCBase, err = nd.GetUsePub(kg,
		UseHTLCBase)
	if err != nil {
		return err
	}

	// save channel with new state, new sig, and positive delta set
	// and maybe collision; still haven't checked
	err = nd.SaveQchanState(qc)
	if err != nil {
		return fmt.Errorf("HashSigHandler SaveQchanState err %s", err.Error())
	}

	if qc.State.Collision != 0 {
		err = nd.SendGapSigRev(qc)
		if err != nil {
			return fmt.Errorf("HashSigHandler SendGapSigRev err %s", err.Error())
		}
	} else { // saved to db, now proceed to create & sign their tx
		err = nd.SendSigRev(qc)
		if err != nil {
			return fmt.Errorf("HashSigHandler SendSigRev err %s", err.Error())
		}
	}
	return nil
}

func (nd *LitNode) ClearHTLC(qc *Qchan, R [16]byte, HTLCIdx uint32) error {
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

	if int(HTLCIdx) >= len(qc.State.HTLCs) {
		return fmt.Errorf("HTLC idx %d out of range", HTLCIdx)
	}

	if qc.State.HTLCs[HTLCIdx].Cleared {
		return fmt.Errorf("HTLC %d already cleared", HTLCIdx)
	}

	var timeout bool
	if R == [16]byte{} {
		if int32(qc.State.HTLCs[HTLCIdx].Locktime) < wal.CurrentHeight() {
			return fmt.Errorf("Cannot timeout HTLC because locktime %d has not expired. Height: %d", qc.State.HTLCs[HTLCIdx].Locktime, wal.CurrentHeight())
		}

		timeout = true
	}

	if !timeout {
		RHash := fastsha256.Sum256(R[:])
		if qc.State.HTLCs[HTLCIdx].RHash != RHash {
			return fmt.Errorf("Preimage does not hash to expected value. Expected %x got %x", qc.State.HTLCs[HTLCIdx].RHash, RHash)
		}
	}

	// if we got here, but channel is not in rest state, try to fix it.
	if qc.State.Delta != 0 || qc.State.InProgHTLC != nil {
		err = nd.ReSendMsg(qc)
		if err != nil {
			qc.ClearToSend <- true
			qc.ChanMtx.Unlock()
			return err
		}
		qc.ChanMtx.Unlock()
		return fmt.Errorf("Didn't send.  Recovered though, so try again!")
	}

	qc.State.HTLCs[HTLCIdx].Clearing = true
	qc.State.HTLCs[HTLCIdx].R = R

	// save to db with ONLY Clearing & R changed
	err = nd.SaveQchanState(qc)
	if err != nil {
		// don't clear to send here; something is wrong with the channel
		qc.ChanMtx.Unlock()
		return err
	}

	log.Printf("ClearHTLC: Sending PreimageSig")

	err = nd.SendPreimageSig(qc, HTLCIdx)
	if err != nil {
		qc.ChanMtx.Unlock()
		return err
	}

	log.Printf("ClearHTLC: Done: sent PreimageSig")

	log.Printf("got pre CTS...")
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

	log.Printf("got post CTS...")
	// since we cleared with that statement, fill it again before returning
	qc.ClearToSend <- true
	qc.ChanMtx.Unlock()

	return nil
}

func (nd *LitNode) SendPreimageSig(q *Qchan, Idx uint32) error {
	q.State.StateIdx++

	if q.State.HTLCs[Idx].Incoming != (q.State.HTLCs[Idx].R == [16]byte{}) {
		q.State.MyAmt += q.State.HTLCs[Idx].Amt
	}

	q.State.ElkPoint = q.State.NextElkPoint
	q.State.NextElkPoint = q.State.N2ElkPoint

	// make the signature to send over
	commitmentSig, HTLCSigs, err := nd.SignState(q)
	if err != nil {
		return err
	}

	outMsg := lnutil.NewPreimageSigMsg(q.Peer(), q.Op, Idx, q.State.HTLCs[Idx].R, commitmentSig, HTLCSigs, q.State.Data)

	log.Printf("Sending PreimageSig with %d HTLC sigs", len(HTLCSigs))

	nd.OmniOut <- outMsg

	return nil
}

func (nd *LitNode) PreimageSigHandler(msg lnutil.PreimageSigMsg, qc *Qchan) error {
	log.Printf("Got PreimageSig: %v", msg)

	var collision bool

	// we should be clear to send when we get a preimageSig
	select {
	case <-qc.ClearToSend:
	// keep going, normal
	default:
		// collision
		collision = true
	}

	fmt.Printf("COLLISION is (%t)\n", collision)

	// load state from disk
	err := nd.ReloadQchanState(qc)
	if err != nil {
		return fmt.Errorf("PreimageSigHandler ReloadQchan err %s", err.Error())
	}

	wal, ok := nd.SubWallet[qc.Coin()]
	if !ok {
		return fmt.Errorf("Not connected to coin type %d\n", qc.Coin())
	}

	// TODO we should send a response that the channel is closed.
	// or offer to double spend with a cooperative close?
	// or update the remote node on closed channel status when connecting
	// TODO should disallow 'break' command when connected to the other node
	// or merge 'break' and 'close' UI so that it breaks when it can't
	// connect, and closes when it can.
	if qc.CloseData.Closed {
		return fmt.Errorf("PreimageSigHandler err: %d, %d is closed.",
			qc.Peer(), qc.Idx())
	}

	if collision {
		// TODO: handle collisions
	}

	if qc.State.Delta > 0 {
		fmt.Printf(
			"PreimageSigHandler err: chan %d delta %d, expect rev, send empty rev",
			qc.Idx(), qc.State.Delta)

		return nd.SendREV(qc)
	}

	if !collision {
		// TODO: handle non-collision
	}

	if int(msg.Idx) >= len(qc.State.HTLCs) {
		return fmt.Errorf("HTLC Idx %d out of range", msg.Idx)
	}

	if qc.State.HTLCs[msg.Idx].Cleared {
		return fmt.Errorf("HTLC %d already cleared", msg.Idx)
	}

	var timeout bool
	if msg.R == [16]byte{} {
		if int32(qc.State.HTLCs[msg.Idx].Locktime) < wal.CurrentHeight() {
			return fmt.Errorf("Cannot timeout HTLC because locktime %d has not expired. Height: %d", qc.State.HTLCs[msg.Idx].Locktime, wal.CurrentHeight())
		}

		timeout = true
	}

	if !timeout {
		RHash := fastsha256.Sum256(msg.R[:])
		if qc.State.HTLCs[msg.Idx].RHash != RHash {
			return fmt.Errorf("Preimage does not hash to expected value. Expected %x got %x", qc.State.HTLCs[msg.Idx].RHash, RHash)
		}
	}

	// update to the next state to verify
	qc.State.StateIdx++

	fmt.Printf("Got message %x", msg.Data)
	qc.State.Data = msg.Data

	h := &qc.State.HTLCs[msg.Idx]

	h.Clearing = true
	h.R = msg.R

	if h.Incoming != timeout {
		qc.State.MyAmt += h.Amt
	}

	// verify sig for the next state. only save if this works

	// TODO: There are more signatures required
	err = qc.VerifySigs(msg.CommitmentSignature, msg.HTLCSigs)
	if err != nil {
		return fmt.Errorf("PreimageSigHandler err %s", err.Error())
	}

	// (seems odd, but everything so far we still do in case of collision, so
	// only check here.  If it's a collision, set, save, send gapSigRev

	// save channel with new state, new sig, and positive delta set
	// and maybe collision; still haven't checked
	err = nd.SaveQchanState(qc)
	if err != nil {
		return fmt.Errorf("PreimageSigHandler SaveQchanState err %s", err.Error())
	}

	if qc.State.Collision != 0 {
		err = nd.SendGapSigRev(qc)
		if err != nil {
			return fmt.Errorf("PreimageSigHandler SendGapSigRev err %s", err.Error())
		}
	} else { // saved to db, now proceed to create & sign their tx
		err = nd.SendSigRev(qc)
		if err != nil {
			return fmt.Errorf("PreimageSigHandler SendSigRev err %s", err.Error())
		}
	}
	return nil
}
