package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"

	flags "github.com/jessevdk/go-flags"
)

type config struct { // define a struct for usage with go-flags
	// networks lit can try connecting to
	Tn3host     string `long:"tn3" description:"Connect to bitcoin testnet3."`
	Bc2host     string `long:"bc2" description:"bc2 full node."`
	Lt4host     string `long:"lt4" description:"Connect to litecoin testnet4."`
	Reghost     string `long:"reg" description:"Connect to bitcoin regtest."`
	Litereghost string `long:"litereg" description:"Connect to litecoin regtest."`
	Tvtchost    string `long:"tvtc" description:"Connect to Vertcoin test node."`
	Vtchost     string `long:"vtc" description:"Connect to Vertcoin."`
	// system specific configs
	LitHomeDir string `long:"dir" description:"Specify Home Directory of lit as an absolute path."`
	TrackerURL string `long:"tracker" description:"LN address tracker URL http|https://host:port"`
	ConfigFile string

	// proxy
	ProxyURL      string `long:"proxy" description:"SOCKS5 proxy to use for communicating with the network"`
	LitProxyURL   string `long:"litproxy" description:"SOCKS5 proxy to use for Lit's network communications. Overridden by the proxy flag."`
	ChainProxyURL string `long:"chainproxy" description:"SOCKS5 proxy to use for Wallit's network communications. Overridden by the proxy flag."`

	//UPnP port forwarding and NAT Traversal
	Nat string `long:"nat" description:"Toggle upnp or pmp NAT Traversal NAT Punching"`
	//resync and tower config
	ReSync bool `short:"r" long:"reSync" description:"Resync from the given tip."`
	Tower  bool `long:"tower" description:"Watchtower: Run a watching node"`
	Hard   bool `short:"t" long:"hard" description:"Flag to set networks."`
  
	Verbose bool `short:"v" long:"verbose" description:"Set verbosity to true."`
	// rpc server config
	Rpcport uint16 `short:"p" long:"rpcport" description:"Set RPC port to connect to"`
	Rpchost string `long:"rpchost" description:"Set RPC host to listen to"`
	// auto config
	AutoReconnect         bool   `long:"autoReconnect" description:"Attempts to automatically reconnect to known peers periodically."`
	AutoReconnectInterval int64  `long:"autoReconnectInterval" description:"The interval (in seconds) the reconnect logic should be executed"`
	AutoListenPort        string `long:"autoListenPort" description:"When auto reconnect enabled, starts listening on this port"`
	Params                *coinparam.Params
}

var (
	defaultLitHomeDirName        = os.Getenv("HOME") + "/.lit"
	defaultTrackerURL            = "http://hubris.media.mit.edu:46580"
	defaultKeyFileName           = "privkey.hex"
	defaultConfigFilename        = "lit.conf"
	defaultHomeDir               = os.Getenv("HOME")
	defaultRpcport               = uint16(8001)
	defaultRpchost               = "localhost"
	defaultAutoReconnect         = false
	defaultAutoListenPort        = ":2448"
	defaultAutoReconnectInterval = int64(60)
	defaultUpnPFlag              = false
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
		err = node.LinkBaseWallet(key, 120, conf.ReSync,
			conf.Tower, conf.Reghost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try testnet3
	if !lnutil.NopeString(conf.Tn3host) {
		p := &coinparam.TestNet3Params
		err = node.LinkBaseWallet(
			key, 1256000, conf.ReSync, conf.Tower,
			conf.Tn3host, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try litecoin regtest
	if !lnutil.NopeString(conf.Litereghost) {
		p := &coinparam.LiteRegNetParams
		err = node.LinkBaseWallet(key, 120, conf.ReSync,
			conf.Tower, conf.Litereghost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try litecoin testnet4
	if !lnutil.NopeString(conf.Lt4host) {
		p := &coinparam.LiteCoinTestNet4Params
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.ReSync, conf.Tower,
			conf.Lt4host, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin testnet
	if !lnutil.NopeString(conf.Tvtchost) {
		p := &coinparam.VertcoinTestNetParams
		err = node.LinkBaseWallet(
			key, 25000, conf.ReSync, conf.Tower,
			conf.Tvtchost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin mainnet
	if !lnutil.NopeString(conf.Vtchost) {
		p := &coinparam.VertcoinParams
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.ReSync, conf.Tower,
			conf.Vtchost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}

	}
	return nil
}

func main() {

	conf := config{
		LitHomeDir:            defaultLitHomeDirName,
		Rpcport:               defaultRpcport,
		Rpchost:               defaultRpchost,
		TrackerURL:            defaultTrackerURL,
		AutoReconnect:         defaultAutoReconnect,
		AutoListenPort:        defaultAutoListenPort,
		AutoReconnectInterval: defaultAutoReconnectInterval,
	}

	key := litSetup(&conf)

	if conf.ProxyURL != "" {
		conf.LitProxyURL = conf.ProxyURL
		conf.ChainProxyURL = conf.ProxyURL
	}

	// Setup LN node.  Activate Tower if in hard mode.
	// give node and below file pathof lit home directory
	node, err := qln.NewLitNode(key, conf.LitHomeDir, conf.TrackerURL, conf.LitProxyURL, conf.Nat)
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

	go litrpc.RPCListen(rpcl, conf.Rpchost, conf.Rpcport)

	if conf.AutoReconnect {
		node.AutoReconnect(conf.AutoListenPort, conf.AutoReconnectInterval)
	}

	<-rpcl.OffButton
	log.Printf("Got stop request\n")
	time.Sleep(time.Second)

	return
	// New directory being created over at PWD
	// conf file being created at /
}
