package qln

import (
	"fmt"

	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/lnutil"
)

// ------------------------- break
func (nd *LitNode) BreakChannel(q *Qchan) error {

	if nd.SubWallet[q.Coin()] == nil {
		return fmt.Errorf("Not connected to coin type %d\n", q.Coin())
	}

	err := nd.ReloadQchanState(q)
	if err != nil {
		return err
	}

	if q.CloseData.Closed {
		if q.CloseData.CloseHeight != 0 {
			return fmt.Errorf("Can't break channel %d with peer %d, already closed\n", q.Idx(), q.Peer())
		}
		return fmt.Errorf("Can't break channel %d with peer %d, tx already broadcast, wait for confirmation.\n", q.Idx(), q.Peer())
	}

	logging.Infof("breaking (%d,%d)\n", q.Peer(), q.Idx())

	// set delta to 0... needed for break
	q.State.Delta = 0

	for i, h := range q.State.HTLCs {
		if !h.Cleared && h.Clearing {
			q.State.HTLCs[i].Clearing = false
		}
	}

	q.State.InProgHTLC = nil
	q.State.CollidingHTLC = nil

	tx, err := nd.SignBreakTx(q)
	if err != nil {
		return err
	}

	// broadcast break tx
	err = nd.SubWallet[q.Coin()].PushTx(tx)
	if err != nil {
		return fmt.Errorf("Error while transmitting break tx, try again!")
	}

	// save channel state only after tx is broadcast
	err = nd.SaveQchanUtxoData(q)
	if err != nil {
		return err
	}
	// set channel state to closed
	nd.RemoteMtx.Lock()
	q.CloseData.Closed = true
	nd.RemoteMtx.Unlock()
	q.CloseData.CloseTxid = tx.TxHash()
	return nil
}

func (nd *LitNode) PrintBreakTxForDebugging(q *Qchan) error {
	logging.Info("===== BUILDING Break TX for state [%d]:", q.State.StateIdx)
	saveDelta := q.State.Delta
	q.State.Delta = 0
	tx, err := nd.SignBreakTx(q)
	q.State.Delta = saveDelta
	if err != nil {
		return err
	}
	logging.Info("===== DONE BUILDING Break TX for state [%d]:", q.State.StateIdx)
	logging.Info("Break TX for state [%d]:", q.State.StateIdx)
	lnutil.PrintTx(tx)
	return nil
}
