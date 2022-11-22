package wallit

import (
	"fmt"
	"sort"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/uspv"
	"github.com/mit-dci/lit/wire"
)

/*
type UWallet interface {
	GetPub(k portxo.KeyGen) *koblitz.PublicKey

	GetPriv(k portxo.KeyGen) *koblitz.PrivateKey

	PushTx(tx *wire.MsgTx) error
	ExportUtxo(txo *portxo.PorTxo)
	MaybeSend(txos []*wire.TxOut) ([]*wire.OutPoint, error)
	ReallySend(txid *chainhash.Hash) error
	NahDontSend(txid *chainhash.Hash) error
	WatchThis(wire.OutPoint) error
	LetMeKnow() chan lnutil.OutPointEvent
	LetMeKnowHeight() chan lnutil.HeightEvent
	BlockMonitor() chan *wire.MsgBlock

	Params() *coinparam.Params
}
*/

// --- implementation of BaseWallet interface ----

func (w *Wallit) GetPriv(k portxo.KeyGen) (*koblitz.PrivateKey, error) {
	if w.PathPrivkey(k) != nil {
		return w.PathPrivkey(k), nil
	} else {
		return nil, fmt.Errorf("Nil Wallet Error")
	}
}

func (w *Wallit) GetPub(k portxo.KeyGen) *koblitz.PublicKey {
	return w.PathPubkey(k)
}

func (w *Wallit) PushTx(tx *wire.MsgTx) error {
	return w.Hook.PushTx(tx)
}

func (w *Wallit) Params() *coinparam.Params {
	return w.Param
}

func (w *Wallit) LetMeKnow() chan lnutil.OutPointEvent {
	w.OPEventChan = make(chan lnutil.OutPointEvent, 1)
	return w.OPEventChan
}

func (w *Wallit) LetMeKnowHeight() chan lnutil.HeightEvent {
	w.HeightEventChan = make(chan lnutil.HeightEvent, 1)
	return w.HeightEventChan
}

func (w *Wallit) CurrentHeight() int32 {
	h, err := w.GetDBSyncHeight()
	if err != nil {
		logging.Errorf("can't get height from db...")
		return -99
	}
	return h
}

func (w *Wallit) NewAdr() ([20]byte, error) {
	return w.NewAdr160()
}

func (w *Wallit) ExportHook() uspv.ChainHook {
	return w.Hook
}

// ExportUtxo is really *IM*port utxo on this side.
func (w *Wallit) ExportUtxo(u *portxo.PorTxo) {

	// zero value utxo counts as an address export, not utxo export.
	if u.Value == 0 {
		err := w.AddPorTxoAdr(u.KeyGen)
		if err != nil {
			logging.Errorf(err.Error())
		}
	} else {
		// if derivation path is 0, that means it's an import triggerd by
		// the user, with a portxo with private key included that does
		// not match up with the wallet derivation tree.  In this
		// case we assign a fixed derivation path and subtract
		if u.KeyGen.Depth == 0 {
			impGen := portxo.KeyGenForImports
			impGen.Step[1] = w.Param.HDCoinType

			var privMaskArr [32]byte
			privMask, err := w.GetPriv(impGen)
			if err != nil {
				logging.Errorf(err.Error())
				return
			}
			copy(privMaskArr[:], privMask.D.Bytes())

			mixedKey := lnutil.SubtractPrivKeys(u.KeyGen.PrivKey, privMaskArr)

			u.KeyGen = impGen
			u.KeyGen.PrivKey = mixedKey
		}
		// either way, gain the utxo
		err := w.GainUtxo(*u)
		if err != nil {
			logging.Errorf(err.Error())
		}
	}

	return
	// don't register an address; utxo import does not imply we
	// have control over that address and can accept new payments there
	// (even though we probably could...)
}

// WatchThis registers an outpoint to watch.  Register as watched OP, and
// passes to chainhook.
func (w *Wallit) WatchThis(op wire.OutPoint) error {

	// first, tell the chainhook
	err := w.Hook.RegisterOutPoint(op)
	if err != nil {
		return err
	}

	// then register in the wallit
	err = w.RegisterWatchOP(op)
	if err != nil {
		return err
	}

	return nil
}

// StopWatchingThis removes an outpoint to watch.
func (w *Wallit) StopWatchingThis(op wire.OutPoint) error {

	// first, tell the chainhook
	err := w.Hook.RegisterOutPoint(op)
	if err != nil {
		return err
	}

	// then unregister from the wallit
	err = w.UnregisterWatchOP(op)
	if err != nil {
		return err
	}

	return nil
}

func (w *Wallit) Fee() int64 {
	return w.FeeRate
}

func (w *Wallit) SetFee(set int64) int64 {
	w.FeeRate = set
	return set
}

// ********* sweep is for testing / spamming, remove for real use
func (w *Wallit) Sweep(outScript []byte, n uint32) ([]*chainhash.Hash, error) {
	var err error
	var txids []*chainhash.Hash

	var utxos portxo.TxoSliceByAmt
	utxos, err = w.GetAllUtxos()
	if err != nil {
		return nil, err
	}

	// smallest and unconfirmed last (because it's reversed)
	sort.Sort(sort.Reverse(utxos))

	for _, u := range utxos {
		if n < 1 {
			return txids, nil
		}

		// this doesn't really work with maybeSend huh...
		if u.Height != 0 && u.Value > 20000 {
			tx, err := w.SendOne(*u, outScript)
			if err != nil {
				return nil, err
			}

			_, err = w.Ingest(tx, 0)
			if err != nil {
				return nil, err
			}

			err = w.PushTx(tx)
			if err != nil {
				return nil, err
			}
			txid := tx.TxHash()
			txids = append(txids, &txid)

			n--
		}
	}

	return txids, nil
}
