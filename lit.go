package main

import (
	"bufio"
	"flag"
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
	hardHeight = 1038000 // height to start at if not specified
)

var (
	//	Params = &chaincfg.TestNet3Params
	//	Params = &chaincfg.RegressionNetParams
	SCon uspv.SPVCon // global here for now

	LNode qln.LnNode

	GlobalConf LitConfig
)

// variables for a goodelivery session
type LitConfig struct {
	spvHost    string
	regTest    bool // flag to set mainnet
	birthblock int32

	Params *chaincfg.Params
}

func setConfig(lc *LitConfig) {
	spvhostptr := flag.String("spv", "", "full node to connect to")
	regtestptr := flag.Bool("reg", false, "use regtest (not testnet3)")
	birthptr := flag.Int("tip", hardHeight, "height to begin db sync")

	flag.Parse()

	lc.spvHost = *spvhostptr
	lc.regTest = *regtestptr
	lc.birthblock = int32(*birthptr)

	if lc.spvHost == "" {
		lc.spvHost = "lit3.co"
	}

	if lc.regTest {
		lc.Params = &chaincfg.RegressionNetParams
		if !strings.Contains(lc.spvHost, ":") {
			lc.spvHost = lc.spvHost + ":18444"
		}
	} else {
		lc.Params = &chaincfg.TestNet3Params
		if !strings.Contains(lc.spvHost, ":") {
			lc.spvHost = lc.spvHost + ":18333"
		}
	}
}

func main() {
	fmt.Printf("lit spv shell v0.0\n")

	fmt.Printf("-spv hostname to specify node to connect to.\n")
	fmt.Printf("-reg for regtest mode instead of testnet3\n")

	conf := new(LitConfig)
	setConfig(conf)

	// read key file (generate if not found)
	rootPriv, err := uspv.ReadKeyFileToECPriv(keyFileName, conf.Params)
	if err != nil {
		log.Fatal(err)
	}
	// setup TxStore first (before spvcon)
	Store := uspv.NewTxStore(rootPriv, conf.Params)
	// setup spvCon

	SCon, err = uspv.OpenSPV(
		conf.spvHost, headerFileName, utxodbFileName, &Store, true, false, conf.Params)
	if err != nil {
		log.Printf("can't connect: %s", err.Error())
		log.Fatal(err) // back to fatal when can't connect
	}

	tip, err := SCon.TS.GetDBSyncHeight() // ask for sync height
	if err != nil {
		log.Fatal(err)
	}
	if tip == 0 || conf.birthblock != hardHeight { // DB has never been used, set to birthday
		if conf.regTest {
			tip = 10 // for regtest
		} else {
			tip = conf.birthblock // for testnet3. should base on keyfile date?
		}
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
