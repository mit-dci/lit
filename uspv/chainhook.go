package uspv

import (
	"path/filepath"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
)

/*
type ChainHook interface {

	Start() chan lnutil.TxAndHeight
	RegisterAddress(address [20]byte) error
	RegisterOutPoint(wire.OutPoint) error
	SetHeight(startHeight int32) chan int32
	PushTx(tx *wire.MsgTx) error
	RawBlocks() chan *wire.MsgBlock
}
*/

// --- implementation of ChainHook interface ----

func (s *SPVCon) Start(
	startHeight int32, path string, params *chaincfg.Params) (
	chan lnutil.TxAndHeight, chan int32, error) {

	// These can be set before calling Start()
	s.HardMode = true
	s.Ironman = false

	s.Param = params

	s.OKTxids = make(map[chainhash.Hash]int32)

	s.TxUpToWallit = make(chan lnutil.TxAndHeight, 1)
	s.CurrentHeightChan = make(chan int32, 1)

	coinName := params.Name
	path = filepath.Join(path, coinName)

	headerFilePath := filepath.Join(path, "header.bin")
	// open header file
	err := s.openHeaderFile(headerFilePath)
	if err != nil {
		return nil, nil, err
	}

	return s.TxUpToWallit, s.CurrentHeightChan, nil
}

func (s *SPVCon) RegisterAddress(adr160 [20]byte) error {
	return nil
}

func (s *SPVCon) RegisterOutPoint(wire.OutPoint) error {
	return nil
}

func (s *SPVCon) SetHeight(startHeight int32) chan int32 {
	return s.CurrentHeightChan
}

func (s *SPVCon) PushTx(tx *wire.MsgTx) error {
	return nil
}

func (s *SPVCon) RawBlocks() chan *wire.MsgBlock {
	return s.RawBlockSender
}
