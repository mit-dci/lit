package qln

import (
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/portxo"
)

// TCPListener starts a litNode listening for incoming LNDC connections
func (nd *LitNode) TCPListener(lisIpPort string) (*btcutil.AddressPubKeyHash, error) {
	idPriv := nd.IdKey()
	listener, err := lndc.NewListener(nd.IdKey(), lisIpPort)
	if err != nil {
		return nil, err
	}

	myId := btcutil.Hash160(idPriv.PubKey().SerializeCompressed())
	lisAdr, err := btcutil.NewAddressPubKeyHash(myId, nd.Param)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Listening on %s\n", listener.Addr().String())
	fmt.Printf("Listening with base58 address: %s lnid: %x\n",
		lisAdr.String(), myId[:16])

	go func() {
		for {
			netConn, err := listener.Accept() // this blocks
			if err != nil {
				log.Printf("Listener error: %s\n", err.Error())
				continue
			}
			newConn, ok := netConn.(*lndc.LNDConn)
			if !ok {
				fmt.Printf("Got something that wasn't a LNDC")
				continue
			}

			peerIdx, err := nd.GetPeerIdx(newConn.RemotePub)
			if err != nil {
				log.Printf("Listener error: %s\n", err.Error())
				continue
			}

			nd.RemoteMtx.Lock()
			nd.RemoteCons[peerIdx] = newConn
			nd.RemoteMtx.Unlock()

			// each connection to a peer gets its own LNDCReader
			go nd.LNDCReader(newConn, peerIdx)
		}
	}()
	return lisAdr, nil
}

// IdKey returns the identity private key
func (nd *LitNode) IdKey() *btcec.PrivateKey {
	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = 0 | 1<<31
	kg.Step[2] = 9 | 1<<31
	kg.Step[3] = 0 | 1<<31
	kg.Step[4] = 0 | 1<<31
	return nd.BaseWallet.GetPriv(kg)
}
