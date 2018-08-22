package lnbolt

import (
	"encoding/json"
)

type BoltWallit struct {
	state *bolt.DB
}

// const strings for db usage
var (
	// storage of all utxos. top level is outpoints.
	BKToutpoint = []byte("DuffelBag")
	// storage of all addresses being watched.  top level is pkscripts
	BKTadr = []byte("adr")

	BKTStxos = []byte("SpentTxs")  // for bookkeeping / not sure
	BKTTxns  = []byte("Txns")      // all txs we care about, for replays
	BKTState = []byte("MiscState") // misc states of DB

	//	BKTWatch = []byte("watch") // outpoints we're watching for someone else
	// these are in the state bucket
	KEYNumKeys = []byte("NumKeys") // number of p2pkh keys used

	KEYTipHeight = []byte("TipHeight") // height synced to
)
