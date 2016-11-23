package watchtower

import (
	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	defaultpath = "."
	dbname      = "watchtower.db"
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

	HAKDBasePoint [33]byte // client's HAKD key base point
	TimeBasePoint [33]byte // potential attacker's timeout basepoint

	// elk 0 here?  Because they won't send a sig for elk0...
	ElkZero chainhash.Hash
}

// the message describing the next commitment tx, sent from the client to the watchtower
type ComMsg struct {
	DestPKH [20]byte       // identifier for channel; could be optimized away
	Elk     chainhash.Hash // elkrem for this state index
	ParTxid [16]byte       // first half of txid of close tx
	Sig     [64]byte       // sig for the grab tx
}

// 2 structs used in the DB: IdxSigs and ChanStatic

// IdxSig is what we save in the DB for each txid
type IdxSig struct {
	PKHIdx   uint32   // Who
	StateIdx uint64   // When
	Sig      [64]byte // What
}
