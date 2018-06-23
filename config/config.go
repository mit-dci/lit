package config

import (
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/mit-dci/lit/coinparam"
)

type Config struct { // define a struct for usage with go-flags
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
	ProxyURL    string `long:"proxy" description:"SOCKS5 proxy to use for communicating with the network"`

	ReSync  bool `short:"r" long:"reSync" description:"Resync from the given tip."`
	Tower   bool `long:"tower" description:"Watchtower: Run a watching node"`
	Hard    bool `short:"t" long:"hard" description:"Flag to set networks."`
	Verbose bool `short:"v" long:"verbose" description:"Set verbosity to true."`

	Rpcport uint16 `short:"p" long:"rpcport" description:"Set RPC port to connect to"`
	Rpchost string `long:"rpchost" description:"Set RPC host to listen to"`

	AutoReconnect         bool   `long:"autoReconnect" description:"Attempts to automatically reconnect to known peers periodically."`
	AutoReconnectInterval int64  `long:"autoReconnectInterval" description:"The interval (in seconds) the reconnect logic should be executed"`
	AutoListenPort        string `long:"autoListenPort" description:"When auto reconnect enabled, starts listening on this port"`
	Params                *coinparam.Params
}

var (
	DefaultLitHomeDirName        = os.Getenv("HOME") + "/.lit"
	DefaultTrackerURL            = "http://hubris.media.mit.edu:46580"
	DefaultKeyFileName           = "privkey.hex"
	DefaultConfigFilename        = "lit.conf"
	DefaultHomeDir               = os.Getenv("HOME")
	DefaultRpcport               = uint16(8001)
	DefaultRpchost               = "localhost"
	DefaultAutoReconnect         = false
	DefaultAutoListenPort        = ":2448"
	DefaultAutoReconnectInterval = int64(60)
)

// newConfigParser returns a new command line flags parser.
func NewConfigParser(conf *Config, options flags.Options) *flags.Parser {
	parser := flags.NewParser(conf, options)
	return parser
}
