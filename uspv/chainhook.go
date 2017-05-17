package uspv

import (
	"path/filepath"

	"github.com/adiabat/btcd/chaincfg/chainhash"
	"github.com/adiabat/btcd/wire"
	"github.com/mit-dci/lit/coinparam"
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
	startHeight int32, host, path string, params *coinparam.Params) (
	chan lnutil.TxAndHeight, chan int32, error) {

	// These can be set before calling Start()
	s.HardMode = true
	s.Ironman = false

	s.Param = params

	s.TrackingAdrs = make(map[[20]byte]bool)
	s.TrackingOPs = make(map[wire.OutPoint]bool)

	s.TxMap = make(map[chainhash.Hash]*wire.MsgTx)

	s.OKTxids = make(map[chainhash.Hash]int32)

	s.TxUpToWallit = make(chan lnutil.TxAndHeight, 1)
	s.CurrentHeightChan = make(chan int32, 1)

	s.syncHeight = startHeight

	headerFilePath := filepath.Join(path, "header.bin")
	// open header file
	err := s.openHeaderFile(headerFilePath)
	if err != nil {
		return nil, nil, err
	}

	err = s.Connect(host)
	if err != nil {
		return nil, nil, err
	}

	err = s.AskForHeaders()
	if err != nil {
		return nil, nil, err
	}

	return s.TxUpToWallit, s.CurrentHeightChan, nil
}

func (s *SPVCon) RegisterAddress(adr160 [20]byte) error {
	s.TrackingAdrsMtx.Lock()
	s.TrackingAdrs[adr160] = true
	s.TrackingAdrsMtx.Unlock()
	return nil
}

func (s *SPVCon) RegisterOutPoint(op wire.OutPoint) error {
	s.TrackingOPsMtx.Lock()
	s.TrackingOPs[op] = true
	s.TrackingOPsMtx.Unlock()
	return nil
}

// PushTx sends a tx out to the global network
func (s *SPVCon) PushTx(tx *wire.MsgTx) error {
	// store tx in the RAM map for when other nodes ask for it
	txid := tx.TxHash()
	s.TxMap[txid] = tx

	// send out an inv message telling nodes we have this new tx
	iv1 := wire.NewInvVect(wire.InvTypeWitnessTx, &txid)
	invMsg := wire.NewMsgInv()
	err := invMsg.AddInvVect(iv1)
	if err != nil {
		return err
	}
	// broadcast inv message
	s.outMsgQueue <- invMsg

	return nil
}

func (s *SPVCon) RawBlocks() chan *wire.MsgBlock {
	return s.RawBlockSender
}
