package powless

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/adiabat/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
)

// powless is a couple steps below uspv in that it doesn't check
// proof of work.  It just asks some web API thing about if it has
// received money or not.

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
				fmt.Printf(err.Error())
			}
		} else {
			fmt.Printf("clean, sleep 5 sec\n")
			time.Sleep(time.Second * 5)
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

		response, err := http.Get(apitxourl + "/rawtx/" + ad.Txid)
		if err != nil {
			return err
		}

		var rtx RawTxResponse

		err = json.NewDecoder(response.Body).Decode(&rtx)
		if err != nil {
			return err
		}

		txBytes, err := hex.DecodeString(rtx.RawTx)
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
		txah.Height = int32(ad.Height)
		txah.Tx = tx

		fmt.Printf("tx %s at height %d\n", txah.Tx.TxHash().String(), txah.Height)

		a.TxUpToWallit <- txah

	}

	return nil
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

/*
func (a *APILink) GetAdrTxosSmartBit() error {

	apitxourl := "https://testnet-api.smartbit.com.au/v1/blockchain/address/"
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

	// chop off last comma
	adrlist = adrlist[:len(adrlist)-1] + "/unspent"

	response, err := http.Get(apitxourl + adrlist)
	if err != nil {
		return err
	}

	ar := new(AdrResponse)

	err = json.NewDecoder(response.Body).Decode(ar)
	if err != nil {
		return err
	}

	if !ar.Success {
		return fmt.Errorf("ar success = false...")
	}

	var txidlist string

	// go through all unspent txos.  All we want is the txids, to request the
	// full txs.
	for i, txo := range ar.Unspent {
		txidlist += txo.Txid + ","
	}
	txidlist = txidlist[:len(txidlist)-1] + "/hex"

	// now request all those txids
	// need to request twice! To find height.  Blah.
	apitxurl := "https://testnet-api.smartbit.com.au/v1/blockchain/tx/"

	response, err = http.Get(apitxurl + txidlist)
	if err != nil {
		return err
	}

	tr := new(TxResponse)

	err = json.NewDecoder(response.Body).Decode(tr)
	if err != nil {
		return err
	}

	if !tr.Success {
		return fmt.Errorf("tr success = false...")
	}

	//	chainhash.NewHashFromStr()

	for _, txjson := range tr.Hex {
		buf, err := hex.DecodeString(txjson.Hex)
		if err != nil {
			return err
		}
		buf := bytes.NewBuffer(buf)
		tx := wire.NewMsgTx()
		err = tx.Deserialize(buf)
		if err != nil {
			return err
		}
		//		a.TxUpToWallit
	}

	return nil
}
*/

/*
// for blockcypher thing.  these are all crummy.
func (a *APILink) GetAdrTxos() error {
	api := gobcy.API{"", "btc", "test3"}

	// make 1 request per address; insta-get the tx hex in the first call.

	a.TrackingAdrsMtx.Lock()
	defer a.TrackingAdrsMtx.Unlock()

	for adr160, _ := range a.TrackingAdrs {
		adr58, err := btcutil.NewAddressPubKeyHash(adr160[:], a.p)
		if err != nil {
			return err
		}
		fmt.Printf("making request for %s\n", adr58.String())
		adrResp, err := api.GetAddrFull(
			adr58.String(), nil) // map[string]string{"includeHex": "true"})
		if err != nil {
			fmt.Printf("got err %s\n", err.Error())
			return err
		}
		fmt.Printf("done addr %s\n", adrResp.Address)
		fmt.Printf("got %d txs\n", len(adrResp.TXs))

		for _, txResp := range adrResp.TXs {

			txBytes, err := hex.DecodeString(txResp.Hex)
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
			txah.Height = int32(txResp.BlockHeight)
			txah.Tx = tx
			a.TxUpToWallit <- txah
		}

	}
	return nil
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
