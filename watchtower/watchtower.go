package watchtower

import (
	"fmt"

	"github.com/adiabat/btcd/wire"
	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/lnutil"
)

type Watcher interface {
	// Links to the blockchain.  Blocks go in to the watcher, and
	// justice transactions come out.  The uint32 is the cointype
	ChainLink(uint32, chan *wire.MsgBlock) chan *wire.MsgTx

	// New Channel to watch
	NewChannel(lnutil.WatchDescMsg) error

	// Update a channel being watched
	UpdateChannel(lnutil.WatchStateMsg) error

	// Delete a channel being watched
	DeteleChannel(lnutil.WatchDelMsg) error

	// Later on, allow users to recover channel state from
	// the data in a watcher.  Like if they wipe their ln.db files but
	// still have their keys.
}

// The main watchtower struct
type WatchTower struct {
	Path    string   // where the DB goes?  needed?
	WatchDB *bolt.DB // DB with everything in it

	Accepting bool // true if new channels and sigs are allowed in
	Watching  bool // true if there are txids to watch for

	SyncHeight int32 // last block we've sync'd to.  Not needed?

	OutBox chan *wire.MsgTx // where the tower sends its justice txs
}

func (w *WatchTower) ChainLink(
	cointype uint32, blockchan chan *wire.MsgBlock) chan *wire.MsgTx {

	go w.BlockHandler(blockchan)

	return nil
}

// 2 structs used in the DB: IdxSigs and ChanStatic

// IdxSig is what we save in the DB for each txid
type IdxSig struct {
	PKHIdx   uint32   // Who
	StateIdx uint64   // When
	Sig      [64]byte // What
}

func (w *WatchTower) HandleMessage(msg lnutil.LitMsg) error {
	fmt.Printf("got message from %x\n", msg.Peer())

	switch msg.MsgType() {
	case lnutil.MSGID_WATCH_DESC:
		fmt.Printf("new channel to watch\n")
		message, ok := msg.(lnutil.WatchDescMsg)
		if !ok {
			return fmt.Errorf("didn't work")
		} else {
			return w.AddNewChannel(message)
		}

	case lnutil.MSGID_WATCH_STATEMSG:
		fmt.Printf("new commsg\n")
		message, ok := msg.(lnutil.WatchStateMsg)
		if !ok {
			return fmt.Errorf("didn't work")
		} else {
			return w.AddState(message)
		}

	case lnutil.MSGID_WATCH_DELETE:
		fmt.Printf("delete message\n")
		// delete not yet implemented
	default:
		fmt.Printf("unknown message type %x\n", msg.MsgType())
	}
	return nil
}

func (w *WatchTower) JusticeOutbox() chan *wire.MsgTx {
	return w.OutBox
}
