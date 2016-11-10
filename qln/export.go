package qln

import (
	"fmt"

	"github.com/mit-dci/lit/lnutil"

	"github.com/btcsuite/btcd/wire"
)

// it's not really client / server, it's client and watcher.  but we can say
// client / server.
// on the client we should keep track of what we've sent to the server.  Initially
// it can just be what state we've exported up to; 0 means we haven't sent anything,
// as channels start on state 1.

/*

Since we don't remember anything about previous states, we can't
reproduce them and create penalty txs.  So we have to do that each time we make a
new state, and then we can clear them out when sending to the watchtower.

We could keep track of previous states, but then we're at O(n) storage.  And who
needs receipts for all those donuts.

*/

// WatchState returns the txid / sig pair for the current state's penalty tx.
// It can't make the elk, because it doesn't have it yet.
func (nd *LnNode) WatchState(qc *Qchan) error {
	var err error

	fee := int64(5000) // fixed fee for now

	tx, err := qc.BuildStateTx(true)
	if err != nil {
		return err
	}
	closeTxid := tx.TxHash()

	var revIdx uint32
	var revAmt int64
	for i, out := range tx.TxOut {
		if out.Value == qc.State.MyAmt-fee {
			revIdx = uint32(i)
			revAmt = out.Value
			break
		}
	}

	if revIdx == 255 {
		return fmt.Errorf("couldn't find revocable SH output")
	}

	// the justiceTx is the tx which the watchtower will generate and broadcast.
	// It's usually called the "penalty" tx but this sounds more fun
	justiceTx := wire.NewMsgTx()

	// set to version 2, though might not matter as no CSV is used
	justiceTx.Version = 2

	// make the outpoint being spent
	justiceOP := wire.NewOutPoint(&closeTxid, revIdx)

	// txin has no sigscript / witness
	justiceIn := wire.NewTxIn(justiceOP, nil, nil)

	// add input
	justiceTx.AddTxIn(justiceIn)

	// make output script
	justiceScript := lnutil.DirectWPKHScriptFromPKH(qc.WatchRefundAdr)

	// make txout
	justiceOut := wire.NewTxOut(revAmt, justiceScript)
	// add txout to justice tx
	justiceTx.AddTxOut(justiceOut)

	// now have txid, but can't make signature

	return nil
}

/*
	if !(qc.State.WatchUpTo < qc.State.StateIdx) {
		// equal is also not OK; can't send elks for current state
		return fmt.Errorf("watchupto is equal / more than current state")
	}

	if qc.State.WatchUpTo < qc.State.StateIdx-1 {

	}

	synced := qc.State.WatchUpTo == qc.State.StateIdx-1
	if synced {
		return synced, nil
	}

	// we're more than 1 state behind current, so can generate.
	stateToBuild := qc.State.WatchUpTo + 1

	cm := new(watchtower.ComMsg)

	// copy watchRefundPKH to identify channel
	cm.DestPKHScript = qc.WatchRefundAdr

	// Get the elkrem
	elk, err := qc.ElkRcv.AtIndex(stateToBuild)
	if err != nil {
		return false, nil
	}
	cm.Elk = *elk

	msg := []byte{watchtower.MSGID_WATCH_COMMSG}

	_, err := nd.WatchCon.Write(msg)
	if err != nil {
		return synced, err
	}


*/
