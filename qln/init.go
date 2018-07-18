package qln

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/dlc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/wallit"
	"github.com/mit-dci/lit/watchtower"
)

// Init starts up a lit node.  Needs priv key, and a path.
// Does not activate a subwallet; do that after init.
func NewLitNode(privKey *[32]byte, path string, trackerURL string, proxyURL string, nat string) (*LitNode, error) {

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

	nd.TrackerURL = trackerURL

	nd.ProxyURL = proxyURL

	nd.Nat = nat

	nd.InitRouting()

	// optional tower activation

	nd.Tower = new(watchtower.WatchTower)

	// Create a new manager for the discreet log contracts
	nd.DlcManager, err = dlc.NewManager(filepath.Join(nd.LitFolder, "dlc.db"))
	if err != nil {
		return nil, err
	}

	// make maps and channels
	nd.UserMessageBox = make(chan string, 32)

	nd.InProg = new(InFlightFund)
	nd.InProg.done = make(chan uint32, 1)

	nd.InProgDual = new(InFlightDualFund)
	nd.InProgDual.done = make(chan *DualFundingResult, 1)

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
	privKey *[32]byte, birthHeight int32, resync bool, tower bool,
	host string, proxy string, param *coinparam.Params) error {

	rootpriv, err := hdkeychain.NewMaster(privKey[:], param)
	if err != nil {
		return err
	}

	WallitIdx := param.HDCoinType

	// see if we've already attached a wallet for this coin type
	if nd.SubWallet[WallitIdx] != nil {
		return fmt.Errorf("coin type %d already linked", WallitIdx)
	}
	// see if startheight is below allowed with coinparam
	if birthHeight < param.StartHeight {
		return fmt.Errorf("%s birth height give as %d, but parameters start at %d",
			param.Name, birthHeight, param.StartHeight)
	}

	// see if there are other wallets already linked
	if len(nd.SubWallet) != 0 {
		// there are; assert multiwallet (may already be asserted)
		nd.MultiWallet = true
	}

	// if there aren't, Multiwallet will still be false; set new wallit to
	// be the first & default
	var cointype int
	nd.SubWallet[WallitIdx], cointype, err = wallit.NewWallit(
		rootpriv, birthHeight, resync, host, nd.LitFolder, proxy, param)

	if err != nil {
		 log.Println(err)
		 return nil
	}

	if nd.ConnectedCoinTypes == nil {
		nd.ConnectedCoinTypes = make(map[uint32]bool)
		nd.ConnectedCoinTypes[uint32(cointype)] = true
	}
	nd.ConnectedCoinTypes[uint32(cointype)] = true
	// re-register channel addresses
	qChans, err := nd.GetAllQchans()
	if err != nil {
		return err
	}

	for _, qChan := range qChans {
		var pkh [20]byte
		pkhSlice := btcutil.Hash160(qChan.MyRefundPub[:])
		copy(pkh[:], pkhSlice)
		nd.SubWallet[WallitIdx].ExportHook().RegisterAddress(pkh)

		log.Printf("Registering outpoint %v", qChan.PorTxo.Op)

		nd.SubWallet[WallitIdx].WatchThis(qChan.PorTxo.Op)
	}

	go nd.OPEventHandler(nd.SubWallet[WallitIdx].LetMeKnow())

	if !nd.MultiWallet {
		nd.DefaultCoin = param.HDCoinType
	}

	// if this node is running a watchtower, link the watchtower to the
	// new wallet block events

	if tower {
		err = nd.Tower.HookLink(
			nd.LitFolder, param, nd.SubWallet[WallitIdx].ExportHook())
		if err != nil {
			return err
		}
	}

	return nil
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
