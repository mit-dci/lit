package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/mit-dci/lit/logging"

	"github.com/mit-dci/lit/coinparam"
	consts "github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"

	flags "github.com/jessevdk/go-flags"
)

type litConfig struct { // define a struct for usage with go-flags
	// networks lit can try connecting to
	Tn3host      string `long:"tn3" description:"Connect to bitcoin testnet3."`
	Bc2host      string `long:"bc2" description:"bc2 full node."`
	Lt4host      string `long:"lt4" description:"Connect to litecoin testnet4."`
	Reghost      string `long:"reg" description:"Connect to bitcoin regtest."`
	Litereghost  string `long:"litereg" description:"Connect to litecoin regtest."`
	Dummyusdhost string `long:"dusd" description:"Connect to Dummy USD node."`
	Rtvtchost    string `long:"rtvtc" description:"Connect to Vertcoin regtest node."`
	Tvtchost     string `long:"tvtc" description:"Connect to Vertcoin test node."`
	Vtchost      string `long:"vtc" description:"Connect to Vertcoin."`
	// system specific configs
	LitHomeDir string `long:"dir" description:"Specify Home Directory of lit as an absolute path."`
	TrackerURL string `long:"tracker" description:"LN address tracker URL http|https://host:port"`
	ConfigFile string
	UnauthRPC  bool `long:"unauthrpc" description:"Enables unauthenticated Websocket RPC"`

	// proxy
	ProxyURL      string `long:"proxy" description:"SOCKS5 proxy to use for communicating with the network"`
	LitProxyURL   string `long:"litproxy" description:"SOCKS5 proxy to use for Lit's network communications. Overridden by the proxy flag."`
	ChainProxyURL string `long:"chainproxy" description:"SOCKS5 proxy to use for Wallit's network communications. Overridden by the proxy flag."`

	// UPnP port forwarding and NAT Traversal
	Nat string `long:"nat" description:"Toggle upnp or pmp NAT Traversal NAT Punching"`
	//resync and tower config
	Resync string `long:"resync" description:"Resync the given chain from the given tip (requires --tip) or from default params"`
	Tip    int32  `long:"tip" description:"Given tip to resync from"`
	Tower  bool   `long:"tower" description:"Watchtower: Run a watching node"`
	Hard   bool   `short:"t" long:"hard" description:"Flag to set networks."`

	// logging and debug parameters
	LogLevel []bool `short:"v" description:"Set verbosity level to verbose (-v), very verbose (-vv) or very very verbose (-vvv)"`

	// rpc server config
	Rpcport uint16 `short:"p" long:"rpcport" description:"Set RPC port to connect to"`
	Rpchost string `long:"rpchost" description:"Set RPC host to listen to"`
	// auto config
	AutoReconnect                   bool   `long:"autoReconnect" description:"Attempts to automatically reconnect to known peers periodically."`
	AutoReconnectInterval           int64  `long:"autoReconnectInterval" description:"The interval (in seconds) the reconnect logic should be executed"`
	AutoReconnectOnlyConnectedCoins bool   `long:"autoReconnectOnlyConnectedCoins" description:"Only reconnect to peers that we have channels with in a coin whose coin daemon is available"`
	AutoListenPort                  string `long:"autoListenPort" description:"When auto reconnect enabled, starts listening on this port"`
	Params                          *coinparam.Params
}

var (
	defaultLitHomeDirName                  = os.Getenv("HOME") + "/.lit"
	defaultTrackerURL                      = "http://hubris.media.mit.edu:46580"
	defaultKeyFileName                     = "privkey.hex"
	defaultConfigFilename                  = "lit.conf"
	defaultHomeDir                         = os.Getenv("HOME")
	defaultRpcport                         = uint16(8001)
	defaultRpchost                         = "localhost"
	defaultAutoReconnect                   = true
	defaultAutoListenPort                  = ":2448"
	defaultAutoReconnectInterval           = int64(60)
	defaultUpnPFlag                        = false
	defaultLogLevel                        = 0
	defaultAutoReconnectOnlyConnectedCoins = false
	defaultUnauthRPC                       = false
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
func newConfigParser(conf *litConfig, options flags.Options) *flags.Parser {
	parser := flags.NewParser(conf, options)
	return parser
}
func linkWallets(node *qln.LitNode, key *[32]byte, conf *litConfig) error {
	// for now, wallets are linked to the litnode on startup, and
	// can't appear / disappear while it's running.  Later
	// could support dynamically adding / removing wallets

	// order matters; the first registered wallet becomes the default

	var err error
	// try regtest
	if !lnutil.NopeString(conf.Reghost) {
		p := &coinparam.RegressionNetParams
		logging.Infof("reg: %s\n", conf.Reghost)
		resync := false
		if conf.Resync == "reg" {
			if conf.Tip < consts.BitcoinRegtestBHeight {
				conf.Tip = consts.BitcoinRegtestBHeight
			}
			resync = true
		}
		err = node.LinkBaseWallet(key, conf.Tip, resync,
			conf.Tower, conf.Reghost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try testnet3
	if !lnutil.NopeString(conf.Tn3host) {
		p := &coinparam.TestNet3Params
		resync := false
		if conf.Resync == "tn3" {
			if conf.Tip < consts.BitcoinTestnet3BHeight {
				conf.Tip = consts.BitcoinTestnet3BHeight
			}
			resync = true
		}
		err = node.LinkBaseWallet(
			key, conf.Tip, resync, conf.Tower,
			conf.Tn3host, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try litecoin regtest
	if !lnutil.NopeString(conf.Litereghost) {
		p := &coinparam.LiteRegNetParams
		resync := false
		if conf.Resync == "ltcreg" {
			if conf.Tip < consts.BitcoinRegtestBHeight {
				conf.Tip = consts.BitcoinRegtestBHeight // birth heights are the same for btc and ltc regtests
			}
			resync = true
		}
		err = node.LinkBaseWallet(key, conf.Tip, resync,
			conf.Tower, conf.Litereghost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try litecoin testnet4
	if !lnutil.NopeString(conf.Lt4host) {
		p := &coinparam.LiteCoinTestNet4Params
		resync := false
		conf.Tip = p.StartHeight
		if conf.Resync == "ltctn" {
			if conf.Tip < 1 {
				conf.Tip = 1
			}
			resync = true
		}
		err = node.LinkBaseWallet(
			key, p.StartHeight, resync, conf.Tower,
			conf.Lt4host, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin testnet
	if !lnutil.NopeString(conf.Tvtchost) {
		p := &coinparam.VertcoinTestNetParams
		resync := false
		if conf.Resync == "vtctn" {
			resync = true
		}
		err = node.LinkBaseWallet(
			key, consts.VertcoinTestnetBHeight, resync, conf.Tower,
			conf.Tvtchost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin mainnet
	if !lnutil.NopeString(conf.Vtchost) {
		p := &coinparam.VertcoinParams
		resync := false
		conf.Tip = p.StartHeight
		if conf.Resync == "ltctn" {
			if conf.Tip < 1 {
				conf.Tip = 1
			}
			resync = true
		}
		err = node.LinkBaseWallet(
			key, conf.Tip, resync, conf.Tower,
			conf.Vtchost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	// try dummyusd
	if !lnutil.NopeString(conf.Dummyusdhost) {
		p := &coinparam.DummyUsdNetParams
		logging.Infof("Dummyusd: %s\n", conf.Dummyusdhost)
		resync := false
		conf.Tip = p.StartHeight
		if conf.Resync == "dusd" {
			if conf.Tip < 1 {
				conf.Tip = 1
			}
			resync = true
		}
		err = node.LinkBaseWallet(key, consts.BitcoinRegtestBHeight, resync,
			conf.Tower, conf.Dummyusdhost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}

	// try vertcoin regtest
	if !lnutil.NopeString(conf.Rtvtchost) {
		p := &coinparam.VertcoinRegTestParams
		resync := false
		conf.Tip = p.StartHeight
		if conf.Resync == "rtvtc" {
			if conf.Tip < 1 {
				conf.Tip = 1
			}
			resync = true
		}

		err = node.LinkBaseWallet(
			key, consts.BitcoinRegtestBHeight, resync, conf.Tower,
			conf.Rtvtchost, conf.ChainProxyURL, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {

	conf := litConfig{
		LitHomeDir:                      defaultLitHomeDirName,
		Rpcport:                         defaultRpcport,
		Rpchost:                         defaultRpchost,
		TrackerURL:                      defaultTrackerURL,
		AutoReconnect:                   defaultAutoReconnect,
		AutoListenPort:                  defaultAutoListenPort,
		AutoReconnectInterval:           defaultAutoReconnectInterval,
		AutoReconnectOnlyConnectedCoins: defaultAutoReconnectOnlyConnectedCoins,
		UnauthRPC:                       defaultUnauthRPC,
	}

	key := litSetup(&conf)
	if conf.ProxyURL != "" {
		conf.LitProxyURL = conf.ProxyURL
		conf.ChainProxyURL = conf.ProxyURL
	}

	// SIGQUIT handler for debugging
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGQUIT)
		buf := make([]byte, 1<<20)
		for {
			<-sigs
			stacklen := runtime.Stack(buf, true)
			logging.Warnf("=== received SIGQUIT ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
		}
	}()

	// Setup LN node.  Activate Tower if in hard mode.
	// give node and below file pathof lit home directory
	node, err := qln.NewLitNode(key, conf.LitHomeDir, conf.TrackerURL, conf.LitProxyURL, conf.Nat)
	if err != nil {
		logging.Fatal(err)
	}

	// node is up; link wallets based on args
	err = linkWallets(node, key, &conf)
	if err != nil {
		logging.Fatal(err)
	}

	rpcl := new(litrpc.LitRPC)
	rpcl.Node = node
	rpcl.OffButton = make(chan bool, 1)
	node.RPC = rpcl

	if conf.UnauthRPC {
		go litrpc.RPCListen(rpcl, conf.Rpchost, conf.Rpcport)
	}

	if conf.AutoReconnect {
		node.AutoReconnect(conf.AutoListenPort, conf.AutoReconnectInterval, conf.AutoReconnectOnlyConnectedCoins)
	}

	<-rpcl.OffButton
	logging.Infof("Got stop request\n")
	time.Sleep(time.Second)

	return
	// New directory being created over at PWD
	// conf file being created at /
}
