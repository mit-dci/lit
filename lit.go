package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/fatih/color"
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
	bamfport              uint16
	litHomeDir            string

	Params *chaincfg.Params
}

type arguments struct {
	name         string
	ptrName      string
	defaultValue interface{}
	description  string
}

var verbArg = &arguments{
	name:         "v",
	defaultValue: false,
	description:  "verbose: print all logs to stdout",
}

var spvHostArg = &arguments{
	name:         "spv",
	ptrName:      "hostname/address",
	defaultValue: "127.0.0.1",
	description:  "full node to connect to",
}

var birthArg = &arguments{
	name:         "tip",
	ptrName:      "block",
	defaultValue: hardHeight,
	description:  "height to begin db sync",
}
var easyArg = &arguments{
	name:        "ez",
	description: "use easy mode (bloom filters)",
}
var regtestArg = &arguments{
	name:        "reg",
	description: "use regtest (not testnet3)",
}
var bc2Arg = &arguments{
	name:        "bc2",
	description: "use bc2 network (not testnet3)",
}
var resyncArg = &arguments{
	name:        "resync",
	description: "force resync from given tip",
}
var rpcportArg = &arguments{
	name:         "rpcport",
	ptrName:      "port",
	defaultValue: 8001,
	description:  "port to listen for RPC",
}
var bamfportArg = &arguments{
	name:         "bamfport",
	ptrName:      "port",
	defaultValue: 8001,
	description:  "port to listen for Lit-BAMF",
}
var dirArg = &arguments{
	name:         "dir",
	ptrName:      "directory",
	defaultValue: filepath.Join(os.Getenv("HOME"), litHomeDirName),
	description:  "lit home directory",
}

func setConfig(lc *LitConfig) {

	verbptr := flag.Bool(
		verbArg.name, verbArg.defaultValue.(bool), verbArg.description)
	spvHostptr := flag.String(
		spvHostArg.name, spvHostArg.defaultValue.(string), spvHostArg.description)
	birthptr := flag.Int(birthArg.name, birthArg.defaultValue.(int), birthArg.description)
	easyptr := flag.Bool(easyArg.name, false, easyArg.description)

	regtestptr := flag.Bool(regtestArg.name, false, regtestArg.description)
	bc2ptr := flag.Bool(bc2Arg.name, false, bc2Arg.description)
	resyncprt := flag.Bool(resyncArg.name, false, resyncArg.description)

	rpcportptr := flag.Int(
		rpcportArg.name, rpcportArg.defaultValue.(int), rpcportArg.description)
	bamfportptr := flag.Int(
		bamfportArg.name, bamfportArg.defaultValue.(int), bamfportArg.description)

	litHomeDir := flag.String(dirArg.name, dirArg.defaultValue.(string), dirArg.description)

	ptrs := []arguments{*spvHostArg, *birthArg, *easyArg, *regtestArg, *bc2Arg, *resyncArg,
		*rpcportArg, *bamfportArg, *dirArg, *verbArg}
	flag.Usage = func() {
		file, err := os.Open("lit.ascii")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		b, _ := ioutil.ReadAll(file)
		fmt.Print(string(b))

		bolderline := color.New(color.Bold).Add(color.Underline).SprintFunc()
		fmt.Fprintln(os.Stderr, bolderline("Usage"))

		w := tabwriter.NewWriter(os.Stderr, 3, 4, 1, ' ', 0)
		bold := color.New(color.Bold).SprintFunc()
		underline := color.New(color.Underline).SprintFunc()
		for _, ptr := range ptrs {
			ptrName := ""
			if ptr.ptrName != "" {
				ptrName = ptr.ptrName
			}

			description := ptr.description
			if ptr.defaultValue != nil {
				description = fmt.Sprintf("%s (default \"%v\")", description, ptr.defaultValue)
			}

			ptrOutput := fmt.Sprintf("\t%s\t%s\t\t%s", bold("--"+ptr.name), underline(ptrName), description)
			fmt.Fprintln(w, ptrOutput)
		}
		w.Flush()
	}

	flag.Parse()

	lc.spvHost = *spvHostptr
	lc.birthblock = int32(*birthptr)

	lc.regTest = *regtestptr
	lc.bc2Net = *bc2ptr
	lc.reSync = *resyncprt
	lc.hard = !*easyptr
	lc.verbose = *verbptr

	lc.rpcport = uint16(*rpcportptr)
	lc.bamfport = uint16(*bamfportptr)

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
