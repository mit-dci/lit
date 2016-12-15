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

func (s *SPVCon) GetPriv(k portxo.KeyGen) *btcec.PrivateKey {
	return s.TS.PathPrivkey(k)
}

func (s *SPVCon) GetPub(k portxo.KeyGen) *btcec.PublicKey {
	return s.TS.PathPubkey(k)
}

func (s *SPVCon) PushTx(tx *wire.MsgTx) error {
	return s.DirectSendTx(tx)
}

func (s *SPVCon) Params() *chaincfg.Params {
	return s.Param
}

func (s *SPVCon) BlockMonitor() chan *wire.MsgBlock {
	s.RawBlockSender = make(chan *wire.MsgBlock, 1)
	return s.RawBlockSender
}

func (s *SPVCon) LetMeKnow() chan lnutil.OutPointEvent {
	s.TS.OPEventChan = make(chan lnutil.OutPointEvent, 1)
	return s.TS.OPEventChan
}

// ExportUtxo is really *IM*port utxo on this side.
// Not implemented yet.  Fix "ingest many" at the same time eh?
func (s *SPVCon) ExportUtxo(u *portxo.PorTxo) {

	// zero value utxo counts as an address exort, not utxo export.
	if u.Value == 0 {
		err := s.TS.AddPorTxoAdr(u.KeyGen)
		if err != nil {
			log.Printf(err.Error())
		}
	} else {
		err := s.TS.GainUtxo(*u)
		if err != nil {
			log.Printf(err.Error())
		}
	}

	// make new filter
	filt, err := s.TS.GimmeFilter()
	if err != nil {
		log.Printf(err.Error())
	}
	// send filter
	s.Refilter(filt)
}

// Watch this registers an outpoint to watch.
func (s *SPVCon) WatchThis(op wire.OutPoint) error {
	err := s.TS.RegisterWatchOP(op)
	if err != nil {
		return err
	}
	// make new filter
	filt, err := s.TS.GimmeFilter()
	if err != nil {
		return err
	}
	// send filter
	s.Refilter(filt)
	return nil
}
