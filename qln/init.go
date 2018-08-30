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
	"github.com/mit-dci/lit/invoice"
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

	// Create a new manager for the discreet log contracts
	nd.InvoiceManager, err = invoice.NewManager(filepath.Join(nd.LitFolder, "invoice.db"))
	if err != nil {
		return nil, err
	}

	// test stuff for invoices, delete this once done
	var test lnutil.InvoiceReplyMsg
	test.PeerIdx = uint32(60000)
	test.Id = "3"
	test.CoinType = "bcrt"
	test.Amount = 90000
	err = nd.InvoiceManager.SaveGeneratedInvoice(&test)
	if err != nil {
		log.Println("ERROR WHILE TRYING TO SAVE INVOICE", err)
		return nd, fmt.Errorf("ERROR WHILE TRYING TO SAVE INVOICE")
	}
	x, err := nd.InvoiceManager.LoadGeneratedInvoice("3")
	if err != nil {
		return nd, fmt.Errorf("ERROR WHILE TRYING OT READ INVOICE")
	}
	log.Println("X0",  x.PeerIdx, x.Id, x.CoinType, x.Amount,)

	// 	PeerIdx uint32
	// 	Id      string
	var test2 lnutil.InvoiceMsg
	test2.PeerIdx = uint32(2)
	test2.Id = "q"
	err = nd.InvoiceManager.SaveRepliedInvoice(&test2) // we replied to someone else's request
	if err != nil {
		log.Println("ERROR WHILE TRYING TO LOAD INVOICEMSG")
		return nd, fmt.Errorf("ERROR WHILE TRYING TO LOAD INVOICEMSG")
	}
	x1, err := nd.InvoiceManager.LoadRepliedInvoice("2")
	// search for all invoices belonging to this peer
	// might have to execute multiple times?
	if err != nil {
		log.Println("ERROR WHILE TRYING TO LOAD REPLIED INVOICES")
		return nd, fmt.Errorf("ERROR WHILE TRYING TO LOAD REPLIED INVOICES")
	}
	log.Println("X1", x1.PeerIdx, x1.Id)

	err = nd.InvoiceManager.SaveRequestedInvoice(&test)
	if err != nil {
		log.Println("FAILED WHILE STORING AN INVOICE SENT OUT TO ANOTHER PEER")
		return nd, fmt.Errorf("FAILED WHILE STORING AN INVOICE SENT OUT TO ANOTHER PEER")
	}
	x2, err := nd.InvoiceManager.LoadRequestedInvoice(60000, "3") //pass peerIdx, invoiceId
	if err != nil {
		log.Println("FAILED WHILE STORING AN INVOICE SENT OUT TO ANOTHER PEER")
		return nd, fmt.Errorf("FAILED WHILE STORING AN INVOICE SENT OUT TO ANOTHER PEER")
	}
	log.Println("X2", x2)

	err = nd.InvoiceManager.SavePendingInvoice(&test)
	if err != nil {
		log.Println("FAILED TO STORE PENDING INVOICES")
		return nd, fmt.Errorf("FAILED TO STORE PENDING INVOICES")
	}
	x2, err = nd.InvoiceManager.LoadPendingInvoice(60000, "3") //pass peerIdx, invoiceId
	if err != nil {
		log.Println("FAILED WHILE RETRIEVING A PENDING INVOICE")
		return nd, fmt.Errorf("FAILED WHILE RETRIEVING A PENDING INVOICE")
	}
	log.Println("X3", x2)

	err = nd.InvoiceManager.SavePaidInvoice(&test)
	if err != nil {
		log.Println("FAILED TO STORE PAID INVOICE IN DB")
		return nd, fmt.Errorf("FAILED TO STORE PAID INVOICE IN DB")
	}
	x2, err = nd.InvoiceManager.LoadPaidInvoice(60000, "3") //pass peerIdx, invoiceId
	if err != nil {
		log.Println("FAILED WHILE RETRIEVING A PENDING INVOICE")
		return nd, fmt.Errorf("FAILED WHILE RETRIEVING A PENDING INVOICE")
	}
	log.Println("X4", x2)
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

		_, err = btx.CreateBucketIfNotExists(BKTHTLCOPs)
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
