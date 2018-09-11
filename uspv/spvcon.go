package uspv

import (
	"net"
	"os"
	"sync"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/wire"
)

// SPVCon is a SPV connection to a coin daemon.
type SPVCon struct {
	con net.Conn // the (probably tcp) connection to the node

	// Enhanced SPV modes for users who have outgrown easy mode SPV
	// but have not yet graduated to full nodes.
	HardMode bool   // hard mode doesn't use filters.
	Ironman  bool   // ironman only gets blocks, never requests txs.
	ProxyURL string // Optionally the URL of a SOCKS5 proxy to use

	headerMutex       sync.Mutex
	headerFile        *os.File // file for SPV headers
	headerStartHeight int32    // first header on disk is nth header in chain

	syncHeight int32 // internal, in memory synchronization height

	OKTxids map[chainhash.Hash]int32 // known good txids and their heights
	OKMutex sync.Mutex

	// TrackingAdrs and OPs are slices of addresses and outpoints to watch for.
	// Using struct{} saves a byte of RAM but is ugly so I'll use bool.
	TrackingAdrs    map[[20]byte]bool
	TrackingAdrsMtx sync.Mutex

	TrackingOPs    map[wire.OutPoint]bool
	TrackingOPsMtx sync.Mutex

	// TxMap is an in-memory map of all the Txs the SPVCon knows about
	TxMap map[chainhash.Hash]*wire.MsgTx

	//[doesn't work without fancy mutexes, nevermind, just use header file]
	// localHeight   int32  // block height we're on
	remoteHeight  int32  // block height they're on
	localVersion  uint32 // version we report
	remoteVersion uint32 // version remote node

	// what's the point of the input queue? remove? leave for now...
	inMsgQueue  chan wire.Message // Messages coming in from remote node
	outMsgQueue chan wire.Message // Messages going out to remote node

	WBytes uint64 // total bytes written
	RBytes uint64 // total bytes read

	Param *coinparam.Params // network parameters (testnet3, segnet, etc)

	// TxUpToWallit is the channel for sending txs up a level to the wallit.
	TxUpToWallit chan lnutil.TxAndHeight
	// CurrentHeightChan is how we tell the wallit when blocks come in
	CurrentHeightChan chan int32

	// RawBlockSender is a channel to send full blocks up to the qln / watchtower
	// only kicks in when requested from upper layer
	RawBlockSender chan *wire.MsgBlock

	// If the above RawBlockSender chan isn't being pulled from, don't send to it
	RawBlockActive bool

	// for internal use -------------------------

	// mBlockQueue is for keeping track of what height we've requested.
	blockQueue chan HashAndHeight
	// fPositives is a channel to keep track of bloom filter false positives.
	fPositives chan int32

	// waitState is a channel that is empty while in the header and block
	// sync modes, but when in the idle state has a "true" in it.
	inWaitState   chan bool
	randomNodesOK bool
}
