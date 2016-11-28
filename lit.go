package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"

	"github.com/mit-dci/lit/qln"
	"github.com/mit-dci/lit/uspv"
)

const (
	keyFileName     = "testkey.hex"
	headerFileName  = "headers.bin"
	utxodbFileName  = "utxo.db"
	lndbFileName    = "ln.db"
	watchdbFileName = "watch.db"
	sorcFileName    = "sorc.db"
	// this is my local testnet node, replace it with your own close by.
	// Random internet testnet nodes usually work but sometimes don't, so
	// maybe I should test against different versions out there.
	//	SPVHostAdr = "slab.lan:18333" // for testnet3
	SPVHostAdr = "slab.lan:18444" // for regtest
)

var (
	//	Params = &chaincfg.TestNet3Params
	Params = &chaincfg.RegressionNetParams
	SCon   uspv.SPVCon // global here for now

	LNode qln.LnNode
)

func main() {
	fmt.Printf("lit spv shell v0.0\n")

	// read key file (generate if not found)
	rootPriv, err := uspv.ReadKeyFileToECPriv(keyFileName, Params)
	if err != nil {
		log.Fatal(err)
	}
	// setup TxStore first (before spvcon)
	Store := uspv.NewTxStore(rootPriv, Params)
	// setup spvCon

	SCon, err = uspv.OpenSPV(
		SPVHostAdr, headerFileName, utxodbFileName, &Store, false, false, Params)
	if err != nil {
		log.Printf("can't connect: %s", err.Error())
		log.Fatal(err) // back to fatal when can't connect
	}

	tip, err := SCon.TS.GetDBSyncHeight() // ask for sync height
	if err != nil {
		log.Fatal(err)
	}
	if tip == 0 { // DB has never been used, set to birthday
		tip = 10 // for regtest
		//		tip = 1034500 // for testnet3. hardcoded; later base on keyfile date?
		err = SCon.TS.SetDBSyncHeight(tip)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = LNode.Init(lndbFileName, watchdbFileName, &SCon)
	if err != nil {
		log.Fatal(err)
	}

	// once we're connected, initiate headers sync
	err = SCon.AskForHeaders()
	if err != nil {
		log.Fatal(err)
	}

	err = rpcShellListen()
	if err != nil {
		log.Printf(err.Error())
	}
	// main shell loop
	for {
		// setup reader with max 4K input chars
		reader := bufio.NewReaderSize(os.Stdin, 4000)
		fmt.Printf("LND# ")                 // prompt
		msg, err := reader.ReadString('\n') // input finishes on enter key
		if err != nil {
			log.Fatal(err)
		}

		cmdslice := strings.Fields(msg) // chop input up on whitespace
		if len(cmdslice) < 1 {
			continue // no input, just prompt again
		}
		fmt.Printf("entered command: %s\n", msg) // immediate feedback
		err = Shellparse(cmdslice)
		if err != nil { // only error should be user exit
			log.Fatal(err)
		}
	}

	return
}
