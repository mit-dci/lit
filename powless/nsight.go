package powless

// GetAdrTxos
import (
	"bytes"
	"time"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/mit-dci/lit/wire"
	"github.com/mit-dci/lit/lnutil"
)

// ARGHGH all fields have to be exported (caps) or the json unmarshaller won't
// populate them !
type AdrUtxoResponse struct {
	Txid     string
	Height   int32
	Satoshis int64
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

// ...use insight api.  at least that's open source, can run yourself, seems to have
// some dev activity behind it.
func (a *APILink) NsightGetAdrTxos() error {

	apitxourl := "https://test-insight.bitpay.com/api"
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

	client := &http.Client{
		Timeout: time.Second * 5, // 5s to accomodate the 10s RPC timeout
	}
	response, err := client.Get(apitxourl + "/addrs/" + adrlist)
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

		log.Printf("tx %s at height %d\n", txah.Tx.TxHash().String(), txah.Height)
		a.TxUpToWallit <- txah

		// don't know what order we get these in, so update APILink height at the end
		// I think it's OK to do this?  Seems OK but haven't seen this use of defer()
		defer a.UpdateHeight(adrUtxo.Height)
	}

	return nil
}

func (a *APILink) GetOPTxs() error {
	apitxourl := "https://test-insight.bitpay.com/api/"

	var oplist []wire.OutPoint

	// copy registered ops here to minimize time mutex is locked
	a.TrackingOPsMtx.Lock()
	for op, _ := range a.TrackingOPs {
		oplist = append(oplist, op)
	}
	a.TrackingOPsMtx.Unlock()

	// need to query each txid with a different http request
	for _, op := range oplist {
		log.Printf("asking for %s\n", op.String())
		// get full tx info for the outpoint's tx
		// (if we have 2 outpoints with the same txid we query twice...)
		client := &http.Client{
			Timeout: time.Second * 5, // 5s to accomodate the 10s RPC timeout
		}
		response, err := client.Get(apitxourl + "tx/" + op.Hash.String())
		if err != nil {
			return err
		}

		var txr TxResponse
		// parse the response to get the spending txid
		err = json.NewDecoder(response.Body).Decode(&txr)
		if err != nil {
			log.Printf("json decode error; op %s not found\n", op.String())
			continue
		}

		// what is the "v" for here?
		for _, txout := range txr.Vout {
			if op.Index == txout.N { // hit; request this outpoint's spend tx
				// see if it's been spent
				if txout.SpentTxId == "" {
					log.Printf("%s has nil spenttxid\n", op.String())
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

// GetRawTx is a helper function to get a tx from the insight api
func GetRawTx(txid string) (*wire.MsgTx, error) {
	rawTxURL := "https://test-insight.bitpay.com/api/rawtx/"
	client := &http.Client{
		Timeout: time.Second * 5, // 5s to accomodate the 10s RPC timeout
	}
	response, err := client.Get(rawTxURL + txid)
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

(put in other file?)

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

	log.Printf("tx hex string is %s\n", txHexString)

	apiurl := "https://testnet-api.smartbit.com.au/v1/blockchain/pushtx"
	response, err := http.Post(
		apiurl, "application/json", bytes.NewBuffer([]byte(txHexString)))
	log.Printf("respo	nse: %s", response.Status)
	_, err = io.Copy(os.Stdout, response.Body)

	return err
}
