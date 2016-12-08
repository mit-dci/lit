package watchtower

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	// desc describes a new channel
	MSGID_WATCH_DESC = 0xA0
	// commsg is a single state in the channel
	MSGID_WATCH_COMMSG = 0xA1
	// Watch_clear marks a channel as ok to delete.  No further updates possible.
	MSGID_WATCH_DELETE = 0xA2
)

// The main watchtower struct
type WatchTower struct {
	Path       string   // where the DB goes?  needed?
	WatchDB    *bolt.DB // DB with everything in it
	SyncHeight int32    // last block we've sync'd to.  Needed?
}

// 2 structs that the watchtower gets from clients: Descriptors and Msgs

// WatchannelDescriptor is the initial message setting up a Watchannel
type WatchannelDescriptor struct {
	DestPKHScript [20]byte // PKH to grab to; main unique identifier.

	Delay uint16 // timeout in blocks
	Fee   int64  // fee to use for grab tx.  Or fee rate...?

	CustomerBasePoint  [33]byte // client's HAKD key base point
	AdversaryBasePoint [33]byte // potential attacker's timeout basepoint

	// elk 0 here?  Because they won't send a sig for elk0...
	ElkZero chainhash.Hash
}

// the message describing the next commitment tx, sent from the client to the watchtower
type ComMsg struct {
	DestPKH [20]byte       // identifier for channel; could be optimized away
	Elk     chainhash.Hash // elkrem for this state index
	ParTxid [16]byte       // 16 bytes of txid
	Sig     [64]byte       // 64 bytes of sig
}

// 2 structs used in the DB: IdxSigs and ChanStatic

// IdxSig is what we save in the DB for each txid
type IdxSig struct {
	PKHIdx   uint32   // Who
	StateIdx uint64   // When
	Sig      [64]byte // What
}

func (w *WatchTower) HandleMessage(from [16]byte, msg []byte) error {
	fmt.Printf("got message from %x\n", from)

	switch msg[0] {
	case MSGID_WATCH_DESC:
		fmt.Printf("new channel to watch\n")
		desc := WatchannelDescriptorFromBytes(msg[1:])
		return w.AddNewChannel(desc)

	case MSGID_WATCH_COMMSG:
		fmt.Printf("new commsg\n")
		commsg := ComMsgFromBytes(msg[1:])
		return w.AddState(commsg)

	case MSGID_WATCH_DELETE:
		fmt.Printf("delete message\n")
		// delete not yet implemented
	default:
		fmt.Printf("unknown message type %x\n", msg[0])
	}
	return nil
}
