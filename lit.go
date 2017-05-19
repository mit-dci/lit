package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/litbamf"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"
)

const (
	litHomeDirName = ".lit"

	keyFileName = "privkey.hex"
)

// variables for a lit node & lower layers
type LitConfig struct {
	reSync, hard bool // flag to set networks

	// hostnames to connect to for different networks
	tn3host, bc2host, lt4host, reghost, litereghost, tvtchost, vtchost string

	verbose    bool
	rpcport    uint16
	litHomeDir string

	Params *coinparam.Params
}

func setConfig(lc *LitConfig) {
	easyptr := flag.Bool("ez", false, "use easy mode (bloom filters)")

	verbptr := flag.Bool("v", false, "verbose; print all logs to stdout")

	tn3ptr := flag.String("tn3", "", "testnet3 full node")
	regptr := flag.String("reg", "", "regtest full node")
	literegptr := flag.String("ltr", "", "litecoin regtest full node")
	bc2ptr := flag.String("bc2", "", "bc2 full node")
	lt4ptr := flag.String("lt4", "", "litecoin testnet4 full node")
    tvtcptr := flag.String("tvtc", "", "vertcoin testnet full node")
    vtcptr := flag.String("vtc", "", "vertcoin mainnet full node")

	resyncprt := flag.Bool("resync", false, "force resync from given tip")

	rpcportptr := flag.Int("rpcport", 8001, "port to listen for RPC")

	litHomeDir := flag.String("dir",
		filepath.Join(os.Getenv("HOME"), litHomeDirName), "lit home directory")

	flag.Parse()

	lc.tn3host, lc.bc2host, lc.lt4host, lc.reghost, lc.tvtchost, lc.vtchost =
		*tn3ptr, *bc2ptr, *lt4ptr, *regptr, *tvtcptr, *vtcptr

	lc.litereghost = *literegptr

	lc.reSync = *resyncprt
	lc.hard = !*easyptr
	lc.verbose = *verbptr

	lc.rpcport = uint16(*rpcportptr)

	lc.litHomeDir = *litHomeDir
}

// linkWallets tries to link the wallets given in conf to the litNode
func linkWallets(node *qln.LitNode, key *[32]byte, conf *LitConfig) error {
	// for now, wallets are linked to the litnode on startup, and
	// can't appear / disappear while it's running.  Later
	// could support dynamically adding / removing wallets

	// order matters; the first registered wallet becomes the default

	var err error
	// try regtest
	if conf.reghost != "" {
		p := &coinparam.RegressionNetParams
		if !strings.Contains(conf.reghost, ":") {
			conf.reghost = conf.reghost + ":" + p.DefaultPort
		}
		fmt.Printf("reg: %s\n", conf.reghost)
		err = node.LinkBaseWallet(key, 120, conf.reSync, conf.reghost, p)
		if err != nil {
			return err
		}
	}
	// try testnet3
	if conf.tn3host != "" {
		p := &coinparam.TestNet3Params
		if !strings.Contains(conf.tn3host, ":") {
			conf.tn3host = conf.tn3host + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.reSync,
			conf.tn3host, p)
		if err != nil {
			return err
		}
	}
	// try litecoin regtest
	if conf.litereghost != "" {
		p := &coinparam.LiteRegNetParams
		if !strings.Contains(conf.litereghost, ":") {
			conf.litereghost = conf.litereghost + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(key, 120, conf.reSync, conf.litereghost, p)
		if err != nil {
			return err
		}
	}

	// try litecoin testnet4
	if conf.lt4host != "" {
		p := &coinparam.LiteCoinTestNet4Params
		if !strings.Contains(conf.lt4host, ":") {
			conf.lt4host = conf.lt4host + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.reSync,
			conf.lt4host, p)
		if err != nil {
			return err
		}
	}
  // try vertcoin testnet
	if conf.tvtchost != "" {
	    p := &coinparam.VertcoinTestNetParams
		if !strings.Contains(conf.tvtchost, ":") {
			conf.tvtchost = conf.tvtchost + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(
			key, 0, conf.reSync,
			conf.tvtchost, p)
		if err != nil {
			return err
		}
	}
  // try vertcoin mainnet
	if conf.vtchost != "" {
	    p := &coinparam.VertcoinParams
		if !strings.Contains(conf.vtchost, ":") {
			conf.vtchost = conf.vtchost + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.reSync,
			conf.vtchost, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {

	log.Printf("lit v0.1\n")
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

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	if conf.verbose {
		logOutput := io.MultiWriter(os.Stdout, logfile)
		log.SetOutput(logOutput)
	} else {
		log.SetOutput(logfile)
	}

	// Allow node with no linked wallets, for testing.
	// TODO Should update tests and disallow nodes without wallets later.
	//	if conf.tn3host == "" && conf.lt4host == "" && conf.reghost == "" {
	//		log.Fatal("error: no network specified; use -tn3, -reg, -lt4")
	//	}

	// Keys: the litNode, and wallits, all get 32 byte keys.
	// Right now though, they all get the *same* key.  For lit as a single binary
	// now, all using the same key makes sense; could split up later.
	keyFilePath := filepath.Join(conf.litHomeDir, keyFileName)

	// read key file (generate if not found)
	key, err := lnutil.ReadKeyFile(keyFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Setup LN node.  Activate Tower if in hard mode.
	// give node and below file pathof lit home directoy
	node, err := qln.NewLitNode(key, conf.litHomeDir, false)
	if err != nil {
		log.Fatal(err)
	}

	// node is up; link wallets based on args
	err = linkWallets(node, key, conf)
	if err != nil {
		log.Fatal(err)
	}

	rpcl := new(litrpc.LitRPC)
	rpcl.Node = node
	rpcl.OffButton = make(chan bool, 1)

	litrpc.RPCListen(rpcl, conf.rpcport)
	litbamf.BamfListen(conf.rpcport, conf.litHomeDir)

	<-rpcl.OffButton
	fmt.Printf("Got stop request\n")
	time.Sleep(time.Second)

	return
}
