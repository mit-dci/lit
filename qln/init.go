package qln

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/wire"
)

func (nd *LnNode) Init(
	dbfilename, watchname string, basewal UWallet, tower bool) error {

	err := nd.OpenDB(dbfilename)
	if err != nil {
		return err
	}

	nd.OmniChan = make(chan []byte, 10)
	go nd.OmniHandler()

	// connect to base wallet
	nd.BaseWallet = basewal
	// ask basewallet for outpoint event messages
	go nd.OPEventHandler(nd.BaseWallet.LetMeKnow())
	// optional tower activation
	if tower {
		err = nd.Tower.OpenDB(watchname)
		if err != nil {
			return err
		}
		nd.Tower.Accepting = true
		// call base wallet blockmonitor and hand this channel to the tower
		go nd.Tower.BlockHandler(nd.BaseWallet.BlockMonitor())
		go nd.Relay(nd.Tower.JusticeOutbox())
	}

	return nil
}

// relay txs from the watchtower to the underlying wallet...
// small, but a little ugly; maybe there's a cleaner way
func (nd *LnNode) Relay(outbox chan *wire.MsgTx) {
	for {
		err := nd.BaseWallet.PushTx(<-outbox)
		if err != nil {
			fmt.Printf("PushTx error: %s", err.Error())
		}
	}
}

// Opens the DB file for the LnNode
func (nd *LnNode) OpenDB(filename string) error {
	var err error

	nd.LnDB, err = bolt.Open(filename, 0644, nil)
	if err != nil {
		return err
	}
	// create buckets if they're not already there
	err = nd.LnDB.Update(func(btx *bolt.Tx) error {
		_, err := btx.CreateBucketIfNotExists(BKTPeers)
		if err != nil {
			return err
		}

		_, err = btx.CreateBucketIfNotExists(BKTWatch)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
