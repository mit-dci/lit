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
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
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
	// time based polling
	dirtybool bool

	p *chaincfg.Params
}

func (a *APILink) Start(
	startHeight int32, host, path string, params *chaincfg.Params) (
	chan lnutil.TxAndHeight, chan int32, error) {

	// later, use params to detect which api to connect to
	a.p = params

	a.TrackingAdrs = make(map[[20]byte]bool)
	a.TrackingOPs = make(map[wire.OutPoint]bool)

	a.TxUpToWallit = make(chan lnutil.TxAndHeight, 1)
	a.CurrentHeightChan = make(chan int32, 1)

	go a.ClockLoop()

	return a.TxUpToWallit, a.CurrentHeightChan, nil
}

func (a *APILink) ClockLoop() {

	for {
		if a.dirtybool {
			a.dirtybool = false
			err := a.GetAdrTxos()
			if err != nil {
				log.Printf(err.Error())
			}
			err = a.GetOPTxs()
			if err != nil {
				log.Printf(err.Error())
			}
		} else {
			fmt.Printf("clean, sleep 5 sec\n")
			time.Sleep(time.Second * 5)
			// some kind of long range refresh for blocks...?
		}
	}

	return
}

func (a *APILink) RegisterAddress(adr160 [20]byte) error {
	a.TrackingAdrsMtx.Lock()
	a.TrackingAdrs[adr160] = true
	a.TrackingAdrsMtx.Unlock()
	a.dirtybool = true
	return nil
}

func (a *APILink) RegisterOutPoint(op wire.OutPoint) error {
	a.TrackingOPsMtx.Lock()
	a.TrackingOPs[op] = true
	a.TrackingOPsMtx.Unlock()
	a.dirtybool = true
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

// use insight api.  at least that's open source, can run yourself, seems to have
// some dev activity behind it.
func (a *APILink) GetAdrTxos() error {

	apitxourl := "https://testnet.blockexplorer.com/api"
	// make a comma-separated list of base58 addresses
	var adrlist string

	a.TrackingAdrsMtx.Lock()
	for adr160, _ := range a.TrackingAdrs {
		adr58, err := btcutil.NewAddressPubKeyHash(adr160[:], a.p)
		if err != nil {
			return err
		}
		adrlist += adr58.String()
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
	for _, ad := range *ars {
		tx, err := GetRawTx(ad.Txid)
		if err != nil {
			return err
		}

		var txah lnutil.TxAndHeight
		txah.Height = int32(ad.Height)
		txah.Tx = tx

		fmt.Printf("tx %s at height %d\n", txah.Tx.TxHash().String(), txah.Height)
		a.TxUpToWallit <- txah
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
			return err
		}

		// what is the "v" for here?
		for _, txout := range txr.Vout {
			if op.Index == txout.N { // hit; request this one
				tx, err := GetRawTx(op.Hash.String())
				if err != nil {
					return err
				}

				var txah lnutil.TxAndHeight
				txah.Tx = tx
				txah.Height = txout.SpentHeight
				a.TxUpToWallit <- txah
			}
		}
	}

	return nil
}

type TxResponse struct {
	Vout []VoutJson
}

// Get txid of spending tx
type VoutJson struct {
	N           uint32
	SpentTxId   string
	SpentHeight int32
}

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

func (a *APILink) PushTx(tx *wire.MsgTx) error {
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
