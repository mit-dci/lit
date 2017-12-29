package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/litbamf"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"
)

type config struct { // define a struct for usage with go-flags
	Tn3host     string `long:"tn3" description:"Connect to bitcoin testnet3."`
	Bc2host     string `long:"bc2" description:"bc2 full node."`
	Lt4host     string `long:"lt4" description:"Connect to litecoin testnet4."`
	Reghost     string `long:"reg" description:"Connect to bitcoin regtest."`
	Litereghost string `long:"litereg" description:"Connect to litecoin regtest."`
	Tvtchost    string `long:"tvtc" description:"Connect to Vertcoin test node."`
	Vtchost     string `long:"vtc" description:"Connect to Vertcoin."`
	LitHomeDir  string `long:"dir" description:"Specify Home Directory of lit as an absolute path."`
	TrackerURL  string `long:"tracker" description:"LN address tracker URL http|https://host:port"`
	ConfigFile  string

	Resync  bool `short:"r" long:"resync" description:"Resync from the given tip. Requires --tip"`
	Tower   bool `long:"tower" description:"Watchtower: Run a watching node"`
	Hard    bool `long:"hard" description:"Flag to set networks."`
	Verbose bool `short:"v" long:"verbose" description:"Set verbosity to true."`

	Rpcport uint16 `short:"p" long:"rpcport" description:"Set RPC port to connect to"`
	Tip     int32  `short:"t" long:"tip" description:"Specify tip to begin sync from"`

	Params *coinparam.Params
}

var (
	defaultLitHomeDirName = os.Getenv("HOME") + "/.lit"
	defaultTrackerURL     = "http://ni.media.mit.edu:46580"
	defaultKeyFileName    = "privkey.hex"
	defaultConfigFilename = "lit.conf"
	defaultHomeDir        = os.Getenv("HOME")
	defaultRpcport        = uint16(8001)
	defaultTip            = int32(-1) // wallit.GetDBSyncHeight()
)

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// newConfigParser returns a new command line flags parser.
func newConfigParser(conf *config, options flags.Options) *flags.Parser {
	parser := flags.NewParser(conf, options)
	return parser
}
func linkWallets(node *qln.LitNode, key *[32]byte, conf *config) error {
	// for now, wallets are linked to the litnode on startup, and
	// can't appear / disappear while it's running.  Later
	// could support dynamically adding / removing wallets

	// order matters; the first registered wallet becomes the default

	var err error
	// try regtest
	if !lnutil.NopeString(conf.Reghost) {
		p := &coinparam.RegressionNetParams
		fmt.Printf("reg: %s\n", conf.Reghost)
		err = node.LinkBaseWallet(key, conf.Tip, conf.Resync, conf.Tower, conf.Reghost, p)
		if err != nil {
			return err
		}
	}
	// try testnet3
	if !lnutil.NopeString(conf.Tn3host) {
		p := &coinparam.TestNet3Params
		err = node.LinkBaseWallet(
			// key, 1210000, conf.Resync, conf.Tower,
			key, conf.Tip, conf.Resync, conf.Tower,
			conf.Tn3host, p)
		if err != nil {
			return err
		}
	}
	// try litecoin regtest
	if !lnutil.NopeString(conf.Litereghost) {
		p := &coinparam.LiteRegNetParams
		err = node.LinkBaseWallet(key, conf.Tip, conf.Resync, conf.Tower, conf.Litereghost, p)
		if err != nil {
			return err
		}
	}

	// try litecoin testnet4
	if !lnutil.NopeString(conf.Lt4host) {
		p := &coinparam.LiteCoinTestNet4Params
		err = node.LinkBaseWallet(
			key, conf.Tip, conf.Resync, conf.Tower, // start height is 48384 for litecoin
			conf.Lt4host, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin testnet
	if !lnutil.NopeString(conf.Tvtchost) {
		p := &coinparam.VertcoinTestNetParams
		err = node.LinkBaseWallet(
			key, conf.Tip, conf.Resync, conf.Tower, // vtc start height is 0
			conf.Tvtchost, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin mainnet
	if !lnutil.NopeString(conf.Vtchost) {
		p := &coinparam.VertcoinParams
		err = node.LinkBaseWallet(
			key, conf.Tip, conf.Resync, conf.Tower,
			conf.Vtchost, p)
		if err != nil {
			return err
		}

	}
	return nil
}

func main() {

	conf := config{
		LitHomeDir: defaultLitHomeDirName,
		Rpcport:    defaultRpcport,
		TrackerURL: defaultTrackerURL,
		Tip:        defaultTip,
	}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.
	usageMessage := fmt.Sprintf("Use %s -h to show usage", "./lit")
	preconf := conf
	preParser := newConfigParser(&preconf, flags.HelpFlag)
	_, err := preParser.ParseArgs(os.Args)
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Println(err)
			return
		}
	}

	if preconf.Resync && preconf.Tip == -1 {
		log.Fatal("--reSync requires --tip. Exiting.")
		return
	}
	if !preconf.Resync { // set the default tip values
		if preconf.Tn3host != "" {
			preconf.Tip = 1210000
		}
		if preconf.Reghost != "" {
			preconf.Tip = 120
		}
		if preconf.Lt4host != "" {
			preconf.Tip = 48384
		}
		if preconf.Litereghost != "" {
			preconf.Tip = 120
		}
		if preconf.Vtchost != "" {
			preconf.Tip = 598752
		}
		if preconf.Tvtchost != "" {
			preconf.Tip = 0
		}
	}
	log.Printf("Default chain header tip %d", preconf.Tip)
	parser := newConfigParser(&conf, flags.Default)

	_, err = os.Stat(preconf.LitHomeDir) // create directory
	if err != nil {
		log.Println("Error while creating a directory")
	}
	if os.IsNotExist(err) {
		// first time the guy is running lit, lets set tn3 to true
		os.Mkdir(preconf.LitHomeDir, 0700)
		log.Println("Creating a new config file")
		err := createDefaultConfigFile(preconf.LitHomeDir) // Source of error
		if err != nil {
			log.Println("Error creating a default config file: %v\n")
			log.Fatal(err)
		}
	}

	if _, err := os.Stat(filepath.Join(filepath.Join(preconf.LitHomeDir), "lit.conf")); os.IsNotExist(err) {
		// if there is no config file found over at the directory, create one
		if err != nil {
			log.Println(err)
		}
		log.Println("Creating a new config file")
		err := createDefaultConfigFile(filepath.Join(preconf.LitHomeDir)) // Source of error
		if err != nil {
			log.Fatal(err)
			return
		}
	}

	preconf.ConfigFile = filepath.Join(filepath.Join(preconf.LitHomeDir), "lit.conf")
	// lets parse the config file provided, if any
	err = flags.NewIniParser(parser).ParseFile(preconf.ConfigFile)
	if err != nil {
		_, ok := err.(*os.PathError)
		if !ok {
			log.Fatal(err)
		}
	}
	// Parse command line options again to ensure they take precedence.
	_, err = parser.ParseArgs(os.Args) // returns invalid flags
	if err != nil {
		fmt.Println(usageMessage)
		// no need to print the error as we already have
		return
	}

	logFilePath := filepath.Join(conf.LitHomeDir, "lit.log")

	logfile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer logfile.Close()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	if conf.Verbose {
		logOutput := io.MultiWriter(os.Stdout, logfile)
		log.SetOutput(logOutput)
	} else {
		log.SetOutput(logfile)
	}

	// Allow node with no linked wallets, for testing.
	// TODO Should update tests and disallow nodes without wallets later.
	//	if conf.Tn3host == "" && conf.Lt4host == "" && conf.Reghost == "" {
	//		log.Fatal("error: no network specified; use -tn3, -reg, -lt4")
	//	}

	// Keys: the litNode, and wallits, all get 32 byte keys.
	// Right now though, they all get the *same* key.  For lit as a single binary
	// now, all using the same key makes sense; could split up later.

	keyFilePath := filepath.Join(conf.LitHomeDir, defaultKeyFileName)

	// read key file (generate if not found)
	key, err := lnutil.ReadKeyFile(keyFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Setup LN node.  Activate Tower if in hard mode.
	// give node and below file pathof lit home directoy
	node, err := qln.NewLitNode(key, conf.LitHomeDir, conf.TrackerURL)
	if err != nil {
		log.Fatal(err)
	}

	// node is up; link wallets based on args
	err = linkWallets(node, key, &conf)
	if err != nil {
		log.Fatal(err)
	}

	rpcl := new(litrpc.LitRPC)
	rpcl.Node = node
	rpcl.OffButton = make(chan bool, 1)

	go litrpc.RPCListen(rpcl, conf.Rpcport)
	litbamf.BamfListen(conf.Rpcport, conf.LitHomeDir)

	<-rpcl.OffButton
	fmt.Printf("Got stop request\n")
	time.Sleep(time.Second)

	return
	// New directory being created over at PWD
	// conf file being created at /
}
