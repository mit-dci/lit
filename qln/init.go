package qln

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
)

func (nd *LitNode) Init(
	dbfilename, watchname string, basewal UWallet, tower bool) error {

	err := nd.OpenDB(dbfilename)
	if err != nil {
		return err
	}

	// connect to base wallet
	nd.BaseWallet = basewal

	nd.Param = nd.BaseWallet.Params()
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

	// make maps and channels

	nd.InProg = new(InFlightFund)
	nd.InProg.done = make(chan uint32, 1)

	nd.RemoteCons = make(map[uint32]*lndc.LNDConn)

	nd.PushClear = make(map[chainhash.Hash]chan bool)

	nd.OmniOut = make(chan *lnutil.LitMsg, 10)
	nd.OmniIn = make(chan *lnutil.LitMsg, 10)
	go nd.OmniHandler()
	go nd.OutMessager()

	return nil
}

// relay txs from the watchtower to the underlying wallet...
// small, but a little ugly; maybe there's a cleaner way
func (nd *LitNode) Relay(outbox chan *wire.MsgTx) {
	for {
		err := nd.BaseWallet.PushTx(<-outbox)
		if err != nil {
			fmt.Printf("PushTx error: %s", err.Error())
		}
	}
}

// Opens the DB file for the LnNode
func (nd *LitNode) OpenDB(filename string) error {
	var err error

	nd.LitDB, err = bolt.Open(filename, 0644, nil)
	if err != nil {
		return err
	}
	// create buckets if they're not already there
	err = nd.LitDB.Update(func(btx *bolt.Tx) error {
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
