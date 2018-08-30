package uspv

import (
	"log"
	"path/filepath"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/wire"
)

// ChainHook is an interface which provides access to a blockchain for the
// wallit.  Right now the USPV package conforms to this interface, but it
// should also be possible to plug in things like fullnode-rpc clients, and
// (yuck) trusted block explorers.

/*  The model here is that the wallit has lots of state on disk about
what it's seen.  The ChainHook is ephemeral, and has some state in RAM but
shouldn't have to keep track of too much; or at least what it keeps track of
is internal and not exported out to the wallit (eg a fullnode keeps track
of a lot, and SPV node, somewhat less, and a block explorer shim basically nothing)
*/

// ChainHook is a thing that lets you interact with the actual blockchains
type ChainHook interface {

	// Start turns on the ChainHook.  Later on, pass more parameters here.
	// Also gets the txChan where txs come in from the ChainHook to the wallit.
	// The TxChannel should never give txs that the wallit doesn't care about.  In the
	// case of bloom filters, false positives should be handled and stopped at the
	// ChainHook layer; wallit should not have to handle ingesting irrelevant txs.
	// You get back an error and 2 channels: one for txs with height attached, and
	// one with block heights.  Subject to change; maybe this is redundant.

	// Note that for reorgs, the height chan just sends a lower height than you
	// already have, and that means "reorg back!"
	Start(height int32, host, path string, proxyURL string, params *coinparam.Params) (
		chan lnutil.TxAndHeight, chan int32, error)

	// The Register functions send information to the ChainHook about what txs to
	// return.  Txs matching either the addresses or outpoints will be returned
	// via the TxChannel

	// RegisterAddress tells the ChainHook about an address of interest.
	// Give it an array; Currently needs to be 20 bytes.  Only p2pkh / p2wpkh
	// are supported.
	// Later could add a separate function for script hashes (20/32)
	RegisterAddress(address [20]byte) error

	// RegisterOutPoint tells the ChainHook about an outpoint of interest.
	RegisterOutPoint(wire.OutPoint) error

	// UnregisterOutPoint tells the ChainHook about loss of interest in an outpoint.
	UnregisterOutPoint(wire.OutPoint) error

	// SetHeight sets the height ChainHook needs to look above.
	// Returns a channel which tells the wallit what height the ChainHook has
	// sync'd up to.  This chan should push int32s *after* the TxAndHeights
	// have come in; so getting a "54" here means block 54 is done and fully parsed.
	// Removed, put into Start().
	//	SetHeight(startHeight int32) chan int32

	// PushTx sends a tx out to the network via the ChainHook.
	// Note that this does NOT register anything in the tx, so by just using this,
	// nothing will come back about confirmation.  It WILL come back with errors
	// though, so this takes some time.
	PushTx(tx *wire.MsgTx) error

	// Request all incoming blocks over this channel.  If RawBlocks isn't called,
	// then the undelying hook package doesn't need to get full blocks.
	// Currently you always call it with uspv...
	RawBlocks() chan *wire.MsgBlock
	// TODO -- reorgs.  Oh and doublespends and stuff.
}

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

// Start ...
func (s *SPVCon) Start(
	startHeight int32, host, path string, proxyURL string, params *coinparam.Params) (
	chan lnutil.TxAndHeight, chan int32, error) {

	// These can be set before calling Start()
	s.HardMode = true
	s.Ironman = false
	s.ProxyURL = proxyURL

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
		log.Printf("Can't connect to host %s\n", host)
		log.Println(err)
		return nil, nil, err
	}

	err = s.AskForHeaders()
	if err != nil {
		log.Printf("AskForHeaders error\n")
		return nil, nil, err
	}

	return s.TxUpToWallit, s.CurrentHeightChan, nil
}

// RegisterAddress ...
func (s *SPVCon) RegisterAddress(adr160 [20]byte) error {
	s.TrackingAdrsMtx.Lock()
	s.TrackingAdrs[adr160] = true
	s.TrackingAdrsMtx.Unlock()
	return nil
}

// RegisterOutPoint ...
func (s *SPVCon) RegisterOutPoint(op wire.OutPoint) error {
	s.TrackingOPsMtx.Lock()
	s.TrackingOPs[op] = true
	s.TrackingOPsMtx.Unlock()
	return nil
}

func (s *SPVCon) UnregisterOutPoint(op wire.OutPoint) error {
	s.TrackingOPsMtx.Lock()
	delete(s.TrackingOPs, op)
	s.TrackingOPsMtx.Unlock()
	return nil
}

// PushTx sends a tx out to the global network
func (s *SPVCon) PushTx(tx *wire.MsgTx) error {
	// store tx in the RAM map for when other nodes ask for it
	txid := tx.TxHash()
	s.TxMap[txid] = tx

	// since we never delete txs, this will eventually run out of RAM.
	// But might take years... might be nice to fix.

	// send out an inv message telling nodes we have this new tx
	iv1 := wire.NewInvVect(wire.InvTypeWitnessTx, &txid)
	invMsg := wire.NewMsgInv()
	err := invMsg.AddInvVect(iv1)
	if err != nil {
		return err
	}
	// broadcast inv message
	s.outMsgQueue <- invMsg

	// TODO wait a few seconds here for a reject message and return it
	return nil
}

// RawBlocks returns a channel where all the blocks appear.
func (s *SPVCon) RawBlocks() chan *wire.MsgBlock {
	s.RawBlockActive = true
	s.RawBlockSender = make(chan *wire.MsgBlock, 8) // I dunno, 8?
	return s.RawBlockSender
}
