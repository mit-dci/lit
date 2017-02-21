package uspv

import (
	"log"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

// --- Uwallet interface ----

func (w *Wallit) GetPriv(k portxo.KeyGen) *btcec.PrivateKey {
	return w.PathPrivkey(k)
}

func (w *Wallit) GetPub(k portxo.KeyGen) *btcec.PublicKey {
	return w.PathPubkey(k)
}

func (w *Wallit) PushTx(tx *wire.MsgTx) error {
	return w.SpvHook.DirectSendTx(tx)
}

func (w *Wallit) Params() *chaincfg.Params {
	return w.Param
}

func (w *Wallit) BlockMonitor() chan *wire.MsgBlock {
	w.SpvHook.RawBlockSender = make(chan *wire.MsgBlock, 1)
	return w.SpvHook.RawBlockSender
}

func (w *Wallit) LetMeKnow() chan lnutil.OutPointEvent {
	w.OPEventChan = make(chan lnutil.OutPointEvent, 1)
	return w.OPEventChan
}

// ExportUtxo is really *IM*port utxo on this side.
// Not implemented yet.  Fix "ingest many" at the same time eh?
func (w *Wallit) ExportUtxo(u *portxo.PorTxo) {

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

	// make new filter
	filt, err := w.GimmeFilter()
	if err != nil {
		log.Printf(err.Error())
	}
	// send filter
	w.SpvHook.Refilter(filt)
}

// Watch this registers an outpoint to watch.
func (w *Wallit) WatchThis(op wire.OutPoint) error {
	err := w.RegisterWatchOP(op)
	if err != nil {
		return err
	}
	// make new filter
	filt, err := w.GimmeFilter()
	if err != nil {
		return err
	}
	// send filter
	w.Refilter(filt)
	return nil
}
