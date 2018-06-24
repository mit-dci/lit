package main

import (
	"log"
	"time"

	litconfig "github.com/mit-dci/lit/config"
	"github.com/mit-dci/lit/litbamf"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/qln"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/tor"
)

func LinkWallets(node *qln.LitNode, key *[32]byte, conf *litconfig.Config) error {
	// for now, wallets are linked to the litnode on startup, and
	// can't appear / disappear while it's running.  Later
	// could support dynamically adding / removing wallets
	// dont move, circular import stuff
	// order matters; the first registered wallet becomes the default

	var err error
	// try regtest
	if !lnutil.NopeString(conf.Reghost) {
		p := &coinparam.RegressionNetParams
		log.Printf("reg: %s\n", conf.Reghost)
		err = node.LinkBaseWallet(key, 120, conf.ReSync, conf.Tower, conf.Reghost, p, conf)
		if err != nil {
			return err
		}
	}
	// try testnet3
	if !lnutil.NopeString(conf.Tn3host) {
		p := &coinparam.TestNet3Params
		err = node.LinkBaseWallet(
			key, 1256000, conf.ReSync, conf.Tower,
			conf.Tn3host, p, conf)
		if err != nil {
			return err
		}
	}
	// try litecoin regtest
	if !lnutil.NopeString(conf.Litereghost) {
		p := &coinparam.LiteRegNetParams
		err = node.LinkBaseWallet(key, 120, conf.ReSync, conf.Tower, conf.Litereghost, p, conf)
		if err != nil {
			return err
		}
	}
	// try litecoin testnet4
	if !lnutil.NopeString(conf.Lt4host) {
		p := &coinparam.LiteCoinTestNet4Params
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.ReSync, conf.Tower,
			conf.Lt4host, p, conf)
		if err != nil {
			return err
		}
	}
	// try vertcoin testnet
	if !lnutil.NopeString(conf.Tvtchost) {
		p := &coinparam.VertcoinTestNetParams
		err = node.LinkBaseWallet(
			key, 25000, conf.ReSync, conf.Tower,
			conf.Tvtchost, p, conf)
		if err != nil {
			return err
		}
	}
	// try vertcoin mainnet
	if !lnutil.NopeString(conf.Vtchost) {
		p := &coinparam.VertcoinParams
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.ReSync, conf.Tower,
			conf.Vtchost, p, conf)
		if err != nil {
			return err
		}

	}
	return nil
}


// initTorController initiliazes the Tor controller backed by lit and
// automatically sets up a v2 onion service in order to listen for inbound
// connections over Tor.
func initTorController(torController *tor.Controller, conf litconfig.Config) error {
	if err := torController.Start(); err != nil {
		log.Println("ERRORRS HERE")
		return err
	}
	defaultPeerPort := 2448
	listenPorts := make(map[int]struct{})
	listenPorts[2448] = struct{}{}

	// Once the port mapping has been set, we can go ahead and automatically
	// create our onion service. The service's private key will be saved to
	// disk in order to regain access to this service when restarting lit.
	virtToTargPorts := tor.VirtToTargPorts{defaultPeerPort: listenPorts}
	onionServiceAddr, err := torController.AddOnionV2( //onionServiceAddrs
		conf.Tor.V2PrivateKeyPath, virtToTargPorts,
	)
	if err != nil {
		return err
	}
	log.Println("Listening on onion address:", onionServiceAddr)
	return nil
}

func main() {

	conf := litconfig.Config{
		LitHomeDir:            litconfig.DefaultLitHomeDirName,
		Rpcport:               litconfig.DefaultRpcport,
		Rpchost:               litconfig.DefaultRpchost,
		TrackerURL:            litconfig.DefaultTrackerURL,
		AutoReconnect:         litconfig.DefaultAutoReconnect,
		AutoListenPort:        litconfig.DefaultAutoListenPort,
		AutoReconnectInterval: litconfig.DefaultAutoReconnectInterval,
		Tor: &litconfig.TorConfig{
			SOCKS:            litconfig.DefaultTorSOCKS,
			DNS:              litconfig.DefaultTorDNS,
			Control:          litconfig.DefaultTorControl,
			V2PrivateKeyPath: litconfig.DefaultTorV2PrivateKeyPath,
		},
		Net: &tor.ClearNet{},
	}

	key := litconfig.LitSetup(&conf)

	var torController *tor.Controller

	if conf.Tor.Active && conf.Tor.V2 {
		torController = tor.NewController(conf.Tor.Control) //main controller?
		if err := initTorController(torController, conf); err != nil {
			log.Fatal(err)
		}
	}

	// Setup LN node.  Activate Tower if in hard mode.
	// give node and below file pathof lit home directory
	node, err := qln.NewLitNode(key, conf.LitHomeDir, conf.TrackerURL, conf.ProxyURL)
	if err != nil {
		log.Fatal(err)
	}

	// node is up; link wallets based on args
	err = LinkWallets(node, key, &conf)
	if err != nil {
		log.Fatal(err)
	}

	rpcl := new(litrpc.LitRPC)
	rpcl.Node = node
	rpcl.OffButton = make(chan bool, 1)

	go litrpc.RPCListen(rpcl, conf.Rpchost, conf.Rpcport)
	litbamf.BamfListen(conf.Rpcport, conf.LitHomeDir)

	if conf.AutoReconnect {
		node.AutoReconnect(conf.AutoListenPort, conf.AutoReconnectInterval)
	}

	<-rpcl.OffButton
	log.Printf("Got stop request\n")
	time.Sleep(time.Second)

	return
}
