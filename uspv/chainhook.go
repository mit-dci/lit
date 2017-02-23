package uspv

import "github.com/btcsuite/btcd/wire"

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

func (s *SPVCon) Start() error {
	/*
	   p *chaincfg.Params,
	   	headerFileName, dbFileName string, hard, iron bool) error {

	   	s.HardMode = hard
	   	s.Ironman = iron
	   	s.Param = p

	   	s.OKTxids = make(map[chainhash.Hash]int32)

	   	err := s.openHeaderFile(headerFileName)
	   	if err != nil {
	   		return err
	   	}
	*/
	return nil
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
