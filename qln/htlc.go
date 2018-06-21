package qln

import (
	"fmt"
	"log"

	"github.com/mit-dci/lit/portxo"

	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/lnutil"
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

	myNewOutputSize := qc.State.MyAmt - qc.State.Fee - int64(amt)
	theirNewOutputSize := qc.Value - myNewOutputSize - int64(amt)

	for _, h := range qc.State.HTLCs {
		theirNewOutputSize -= h.Amt
	}

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

	keyGen := portxo.KeyGen{}
	keyGen.Depth = 5
	keyGen.Step[0] = 44 | 1<<31
	keyGen.Step[1] = qc.Coin() | 1<<31
	keyGen.Step[2] = UseHTLCBase
	keyGen.Step[3] = qc.State.HTLCIdx | 1<<31
	keyGen.Step[4] = qc.Idx() | 1<<31

	qc.State.InProgHTLC.MyHTLCBase, _ = nd.GetUsePub(keyGen, UseHTLCBase)

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
	q.State.HTLCIdx++

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

	log.Printf("Sending HashSig")

	nd.OmniOut <- outMsg

	return nil
}
