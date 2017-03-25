package watchtower

import (
	"log"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
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
	Path    string   // where the DB goes?  needed?
	WatchDB *bolt.DB // DB with everything in it

	Accepting bool // true if new channels and sigs are allowed in
	Watching  bool // true if there are txids to watch for

	SyncHeight int32 // last block we've sync'd to.  Not needed?

	OutBox chan *wire.MsgTx // where the tower sends its justice txs
}

// 2 structs that the watchtower gets from clients: Descriptors and Msgs

// WatchannelDescriptor is the initial message setting up a Watchannel
type WatchannelDescriptor struct {
	DestPKHScript [20]byte // PKH to grab to; main unique identifier.

	Delay uint16 // timeout in blocks
	Fee   int64  // fee to use for grab tx.  Or fee rate...?

	CustomerBasePoint  [33]byte // client's HAKD key base point
	AdversaryBasePoint [33]byte // potential attacker's timeout basepoint
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

func (w *WatchTower) HandleMessage(lm *lnutil.LitMsg) error {
	log.Printf("got message from %x\n", lm.PeerIdx)

	switch lm.MsgType {
	case MSGID_WATCH_DESC:
		log.Printf("new channel to watch\n")
		desc := WatchannelDescriptorFromBytes(lm.Data)
		return w.AddNewChannel(desc)

	case MSGID_WATCH_COMMSG:
		log.Printf("new commsg\n")
		commsg := ComMsgFromBytes(lm.Data)
		return w.AddState(commsg)

	case MSGID_WATCH_DELETE:
		log.Printf("delete message\n")
		// delete not yet implemented
	default:
		log.Printf("unknown message type %x\n", lm.MsgType)
	}
	return nil
}

func (w *WatchTower) JusticeOutbox() chan *wire.MsgTx {
	return w.OutBox
}
