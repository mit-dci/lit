package qln

import (
	"fmt"
	log "github.com/mit-dci/lit/logs"
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

	if q.CloseData.Closed && q.CloseData.CloseHeight != 0 {
		return fmt.Errorf("Can't break (%d,%d), already closed\n", q.Peer(), q.Idx())
	}

	log.Infof("breaking (%d,%d)\n", q.Peer(), q.Idx())
	z, err := q.ElkSnd.AtIndex(0)
	if err != nil {
		return err
	}
	log.Debugf("elk send 0: %s\n", z.String())
	z, err = q.ElkRcv.AtIndex(0)
	if err != nil {
		return err
	}
	log.Debugf("elk recv 0: %s\n", z.String())

	// set delta to 0... needed for break
	q.State.Delta = 0
	tx, err := nd.SignBreakTx(q)
	if err != nil {
		return err
	}

	// set channel state to closed
	q.CloseData.Closed = true
	q.CloseData.CloseTxid = tx.TxHash()

	err = nd.SaveQchanUtxoData(q)
	if err != nil {
		return err
	}

	// broadcast break tx directly
	return nd.SubWallet[q.Coin()].PushTx(tx)
}
