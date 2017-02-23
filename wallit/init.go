package wallit

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/uspv"
)

func NewWallit(rootkey *hdkeychain.ExtendedKey, p *chaincfg.Params) Wallit {
	var w Wallit
	w.rootPrivKey = rootkey
	w.Param = p
	w.FreezeSet = make(map[wire.OutPoint]*FrozenTx)

	u := new(uspv.SPVCon)
	w.Hook = u
	incomingTx, incomingBlockheight, err := w.Hook.Start(1000000, ".", p)
	if err != nil {
		fmt.Printf("crash   ")
	}

	// deal with the incoming txs

	go w.TxHandler(incomingTx)

	// deal with incoming height

	go w.HeightHandler(incomingBlockheight)

	return w
}

func (w *Wallit) TxHandler(incomingTxAndHeight chan lnutil.TxAndHeight) {
	for {
		txah := <-incomingTxAndHeight
		fmt.Printf("got tx %s at height %d\n",
			txah.Tx.TxHash().String(), txah.Height)
	}
}

func (w *Wallit) HeightHandler(incomingHeight chan int32) {
	for {
		h := <-incomingHeight
		fmt.Printf("got height %d\n", h)
	}
}
