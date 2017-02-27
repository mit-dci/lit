package wallit

import (
	"log"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
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

func (w Wallit) GetPriv(k portxo.KeyGen) *btcec.PrivateKey {
	return w.PathPrivkey(k)
}

func (w Wallit) GetPub(k portxo.KeyGen) *btcec.PublicKey {
	return w.PathPubkey(k)
}

func (w Wallit) PushTx(tx *wire.MsgTx) error {
	return w.Hook.PushTx(tx)
}

func (w Wallit) Params() *chaincfg.Params {
	return w.Param
}

func (w Wallit) BlockMonitor() chan *wire.MsgBlock {
	return w.Hook.RawBlocks()
}

func (w Wallit) LetMeKnow() chan lnutil.OutPointEvent {
	w.OPEventChan = make(chan lnutil.OutPointEvent, 1)
	return w.OPEventChan
}

// ExportUtxo is really *IM*port utxo on this side.
// Not implemented yet.  Fix "ingest many" at the same time eh?
func (w Wallit) ExportUtxo(u *portxo.PorTxo) {

	// zero value utxo counts as an address exort, not utxo export.
	if u.Value == 0 {
		err := w.AddPorTxoAdr(u.KeyGen)
		if err != nil {
			log.Printf(err.Error())
		}
	} else {
		err := w.GainUtxo(*u)
		if err != nil {
			log.Printf(err.Error())
		}
	}

	// Register new address with chainhook
	var adr160 [20]byte
	copy(adr160[:], w.PathPubHash160(u.KeyGen))
	err := w.Hook.RegisterAddress(adr160)
	if err != nil {
		log.Printf(err.Error())
	}
}

// WatchThis registers an outpoint to watch.  Register as watched OP, and
// passes to chainhook.
func (w Wallit) WatchThis(op wire.OutPoint) error {
	err := w.Hook.RegisterOutPoint(op)
	if err != nil {
		return err
	}
	err = w.Hook.RegisterOutPoint(op)
	if err != nil {
		return err
	}
	return nil
}
