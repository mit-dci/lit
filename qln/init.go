package qln

import (
	"fmt"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/wallit"
)

// Init starts up a lit node.  Needs priv key, and some paths.
// Right now the SubWallet is hardcoded but can be an arg.
// Also only 1 param arg, but can break that out later.
func NewLitNode(privKey *[32]byte, path string,
	p *chaincfg.Params, tower bool) (*LitNode, error) {

	nd := new(LitNode)

	litdbpath := filepath.Join(path, "ln.db")
	err := nd.OpenDB(litdbpath)
	if err != nil {
		return nil, err
	}

	rootpriv, err := hdkeychain.NewMaster(privKey[:], p)
	if err != nil {
		return nil, err
	}
	// make a base wallet
	wallit := wallit.NewWallit(rootpriv, path, p)

	// connect to base wallet
	nd.SubWallet = wallit

	// ask basewallet for outpoint event messages
	go nd.OPEventHandler(nd.SubWallet.LetMeKnow())
	// optional tower activation
	if tower {
		watchname := filepath.Join(path, "watch.db")
		err = nd.Tower.OpenDB(watchname)
		if err != nil {
			return nil, err
		}
		nd.Tower.Accepting = true
		// call base wallet blockmonitor and hand this channel to the tower
		go nd.Tower.BlockHandler(nd.SubWallet.BlockMonitor())
		go nd.Relay(nd.Tower.JusticeOutbox())
	}

	// make maps and channels
	nd.UserMessageBox = make(chan string, 32)

	nd.InProg = new(InFlightFund)
	nd.InProg.done = make(chan uint32, 1)

	nd.RemoteCons = make(map[uint32]*RemotePeer)

	nd.OmniOut = make(chan *lnutil.LitMsg, 10)
	nd.OmniIn = make(chan *lnutil.LitMsg, 10)
	//	go nd.OmniHandler()
	go nd.OutMessager()

	return nd, nil
}

// relay txs from the watchtower to the underlying wallet...
// small, but a little ugly; maybe there's a cleaner way
func (nd *LitNode) Relay(outbox chan *wire.MsgTx) {
	for {
		err := nd.SubWallet.PushTx(<-outbox)
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
		_, err := btx.CreateBucketIfNotExists(BKTChannel)
		if err != nil {
			return err
		}

		_, err = btx.CreateBucketIfNotExists(BKTPeers)
		if err != nil {
			return err
		}

		_, err = btx.CreateBucketIfNotExists(BKTChanMap)
		if err != nil {
			return err
		}
		_, err = btx.CreateBucketIfNotExists(BKTPeerMap)
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
