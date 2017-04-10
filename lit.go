package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"
)

const (
	litHomeDirName = ".lit"

	keyFileName = "testkey.hex"

	// this is my local testnet node, replace it with your own close by.
	// Random internet testnet nodes usually work but sometimes don't, so
	// maybe I should test against different versions out there.
	hardHeight = 1111111 // height to start at if not specified
)

// variables for a goodelivery session
type LitConfig struct {
	spvHost               string
	regTest, reSync, hard bool // flag to set networks
	bc2Net                bool
	verbose               bool
	birthblock            int32
	rpcport               uint16
	litHomeDir            string

	Params *chaincfg.Params
}

func setConfig(lc *LitConfig) {
	spvhostptr := flag.String("spv", "na", "full node to connect to")

	birthptr := flag.Int("tip", hardHeight, "height to begin db sync")

	easyptr := flag.Bool("ez", false, "use easy mode (bloom filters)")

	verbptr := flag.Bool("v", false, "verbose; print all logs to stdout")

	regtestptr := flag.Bool("reg", false, "use regtest (not testnet3)")
	bc2ptr := flag.Bool("bc2", false, "use bc2 network (not testnet3)")
	resyncprt := flag.Bool("resync", false, "force resync from given tip")

	rpcportptr := flag.Int("rpcport", 8001, "port to listen for RPC")

	litHomeDir := flag.String("dir",
		filepath.Join(os.Getenv("HOME"), litHomeDirName), "lit home directory")

	flag.Parse()

	lc.spvHost = *spvhostptr
	lc.birthblock = int32(*birthptr)

	lc.regTest = *regtestptr
	lc.bc2Net = *bc2ptr
	lc.reSync = *resyncprt
	lc.hard = !*easyptr
	lc.verbose = *verbptr

	lc.rpcport = uint16(*rpcportptr)

	lc.litHomeDir = *litHomeDir

	//	if lc.spvHost == "" {
	//		lc.spvHost = "lit3.co"
	//	}

	if lc.regTest && lc.bc2Net {
		log.Fatal("error: can't have -bc2 and -reg")
	}

	if lc.regTest {
		lc.Params = &chaincfg.RegressionNetParams
		lc.birthblock = 120
		if !strings.Contains(lc.spvHost, ":") {
			lc.spvHost = lc.spvHost + ":18444"
		}
	} else if lc.bc2Net {
		lc.Params = &chaincfg.BC2NetParams
		if !strings.Contains(lc.spvHost, ":") {
			lc.spvHost = lc.spvHost + ":8444"
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

	log.Printf("lit node v0.1\n")
	log.Printf("-h for list of options.\n")

	conf := new(LitConfig)
	setConfig(conf)

	// create lit home directory if the diretory does not exist
	if _, err := os.Stat(conf.litHomeDir); os.IsNotExist(err) {
		os.Mkdir(conf.litHomeDir, 0700)
	}

	logFilePath := filepath.Join(conf.litHomeDir, "lit.log")

	logfile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer logfile.Close()

	if conf.verbose {
		logOutput := io.MultiWriter(os.Stdout, logfile)
		log.SetOutput(logOutput)
	} else {
		log.SetOutput(logfile)
	}

	keyFilePath := filepath.Join(conf.litHomeDir, keyFileName)

	// read key file (generate if not found)
	key, err := lnutil.ReadKeyFile(keyFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Setup LN node.  Activate Tower if in hard mode.
	// give node and below file pathof lit home directoy
	node, err := qln.NewLitNode(conf.litHomeDir, false)
	if err != nil {
		log.Fatal(err)
	}

	err = node.LinkBaseWallet(key, conf.birthblock, conf.spvHost, conf.Params)
	if err != nil {
		log.Fatal(err)
	}

	litrpc.RpcListen(node, conf.rpcport)

	return
}
