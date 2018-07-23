package wallit

import (
	"fmt"
	."github.com/mit-dci/lit/logs"
	"sort"

	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/wire"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/uspv"
)

/*
type UWallet interface {
	GetPub(k portxo.KeyGen) *btcec.PublicKey

	GetPriv(k portxo.KeyGen) *btcec.PrivateKey

	PushTx(tx *wire.MsgTx) error
	ExportUtxo(txo *portxo.PorTxo)
	MaybeSend(txos []*wire.TxOut) ([]*wire.OutPoint, error)
	ReallySend(txid *chainhash.Hash) error
	NahDontSend(txid *chainhash.Hash) error
	WatchThis(wire.OutPoint) error
	LetMeKnow() chan lnutil.OutPointEvent
	BlockMonitor() chan *wire.MsgBlock

	Params() *chaincfg.Params
}
*/

// --- implementation of BaseWallet interface ----

func (w *Wallit) GetPriv(k portxo.KeyGen) (*btcec.PrivateKey, error) {
	if w.PathPrivkey(k) != nil {
		return w.PathPrivkey(k), nil
	} else {
		return nil, fmt.Errorf("Nil Wallet Error")
	}
}

func (w *Wallit) GetPub(k portxo.KeyGen) *btcec.PublicKey {
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

func (w *Wallit) CurrentHeight() int32 {
	h, err := w.GetDBSyncHeight()
	if err != nil {
		Log.Error("can't get height from db...")
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
// Not implemented yet.  Fix "ingest many" at the same time eh?
func (w *Wallit) ExportUtxo(u *portxo.PorTxo) {

	// zero value utxo counts as an address exort, not utxo export.
	if u.Value == 0 {
		err := w.AddPorTxoAdr(u.KeyGen)
		if err != nil {
			Log.Error(err.Error())
		}
	} else {
		err := w.GainUtxo(*u)
		if err != nil {
			Log.Error(err.Error())
		}
	}

	// Register new address with chainhook
	adr160 := w.PathPubHash160(u.KeyGen)
	err := w.Hook.RegisterAddress(adr160)
	if err != nil {
		Log.Error(err.Error())
	}
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
