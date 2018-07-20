package powless

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/wire"
	"golang.org/x/net/proxy"
)

// powless is a couple steps below uspv in that it doesn't check
// proof of work.  It just asks some web API thing about if it has
// received money or not.

/*
Powless is a package to hook up a lit wallit to a block explorer.
The idea is that it's proof-of-work-less, in that there isn't any, which is
not great :(

Here's the API we're building for
https://github.com/gertjaap/blockchain-indexer

Soon we'll try to put in some merkle proofs and header chains, so there will be
less PoW, but still some, and the name still works.  That's forwards-compatibility!

Basic way to works is that powless implements the chainhook interface: a wallit
starts a powless with Start() and then can register addresses and outpoints
with the register() calls.  Powless can respond via 2 channels, the
TxUpToWallit channel and the CurrentHeightChan, which send transactions and
heights respectively.

The way it gets the data is from the indexer API calls, addressTxosSince and
outpointSpend.  addressTxosSince gives us new coins, and outpointSpend takes
them away.

Next up: verify proof of work.  Have all (? or just addressTxosSince?) return
merkle branches up to a header, and then a few headers after that (maybe 10?)
Then also have a hardcoded minimum block work to compare against.  The powless
client will then verify the branch up to the header, and the short chain of
headers with work greater than minWork.  This is weaker than SPV, but does
make it so the indexer has to do a bunch of work in order to send an invalid
transaction.

*/

/*
implement this:

ChainHook interface

	Start(height int32, host, path string, params *chaincfg.Params) (
		chan lnutil.TxAndHeight, chan int32, error)

	RegisterAddress(address [20]byte) error
	RegisterOutPoint(wire.OutPoint) error

	PushTx(tx *wire.MsgTx) error

	RawBlocks() chan *wire.MsgBlock

*/

// APILink is a link to a web API that can tell you about blockchain data.
type APILink struct {
	apiUrl   string
	proxyURL string

	// TrackingAdrs and OPs are slices of addresses and outpoints to watch for.
	// Using struct{} saves a byte of RAM but is ugly so I'll use bool.
	TrackingAdrs    map[[20]byte]bool
	TrackingAdrsMtx sync.Mutex

	TrackingOPs    map[wire.OutPoint]bool
	TrackingOPsMtx sync.Mutex

	TxUpToWallit chan lnutil.TxAndHeight

	CurrentHeightChan chan int32

	// we've "synced" up to this height; older txs won't get pushed up to wallit
	height int32

	// this is the hash on the tip of the chain; if it changes, we need to update
	tipBlockHash string

	// time based polling
	dirtyChan chan interface{}

	client http.Client

	p *coinparam.Params
}

// Start starts the APIlink
func (a *APILink) Start(
	startHeight int32, host, path string, proxyURL string, params *coinparam.Params) (
	chan lnutil.TxAndHeight, chan int32, error) {

	a.proxyURL = proxyURL

	if proxyURL != "" {
		dialer, err := proxy.SOCKS5("tcp", proxyURL, nil, proxy.Direct)
		if err != nil {
			return nil, nil, err
		}

		a.client.Transport = &http.Transport{
			Dial: dialer.Dial,
		}
	}

	// later, use params to detect which api to connect to
	a.p = params

	a.TrackingAdrs = make(map[[20]byte]bool)
	a.TrackingOPs = make(map[wire.OutPoint]bool)

	a.TxUpToWallit = make(chan lnutil.TxAndHeight, 1)
	a.CurrentHeightChan = make(chan int32, 1)

	a.dirtyChan = make(chan interface{}, 100)

	a.height = startHeight
	a.apiUrl = host

	go a.DirtyCheckLoop()
	go a.TipRefreshLoop()

	return a.TxUpToWallit, a.CurrentHeightChan, nil
}

// PushTx for indexer
func (a *APILink) PushTx(tx *wire.MsgTx) error {
	if tx == nil {
		return fmt.Errorf("tx is nil")
	}
	var b bytes.Buffer

	err := tx.Serialize(&b)
	if err != nil {
		return err
	}

	txHexString := fmt.Sprintf("%x", b.Bytes())

	// guess I just put the bytes as the body...?

	apiurl := a.apiUrl + "sendRawTransaction"

	response, err :=
		a.client.Post(apiurl, "text/plain", bytes.NewBuffer([]byte(txHexString)))
	if err != nil {
		return err
	}
	log.Printf("respo	nse: %s", response.Status)
	_, err = io.Copy(os.Stdout, response.Body)

	return err
}

// RegisterAddress gets a 20 byte address from the wallit and starts
// watching for utxos at that address.
func (a *APILink) RegisterAddress(adr160 [20]byte) error {
	log.Printf("register %x\n", adr160)
	a.TrackingAdrsMtx.Lock()
	a.TrackingAdrs[adr160] = true
	a.TrackingAdrsMtx.Unlock()
	a.dirtyChan <- nil
	log.Printf("Register %x complete\n", adr160)

	return nil
}

// RegisterOutPoint gets an outpoint from the wallit and starts looking
// for txins that spend it.
func (a *APILink) RegisterOutPoint(op wire.OutPoint) error {
	log.Printf("register %s\n", op.String())
	a.TrackingOPsMtx.Lock()
	a.TrackingOPs[op] = true
	a.TrackingOPsMtx.Unlock()

	a.dirtyChan <- nil
	log.Printf("Register %s complete\n", op.String())
	return nil
}

// UnregisterOutPoint stops watching an outpoint for spends.
func (a *APILink) UnregisterOutPoint(op wire.OutPoint) error {
	log.Printf("unregister %s\n", op.String())
	a.TrackingOPsMtx.Lock()
	delete(a.TrackingOPs, op)
	a.TrackingOPsMtx.Unlock()

	a.dirtyChan <- nil
	log.Printf("Unregister %s complete\n", op.String())
	return nil
}

// DirtyCheckLoop checks with the server once things have changed on the client end.
// this is actually a bit ugly because it checks *everything* when *anything* has
// changed.  It could be much more efficient if, eg it checks for a newly created
// address by itself.
func (a *APILink) DirtyCheckLoop() {

	for {
		// wait here until something marks the state as dirty
		log.Printf("Waiting for dirt...\n")
		<-a.dirtyChan

		log.Printf("Dirt detected\n")

		err := a.GetVAdrTxos()
		if err != nil {
			log.Printf(err.Error())
		}
		err = a.GetVOPTxs()
		if err != nil {
			log.Printf(err.Error())
		}

		// probably clean, empty it out to prevent cascades
		for len(a.dirtyChan) > 0 {
			<-a.dirtyChan
		}
	}

	return
}

// VBlocksResponse is a list of Vblocks, which comes back from the /blocks
// query to the indexer
type VBlocksResponse []VBlock

// VBlock is the json data that comes back from the /blocks query to the indexer
type VBlock struct {
	Height   int32
	Hash     string
	Time     int64
	TxLength int32
	Size     int32
	PoolInfo string
}

// TipRefreshLoop checks for a change in the highest block hash
// if so it sets the dirty flag to true to trigger a refresh
func (a *APILink) TipRefreshLoop() error {
	for {
		// Fetch the current highest block
		apiurl := a.apiUrl + "blocks?limit=1"

		response, err := a.client.Get(apiurl)
		if err != nil {
			return err
		}

		var blockjsons VBlocksResponse
		err = json.NewDecoder(response.Body).Decode(&blockjsons)
		if err != nil {
			return err
		}

		// only height needed here?
		if blockjsons[0].Hash != a.tipBlockHash &&
			blockjsons[0].Height > a.height {

			a.tipBlockHash = blockjsons[0].Hash
			a.UpdateHeight(blockjsons[0].Height)
			a.dirtyChan <- nil
		}

		log.Printf("blockchain tip %v\n", a.tipBlockHash)

		time.Sleep(time.Second * 60)
	}

	return nil
}

// do you even need a struct here..?
type RawTxResponse struct {
	RawTx string
}

type VRawResponse []VRawTx

type VRawTx struct {
	Height  int32
	Spender string
	Tx      string
}

// GetVAdrTxos gets new utxos for the wallet from the indexer.
func (a *APILink) GetVAdrTxos() error {

	apitxourl := a.apiUrl + "addressTxosSince/"

	var urls []string
	a.TrackingAdrsMtx.Lock()
	for adr160, _ := range a.TrackingAdrs {
		// make the bech32 segwit address
		adrBch, err := bech32.SegWitV0Encode(a.p.Bech32Prefix, adr160[:])
		if err != nil {
			return err
		}
		// make the old base58 address
		adr58 := lnutil.OldAddressFromPKH(adr160, a.p.PubKeyHashAddrID)

		// make request URLs for both
		urls = append(urls,
			fmt.Sprintf("%s%d/%s%s", apitxourl, a.height, adrBch, "?raw=1"))
		urls = append(urls,
			fmt.Sprintf("%s%d/%s%s", apitxourl, a.height, adr58, "?raw=1"))
	}
	a.TrackingAdrsMtx.Unlock()

	log.Printf("have %d adr urls to check\n", len(urls))

	// make an API call for every adr in adrs
	// then grab the tx hex, decode and send up to the wallit
	for _, url := range urls {
		log.Printf("Requesting adr %s\n", url)
		response, err := a.client.Get(url)
		if err != nil {
			return err
		}

		//		bd, err := ioutil.ReadAll(response.Body)
		//		if err != nil {
		//			return err
		//		}

		//		log.Printf(string(bd))

		var txojsons VRawResponse

		err = json.NewDecoder(response.Body).Decode(&txojsons)
		if err != nil {
			return err
		}

		for _, txjson := range txojsons {
			// if there's some text in the spender field, skip this as it's already gone
			// also, really, this shouldn't be returned by the API at all because
			// who cares how much money we used to have.
			if len(txjson.Spender) > 32 {
				continue
			}
			txBytes, err := hex.DecodeString(txjson.Tx)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(txBytes)
			tx := wire.NewMsgTx()
			err = tx.Deserialize(buf)
			if err != nil {
				return err
			}

			var txah lnutil.TxAndHeight
			txah.Height = int32(txjson.Height)
			txah.Tx = tx

			log.Printf("tx %s at height %d", txah.Tx.TxHash().String(), txah.Height)
			// send the tx and height back up to the wallit
			a.TxUpToWallit <- txah
			log.Printf("sent\n")
		}
	}
	log.Printf("GetVAdrTxos complete\n")
	return nil
}

// VSpendResponse is the JSON response from the / outpointSpend call
type VSpendResponse struct {
	Error      bool
	Spender    string
	SpenderRaw string
	Spent      bool
}

func (a *APILink) GetVOPTxs() error {
	apitxourl := a.apiUrl + "outpointSpend/"

	var oplist []wire.OutPoint

	// copy registered ops here to minimize time mutex is locked
	a.TrackingOPsMtx.Lock()
	for op, checking := range a.TrackingOPs {
		if checking {
			oplist = append(oplist, op)
		}
	}
	a.TrackingOPsMtx.Unlock()

	// need to query each txid with a different http request
	for _, op := range oplist {
		log.Printf("asking for %s\n", op.String())
		// get full tx info for the outpoint's tx
		// (if we have 2 outpoints with the same txid we query twice...)
		opstring := op.String()
		opstring = strings.Replace(opstring, ";", "/", 1)
		response, err := a.client.Get(apitxourl + opstring + "?raw=1")
		if err != nil {
			return err
		}

		var txr VSpendResponse
		// parse the response to get the spending txid
		err = json.NewDecoder(response.Body).Decode(&txr)
		if err != nil || txr.Error {
			log.Printf("json decode error; op %s not found\n", op.String())
			continue
		}

		// see if this utxo is spent
		if txr.Spent {

			// if so, decode the tx indicated and give it to the wallit.
			txBytes, err := hex.DecodeString(txr.SpenderRaw)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(txBytes)
			tx := wire.NewMsgTx()
			err = tx.Deserialize(buf)
			if err != nil {
				return err
			}

			var txah lnutil.TxAndHeight
			txah.Tx = tx
			// don't know height from returned data, assume it's current height
			// which could be wrong, gotta fix this.
			// TODO
			txah.Height = a.height
			a.TxUpToWallit <- txah

			// assume you no longer need to monitor this outpoint,
			// because it's gone, and you just told the wallit how it disappeared
			a.TrackingOPsMtx.Lock()
			// mark this outpoint as not checked.  It stays in ram though.
			a.TrackingOPs[op] = false
			a.TrackingOPsMtx.Unlock()
		}
		// don't need per-txout check here; the outpoint itself is spent
	}

	return nil
}

// VGetRawTx is a helper function to get a tx from the indexer
func (a *APILink) VGetRawTx(txid string) (*wire.MsgTx, error) {
	rawTxURL := a.apiUrl + "getTransaction/"
	response, err := a.client.Get(rawTxURL + txid)
	if err != nil {
		return nil, err
	}

	var rtx RawTxResponse

	err = json.NewDecoder(response.Body).Decode(&rtx)
	if err != nil {
		return nil, err
	}

	txBytes, err := hex.DecodeString(rtx.RawTx)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(txBytes)
	tx := wire.NewMsgTx()
	err = tx.Deserialize(buf)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (a *APILink) UpdateHeight(height int32) {
	// if it's an increment (note reorgs are still... not a thing yet)
	if height > a.height {
		// update internal height
		a.height = height
		// send that back up to the wallit
		a.CurrentHeightChan <- height
	}
}

func (a *APILink) RawBlocks() chan *wire.MsgBlock {
	// dummy channel for now
	return make(chan *wire.MsgBlock, 1)
}
