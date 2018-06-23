package main

import (
	"log"
	"time"

	litconfig "github.com/mit-dci/lit/config"
	"github.com/mit-dci/lit/litbamf"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/qln"
	"github.com/mit-dci/lit/tor"
)

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

	// Setup LN node.  Activate Tower if in hard mode.
	// give node and below file pathof lit home directory
	node, err := qln.NewLitNode(key, conf.LitHomeDir, conf.TrackerURL, conf.ProxyURL)
	if err != nil {
		log.Fatal(err)
	}

	// node is up; link wallets based on args
	err = litconfig.LinkWallets(node, key, &conf)
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
