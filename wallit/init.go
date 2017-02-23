package wallit

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/mit-dci/lit/uspv"
)

func NewWallit(rootkey *hdkeychain.ExtendedKey, p *chaincfg.Params) Wallit {
	var w Wallit
	w.rootPrivKey = rootkey
	w.Param = p
	w.FreezeSet = make(map[wire.OutPoint]*FrozenTx)

	u, err := uspv.NewSPV()
	w.Hook = u
	return w
}
