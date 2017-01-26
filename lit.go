package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/qln"
	"github.com/mit-dci/lit/uspv"
)

const (
	litHomeDirName = ".lit"

	keyFileName     = "testkey.hex"
	headerFileName  = "headers.bin"
	utxodbFileName  = "utxo.db"
	lndbFileName    = "ln.db"
	watchdbFileName = "watch.db"
	sorcFileName    = "sorc.db"
	// this is my local testnet node, replace it with your own close by.
	// Random internet testnet nodes usually work but sometimes don't, so
	// maybe I should test against different versions out there.
	hardHeight = 1063333 // height to start at if not specified
)

var (
	//	Params = &chaincfg.TestNet3Params
	//	Params = &chaincfg.RegressionNetParams
	SCon uspv.SPVCon // global here for now

	Node qln.LitNode
)

// variables for a goodelivery session
type LitConfig struct {
	spvHost               string
	regTest, reSync, hard bool // flag to set mainnet
	birthblock            int32
	rpcport               uint16
	litHomeDir            string

	Params *chaincfg.Params
}

func setConfig(lc *LitConfig) {
	spvhostptr := flag.String("spv", "tn.lit3.co", "full node to connect to")

	birthptr := flag.Int("tip", hardHeight, "height to begin db sync")

	easyptr := flag.Bool("ez", false, "use easy mode (bloom filters)")

	regtestptr := flag.Bool("reg", false, "use regtest (not testnet3)")
	resyncprt := flag.Bool("resync", false, "force resync from given tip")

	rpcportptr := flag.Int("rpcport", 9750, "port to listen for RPC")

	litHomeDir := flag.String("dir", filepath.Join(os.Getenv("HOME"), litHomeDirName), "lit home directory")

	flag.Parse()

	lc.spvHost = *spvhostptr
	lc.birthblock = int32(*birthptr)

	lc.regTest = *regtestptr
	lc.reSync = *resyncprt
	lc.hard = !*easyptr

	lc.rpcport = uint16(*rpcportptr)

	lc.litHomeDir = *litHomeDir

	//	if lc.spvHost == "" {
	//		lc.spvHost = "lit3.co"
	//	}

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

	if lc.reSync && lc.birthblock == hardHeight {
		log.Fatal("-resync requires -tip")
	}
}

func main() {
	fmt.Printf("lit node v0.0\n")
	fmt.Printf("-h for list of options.\n")

	conf := new(LitConfig)
	setConfig(conf)

	// create lit home directory if the diretory does not exist
	if _, err := os.Stat(conf.litHomeDir); os.IsNotExist(err) {
		os.Mkdir(conf.litHomeDir, 0700)
	}

	// define file paths based on lit home directory
	keyFilePath := filepath.Join(conf.litHomeDir, keyFileName)
	headerFilePath := filepath.Join(conf.litHomeDir, headerFileName)
	utxodbFilePath := filepath.Join(conf.litHomeDir, utxodbFileName)
	lndbFilePath := filepath.Join(conf.litHomeDir, lndbFileName)
	watchdbFilePath := filepath.Join(conf.litHomeDir, watchdbFileName)

	// read key file (generate if not found)
	rootPriv, err := uspv.ReadKeyFileToECPriv(keyFilePath, conf.Params)
	if err != nil {
		log.Fatal(err)
	}
	// setup TxStore first (before spvcon)
	Store := uspv.NewTxStore(rootPriv, conf.Params)

	// setup SPVCon
	SCon, err = uspv.OpenSPV(headerFilePath, utxodbFilePath, &Store,
		conf.hard, false, conf.Params)
	if err != nil {
		log.Printf("can't connect: %s", err.Error())
		log.Fatal(err) // back to fatal when can't connect
	}

	// Setup LN node.  Activate Tower if in hard mode.
	err = Node.Init(lndbFilePath, watchdbFilePath, &SCon, SCon.HardMode)
	if err != nil {
		log.Fatal(err)
	}

	tip, err := SCon.TS.GetDBSyncHeight() // ask for sync height
	if err != nil {
		log.Fatal(err)
	}

	if tip == 0 || conf.reSync { // DB has never been used, set to birthday
		if conf.regTest {
			if conf.birthblock < 100000 {
				tip = conf.birthblock
			} else {
				tip = 500 // for regtest
			}
		} else {
			tip = conf.birthblock // for testnet3. should base on keyfile date?
		}
		err = SCon.TS.SetDBSyncHeight(tip)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Connect to full node
	err = SCon.Connect(conf.spvHost)
	if err != nil {
		log.Fatal(err)
	}

	// once we're connected, initiate headers sync
	err = SCon.AskForHeaders()
	if err != nil {
		log.Fatal(err)
	}

	// shell loop -- to be removed
	/*
		go func() {
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
		}()
	*/
	litrpc.RpcListen(&SCon, &Node, conf.rpcport)

	return
}
