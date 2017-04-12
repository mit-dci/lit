package watchtower

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
)

// The main watchtower struct
type WatchTower struct {
	Path    string   // where the DB goes?  needed?
	WatchDB *bolt.DB // DB with everything in it

	Accepting bool // true if new channels and sigs are allowed in
	Watching  bool // true if there are txids to watch for

	SyncHeight int32 // last block we've sync'd to.  Not needed?

	OutBox chan *wire.MsgTx // where the tower sends its justice txs
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

	case lnutil.MSGID_WATCH_COMMSG:
		fmt.Printf("new commsg\n")
		message, ok := msg.(lnutil.ComMsg)
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
