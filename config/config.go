package config

import (
	"net"
	"os"
	"path/filepath"
	"strconv"

	flags "github.com/jessevdk/go-flags"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/tor"
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

	Tor *TorConfig `group:"Tor" namespace:"tor"`
	Net tor.Net
}

const (
	DefaultTrackerURL            = "http://hubris.media.mit.edu:46580"
	DefaultKeyFileName           = "privkey.hex"
	DefaultConfigFilename        = "lit.conf"
	DefaultRpcport               = uint16(8001)
	DefaultRpchost               = "localhost"
	DefaultAutoReconnect         = false
	DefaultAutoListenPort        = ":2448"
	DefaultPeerPort              = 2448
	DefaultAutoReconnectInterval = int64(60)
	// tor config
	DefaultTorSOCKSPort            = 9050
	DefaultTorDNSHost              = "soa.nodes.lightning.directory" // hos our own tor host
	DefaultTorDNSPort              = 53
	DefaultTorControlPort          = 9051
	DefaultTorV2PrivateKeyFilename = "v2_onion_private_key"
)

var (
	DefaultLitHomeDirName      = os.Getenv("HOME") + "/.lit"
	DefaultHomeDir             = os.Getenv("HOME")
	DefaultTorSOCKS            = net.JoinHostPort("localhost", strconv.Itoa(DefaultTorSOCKSPort))
	DefaultTorDNS              = net.JoinHostPort(DefaultTorDNSHost, strconv.Itoa(DefaultTorDNSPort))
	DefaultTorControl          = net.JoinHostPort("localhost", strconv.Itoa(DefaultTorControlPort))
	DefaultTorV2PrivateKeyPath = filepath.Join(DefaultHomeDir, DefaultTorV2PrivateKeyFilename)
)

// newConfigParser returns a new command line flags parser.
func NewConfigParser(conf *Config, options flags.Options) *flags.Parser {
	parser := flags.NewParser(conf, options)
	return parser
}

type TorConfig struct {
	Active           bool   `long:"active" description:"Allow outbound and inbound connections to be routed through Tor"`
	SOCKS            string `long:"socks" description:"The host:port that Tor's exposed SOCKS5 proxy is listening on"`
	DNS              string `long:"dns" description:"The DNS server as host:port that Tor will use for SRV queries - NOTE must have TCP resolution enabled"`
	StreamIsolation  bool   `long:"streamisolation" description:"Enable Tor stream isolation by randomizing user credentials for each connection."`
	Control          string `long:"control" description:"The host:port that Tor is listening on for Tor control connections"`
	V2               bool   `long:"v2" description:"Automatically set up a v2 onion service to listen for inbound connections"`
	V2PrivateKeyPath string `long:"v2privatekeypath" description:"The path to the private key of the onion service being created"`
	V3               bool   `long:"v3" description:"Use a v3 onion service to listen for inbound connections"`
}
