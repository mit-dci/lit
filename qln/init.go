package qln

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/db/lnbolt" // TODO Abstract this more.
	"github.com/mit-dci/lit/dlc"
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lnp2p"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/wallit"
	"github.com/mit-dci/lit/watchtower"
)

// NewLitNode starts up a lit node.  Needs priv key, and a path.
// Does not activate a subwallet; do that after init.
func NewLitNode(privKey *[32]byte, path string, trackerURL string, proxyURL string, nat string) (*LitNode, error) {

	var err error

	// Maybe make a new parameter set for "LN".. meh
	// TODO change this to a non-coin
	rootPrivKey, err := hdkeychain.NewMaster(privKey[:], &coinparam.TestNet3Params)
	if err != nil {
		return nil, err
	}

	nd := new(LitNode)
	nd.LitFolder = path

	litdbpath := filepath.Join(nd.LitFolder, "ln.db")
	err = nd.OpenDB(litdbpath)
	if err != nil {
		return nil, err
	}

	db2 := &lnbolt.LitBoltDB{}
	var dbx lncore.LitStorage
	dbx = db2
	err = dbx.Open(filepath.Join(nd.LitFolder, "db2"))
	if err != nil {
		return nil, err
	}
	nd.NewLitDB = dbx
	err = nd.NewLitDB.Check()
	if err != nil {
		return nil, err
	}

	// Event system setup.
	ebus := eventbus.NewEventBus()
	nd.Events = &ebus

	// Peer manager
	nd.PeerMan, err = lnp2p.NewPeerManager(rootPrivKey, nd.NewLitDB.GetPeerDB(), trackerURL, &ebus, nil) // TODO proxy/nat stuff
	if err != nil {
		return nil, err
	}

	// Actually start the thread to send messages.
	nd.PeerMan.StartSending()

	// Register adapter event handlers.  These are for hooking in the new peer management with the old one.
	h1 := makeTmpNewPeerHandler(nd)
	nd.Events.RegisterHandler("lnp2p.peer.new", h1)
	h2 := makeTmpDisconnectPeerHandler(nd)
	nd.Events.RegisterHandler("lnp2p.peer.disconnect", h2)

	// Sets up handlers for all the messages we need to handle.
	nd.registerHandlers()

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

	kg.Step[3] = 1 | 1<<31
	rcPriv, err := kg.DerivePrivateKey(rootPrivKey)
	nd.DefaultRemoteControlKey = rcPriv.PubKey()
	rcPriv = nil

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

	nd.InProgMultihop, err = nd.GetAllMultihopPayments()
	if err != nil {
		return nil, err
	}

	nd.RemoteMtx.Lock()
	nd.RemoteCons = make(map[uint32]*RemotePeer)
	nd.RemoteMtx.Unlock()

	nd.SubWallet = make(map[uint32]UWallet)

	// REFACTORING STUFF
	nd.PeerMap = map[*lnp2p.Peer]*RemotePeer{}
	nd.PeerMapMtx = &sync.Mutex{}

	pdb := nd.NewLitDB.GetPeerDB()
	addrs, err := pdb.GetPeerAddrs()
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		_, err = nd.PeerMan.TryConnectAddress(string(a), nil) // TODO Proxy/NAT
		if err != nil {
			logging.Warnf("init: tried to auto-connect to %s but failed: %s\n", a, err.Error())
		}
	}

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
		logging.Error(err)
		return err
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

		logging.Infof("Registering outpoint %v", qChan.PorTxo.Op)

		nd.SubWallet[WallitIdx].WatchThis(qChan.PorTxo.Op)
	}

	go nd.OPEventHandler(nd.SubWallet[WallitIdx].LetMeKnow())
	go nd.HeightEventHandler(nd.SubWallet[WallitIdx].LetMeKnowHeight())

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
		_, err := btx.CreateBucketIfNotExists(BKTChannelData)
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

		_, err = btx.CreateBucketIfNotExists(BKTHTLCOPs)
		if err != nil {
			return err
		}

		_, err = btx.CreateBucketIfNotExists(BKTPayments)
		if err != nil {
			return err
		}

		_, err = btx.CreateBucketIfNotExists(BKTRCAuth)
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
