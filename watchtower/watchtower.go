package watchtower

import (
	"fmt"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/uspv"
)

type Watcher interface {
	// Links to the blockchain.
	// Uses the same chainhook interface as the wallit does.  But only uses
	// 2 of the functions: PushTx() and RawBlocks()
	// Blocks come in from the chainhook, and justice transactions come out.
	// The uint32 is the cointype, the string is the folder to put all db files.
	HookLink(string, *coinparam.Params, uspv.ChainHook) error

	// New Channel to watch
	NewChannel(lnutil.WatchDescMsg) error

	// Update a channel being watched
	UpdateChannel(lnutil.WatchStateMsg) error

	// Delete a channel being watched
	DeleteChannel(lnutil.WatchDelMsg) error

	// Later on, allow users to recover channel state from
	// the data in a watcher.  Like if they wipe their ln.db files but
	// still have their keys.
}

// The main watchtower struct
type WatchTower struct {
	Path string // where the DB goes?  needed?

	WatchDB *bolt.DB // single DB with everything in it
	// much more efficient to have a separate DB for each cointype
	// ... but that's less anonymous.  To get that efficiency; make a bunch of
	// towers, I guess.

	Accepting bool // true if new channels and sigs are allowed in
	Watching  bool // true if there are txids to watch for

	SyncHeight int32 // last block we've sync'd to.  Not needed?

	// map of cointypes to chainhooks
	Hooks map[uint32]uspv.ChainHook
}

// Hooklink is the connection between the watchtower and the blockchain
// Takes in a channel of blocks, and the cointype.  Immediately returns
// a channel which it will send justice transactions to.
func (w *WatchTower) HookLink(dbPath string, param *coinparam.Params,
	hook uspv.ChainHook) error {

	cointype := param.HDCoinType

	// if the hooks map hasn't been initialized, make it. also open DB
	if len(w.Hooks) == 0 {
		w.Hooks = make(map[uint32]uspv.ChainHook)

		towerDBName := filepath.Join(dbPath, "watch.db")
		err := w.OpenDB(towerDBName)
		if err != nil {
			return err
		}

	}

	// see if this cointype is already registered
	_, ok := w.Hooks[cointype]
	if ok {
		return fmt.Errorf("Coin type %d already linked", cointype)
	}

	// only need this for the pushTx() method
	w.Hooks[cointype] = hook

	go w.BlockHandler(cointype, hook.RawBlocks())

	return nil
}

// 2 structs used in the DB: IdxSigs and ChanStatic

// IdxSig is what we save in the DB for each txid
type IdxSig struct {
	PKHIdx   uint32   // Who
	StateIdx uint64   // When
	Sig      [64]byte // What
}

/*
func (w *WatchTower) HandleMessage(msg lnutil.LitMsg) error {
	logging.Infof("got message from %x\n", msg.Peer())

	switch msg.MsgType() {
	case lnutil.MSGID_WATCH_DESC:
		logging.Infof("new channel to watch\n")
		message, ok := msg.(lnutil.WatchDescMsg)
		if !ok {
			return fmt.Errorf("didn't work")
		} else {
			return w.AddNewChannel(message)
		}

	case lnutil.MSGID_WATCH_STATEMSG:
		logging.Infof("new commsg\n")
		message, ok := msg.(lnutil.WatchStateMsg)
		if !ok {
			return fmt.Errorf("didn't work")
		} else {
			return w.AddState(message)
		}

	case lnutil.MSGID_WATCH_DELETE:
		logging.Infof("delete message\n")
		// delete not yet implemented
	default:
		logging.Infof("unknown message type %x\n", msg.MsgType())
	}
	return nil
}
*/
