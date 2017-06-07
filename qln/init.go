package qln

import (
	"fmt"
	"path/filepath"

	"github.com/adiabat/btcd/wire"
	"github.com/adiabat/btcutil/hdkeychain"
	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/wallit"
)

// Init starts up a lit node.  Needs priv key, and a path.
// Does not activate a subwallet; do that after init.
func NewLitNode(privKey *[32]byte, path string, tower bool) (*LitNode, error) {

	nd := new(LitNode)
	nd.LitFolder = path

	litdbpath := filepath.Join(nd.LitFolder, "ln.db")
	err := nd.OpenDB(litdbpath)
	if err != nil {
		return nil, err
	}

	// Maybe make a new parameter set for "LN".. meh
	// TODO change this to a non-coin
	rootPrivKey, err := hdkeychain.NewMaster(privKey[:], &coinparam.TestNet3Params)
	if err != nil {
		return nil, err
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = 513 | 1<<31
	kg.Step[2] = 9 | 1<<31
	kg.Step[3] = 0 | 1<<31
	kg.Step[4] = 0 | 1<<31
	nd.IdentityKey, err = kg.DerivePrivateKey(rootPrivKey)
	if err != nil {
		return nil, err
	}

	// optional tower activation
	// towers are turned on when wallets are linked
	if tower {
		//		watchname := filepath.Join(nd.LitFolder, "watch.db")
		//		err = nd.Tower.OpenDB(watchname)
		//		if err != nil {
		//			return nil, err
		//		}
		//		nd.Tower.Accepting = true
		// call base wallet blockmonitor and hand this channel to the tower
		//		go nd.Tower.BlockHandler(nd.SubWallet.BlockMonitor())
		//		go nd.Relay(nd.Tower.JusticeOutbox())
	}

	// make maps and channels
	nd.UserMessageBox = make(chan string, 32)

	nd.InProg = new(InFlightFund)
	nd.InProg.done = make(chan uint32, 1)

	nd.RemoteCons = make(map[uint32]*RemotePeer)

	nd.SubWallet = make(map[uint32]UWallet)

	nd.OmniOut = make(chan lnutil.LitMsg, 10)
	nd.OmniIn = make(chan lnutil.LitMsg, 10)
	//	go nd.OmniHandler()
	go nd.OutMessager()

	return nd, nil
}

// LinkBaseWallet activates a wallet and hooks it into the litnode.
func (nd *LitNode) LinkBaseWallet(
	privKey *[32]byte, birthHeight int32, resync bool,
	host string, param *coinparam.Params) error {

	rootpriv, err := hdkeychain.NewMaster(privKey[:], param)
	if err != nil {
		return err
	}

	WallitIdx := param.HDCoinType

	// see if we've already attached a wallet for this coin type
	if nd.SubWallet[WallitIdx] != nil {
		return fmt.Errorf("coin type %d already linked", WallitIdx)
	}

	// see if there are other wallets already linked
	if len(nd.SubWallet) != 0 {
		// there are; assert multiwallet (may already be asserted)
		nd.MultiWallet = true
	}

	// if there aren't, Multiwallet will still be false; set new wallit to
	// be the first & default
	nd.SubWallet[WallitIdx] = wallit.NewWallit(
		rootpriv, birthHeight, resync, host, nd.LitFolder, param)

	go nd.OPEventHandler(nd.SubWallet[WallitIdx].LetMeKnow())

	if !nd.MultiWallet {
		nd.DefaultCoin = param.HDCoinType
	}

	// if this node is running a watchtower, link the watchtower to the
	// new wallet block events
	//	go nd.Tower.BlockHandler()

	return nil
}

// relay txs from the watchtower to the underlying wallet...
// small, but a little ugly; maybe there's a cleaner way
func (nd *LitNode) Relay(outbox chan *wire.MsgTx) {
	for {
		// TODO add watchtower coin type stuff
		//		err := nd.SubWallet.PushTx(<-outbox)
		//		if err != nil {
		//			fmt.Printf("PushTx error: %s", err.Error())
		//		}
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
