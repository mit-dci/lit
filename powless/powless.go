package powless

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/adiabat/bech32"
	"github.com/adiabat/btcd/wire"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
)

// powless is a couple steps below uspv in that it doesn't check
// proof of work.  It just asks some web API thing about if it has
// received money or not.

/*
Here are the inefficiencies of the block explorer model.
The 2 things you want to support are gaining and losing money.
Gaining is done by watching for new txs with outputs to an address.
Losing is done by watching for new txs with inputs from an outpoint.

Insight does allow us to query UTXOs for an address, but not above a
specified height.  So we end up re-downloading all the utxos we already know
about.  But we don't have to send those to the wallit.

Watching for outpoints is easier; they disappear.  We just keep checking the
tx of the outpoint and see if the outpoint index is spent.

To make an optimal web-explorer api, here are the api calls you'd want:

/utxoAboveHeight/[address]/[height]

returns the raw hex (and other json if you want) of all txs sending to [address]
which are confirmed at [height] or above

/outPointSpend/[txid]/[index]

returns either null, or the raw hex (& json if you want) of the transaction
spending outpoint [txid]:[index].

Those two calls get you basically everything you need for a wallet, in a pretty
efficient way.

(but yeah re-orgs and stuff, right?  None of this deals with that yet.  That's
going to be a pain.)

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
	apiCon net.Conn

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

	p *coinparam.Params
}

// Start starts the APIlink
func (a *APILink) Start(
	startHeight int32, host, path string, params *coinparam.Params) (
	chan lnutil.TxAndHeight, chan int32, error) {

	// later, use params to detect which api to connect to
	a.p = params

	a.TrackingAdrs = make(map[[20]byte]bool)
	a.TrackingOPs = make(map[wire.OutPoint]bool)

	a.TxUpToWallit = make(chan lnutil.TxAndHeight, 1)
	a.CurrentHeightChan = make(chan int32, 1)

	a.dirtyChan = make(chan interface{}, 100)

	a.height = startHeight

	go a.DirtyCheckLoop()
	go a.TipRefreshLoop()

	return a.TxUpToWallit, a.CurrentHeightChan, nil
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

		// probably clean, empty it out
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
		apihighestblockurl := "https://tvtc.blkidx.org/blocks?limit=1"
		response, err := http.Get(apihighestblockurl)
		if err != nil {
			return err
		}

		var blockjsons VBlocksResponse
		err = json.NewDecoder(response.Body).Decode(&blockjsons)
		if err != nil {
			return err
		}

		if blockjsons[0].Hash != a.tipBlockHash {
			a.tipBlockHash = blockjsons[0].Hash
			a.dirtyChan <- nil
		}

		fmt.Printf("blockchain tip %v\n", a.tipBlockHash)

		time.Sleep(time.Second * 60)
	}

	return nil
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

// ARGHGH all fields have to be exported (caps) or the json unmarshaller won't
// populate them !
type AdrUtxoResponse struct {
	Txid     string
	Height   int32
	Satoshis int64
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

	apitxourl := "https://tvtc.blkidx.org/addressTxosSince/"

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
		response, err := http.Get(url)
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

// GetAdrTxos
// ...use insight api.  at least that's open source, can run yourself, seems to have
// some dev activity behind it.
func (a *APILink) GetAdrTxos() error {

	apitxourl := "https://testnet.blockexplorer.com/api"
	// make a comma-separated list of base58 addresses
	var adrlist string

	a.TrackingAdrsMtx.Lock()
	for adr160, _ := range a.TrackingAdrs {
		adr58 := lnutil.OldAddressFromPKH(adr160, a.p.PubKeyHashAddrID)
		adrlist += adr58
		adrlist += ","
	}
	a.TrackingAdrsMtx.Unlock()

	// chop off last comma, and add /utxo
	adrlist = adrlist[:len(adrlist)-1] + "/utxo"

	response, err := http.Get(apitxourl + "/addrs/" + adrlist)
	if err != nil {
		return err
	}

	ars := new([]AdrUtxoResponse)

	err = json.NewDecoder(response.Body).Decode(ars)
	if err != nil {
		return err
	}

	if len(*ars) == 0 {
		return fmt.Errorf("no ars \n")
	}

	// go through txids, request hex tx, build txahdheight and send that up
	for _, adrUtxo := range *ars {

		// only request if higher than current 'sync' height
		if adrUtxo.Height < a.height {
			// skip this address; it's lower than we've already seen
			continue
		}

		tx, err := GetRawTx(adrUtxo.Txid)
		if err != nil {
			return err
		}

		var txah lnutil.TxAndHeight
		txah.Height = int32(adrUtxo.Height)
		txah.Tx = tx

		fmt.Printf("tx %s at height %d\n", txah.Tx.TxHash().String(), txah.Height)
		a.TxUpToWallit <- txah

		// don't know what order we get these in, so update APILink height at the end
		// I think it's OK to do this?  Seems OK but haven't seen this use of defer()
		defer a.UpdateHeight(adrUtxo.Height)
	}

	return nil
}

func (a *APILink) GetVOPTxs() error {
	apitxourl := "https://tvtc.blkidx.org/outpointSpend/"

	var oplist []wire.OutPoint

	// copy registered ops here to minimize time mutex is locked
	a.TrackingOPsMtx.Lock()
	for op, _ := range a.TrackingOPs {
		oplist = append(oplist, op)
	}
	a.TrackingOPsMtx.Unlock()

	// need to query each txid with a different http request
	for _, op := range oplist {
		fmt.Printf("asking for %s\n", op.String())
		// get full tx info for the outpoint's tx
		// (if we have 2 outpoints with the same txid we query twice...)
		opstring := op.String()
		opstring = strings.Replace(opstring, ";", "/", 1)
		response, err := http.Get(apitxourl + opstring)
		if err != nil {
			return err
		}

		var txr TxResponse
		// parse the response to get the spending txid
		err = json.NewDecoder(response.Body).Decode(&txr)
		if err != nil {
			fmt.Printf("json decode error; op %s not found\n", op.String())
			continue
		}

		// don't need per-txout check here; the outpoint itself is spent
	}

	return nil
}

func (a *APILink) GetOPTxs() error {
	apitxourl := "https://testnet.blockexplorer.com/api/"

	var oplist []wire.OutPoint

	// copy registered ops here to minimize time mutex is locked
	a.TrackingOPsMtx.Lock()
	for op, _ := range a.TrackingOPs {
		oplist = append(oplist, op)
	}
	a.TrackingOPsMtx.Unlock()

	// need to query each txid with a different http request
	for _, op := range oplist {
		fmt.Printf("asking for %s\n", op.String())
		// get full tx info for the outpoint's tx
		// (if we have 2 outpoints with the same txid we query twice...)
		response, err := http.Get(apitxourl + "tx/" + op.Hash.String())
		if err != nil {
			return err
		}

		var txr TxResponse
		// parse the response to get the spending txid
		err = json.NewDecoder(response.Body).Decode(&txr)
		if err != nil {
			fmt.Printf("json decode error; op %s not found\n", op.String())
			continue
		}

		// what is the "v" for here?
		for _, txout := range txr.Vout {
			if op.Index == txout.N { // hit; request this outpoint's spend tx
				// see if it's been spent
				if txout.SpentTxId == "" {
					fmt.Printf("%s has nil spenttxid\n", op.String())
					// this outpoint is not yet spent, can't request
					continue
				}

				tx, err := GetRawTx(txout.SpentTxId)
				if err != nil {
					return err
				}

				var txah lnutil.TxAndHeight
				txah.Tx = tx
				txah.Height = txout.SpentHeight
				a.TxUpToWallit <- txah

				a.UpdateHeight(txout.SpentHeight)
			}
		}

		// TODO -- REMOVE once there is any API support for segwit addresses.
		// This queries the outpoints we already have and re-downloads them to
		// update their height. (IF the height is non-zero)  It's a huge hack
		// and not scalable.  But allows segwit outputs to be confirmed now.
		//if txr.

		if txr.Blockheight > 1 {
			// this outpoint is confirmed; redownload it and send to wallit
			tx, err := GetRawTx(txr.Txid)
			if err != nil {
				return err
			}

			var txah lnutil.TxAndHeight
			txah.Tx = tx
			txah.Height = txr.Blockheight
			a.TxUpToWallit <- txah

			a.UpdateHeight(txr.Blockheight)
		}

		// TODO -- end REMOVE section

	}

	return nil
}

type TxResponse struct {
	Txid        string
	Blockheight int32
	Vout        []VoutJson
}

// Get txid of spending tx
type VoutJson struct {
	N           uint32
	SpentTxId   string
	SpentHeight int32
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

// GetRawTx is a helper function to get a tx from the insight api
func GetRawTx(txid string) (*wire.MsgTx, error) {
	rawTxURL := "https://testnet.blockexplorer.com/api/rawtx/"
	response, err := http.Get(rawTxURL + txid)
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

/* smartbit structs
type AdrResponse struct {
	Success bool
	//	Paging  interface{}
	Unspent []JsUtxo
}

type TxResponse struct {
	Success     bool
	Transaction []TxJson
}

type TxJson struct {
	Block int32
	Txid  string
}

type TxHexResponse struct {
	Success bool
	Hex     []TxHexString
}

type TxHexString struct {
	Txid string
	Hex  string
}

type JsUtxo struct {
	Value_int int64
	Txid      string
	N         uint32
	Addresses []string // why more than 1 ..? always 1.
}

*/

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

	apiurl := "https://tvtc.blkidx.org/sendRawTransaction"
	response, err :=
		http.Post(apiurl, "text/plain", bytes.NewBuffer([]byte(txHexString)))
	if err != nil {
		return err
	}
	fmt.Printf("respo	nse: %s", response.Status)
	_, err = io.Copy(os.Stdout, response.Body)

	return err
}

// PushTx pushes a tx to the network via the smartbit site / api
// smartbit supports segwit so
func (a *APILink) PushTxSmartBit(tx *wire.MsgTx) error {
	if tx == nil {
		return fmt.Errorf("tx is nil")
	}
	var b bytes.Buffer

	err := tx.Serialize(&b)
	if err != nil {
		return err
	}

	// turn into hex
	txHexString := fmt.Sprintf("{\"hex\": \"%x\"}", b.Bytes())

	fmt.Printf("tx hex string is %s\n", txHexString)

	apiurl := "https://testnet-api.smartbit.com.au/v1/blockchain/pushtx"
	response, err := http.Post(
		apiurl, "application/json", bytes.NewBuffer([]byte(txHexString)))
	fmt.Printf("respo	nse: %s", response.Status)
	_, err = io.Copy(os.Stdout, response.Body)

	return err
}

func (a *APILink) RawBlocks() chan *wire.MsgBlock {
	// dummy channel for now
	return make(chan *wire.MsgBlock, 1)
}
