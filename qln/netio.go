package qln

import (
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
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

// OutMessager takes messages from the outbox and sends them to the ether. net.
func (nd *LitNode) OutMessager() {
	for {
		msg := <-nd.OmniOut
		if !nd.ConnectedToPeer(msg.PeerIdx) {
			fmt.Printf("message type %x to peer %d but not connected\n",
				msg.MsgType, msg.PeerIdx)
			continue
		}

		rawmsg := append([]byte{msg.MsgType}, msg.Data...)
		nd.RemoteMtx.Lock() // not sure this is needed...
		n, err := nd.RemoteCons[msg.PeerIdx].Write(rawmsg)
		if err != nil {
			fmt.Printf("error writing to peer %d: %s\n", err.Error())
		} else {
			fmt.Printf("%d bytes to peer %d\n", n, msg.PeerIdx)
		}
		nd.RemoteMtx.Unlock()
	}
}

type PeerInfo struct {
	PeerNumber uint32
	RemoteHost string
}

func (nd *LitNode) GetConnectedPeerList() []PeerInfo {
	nd.RemoteMtx.Lock()
	nd.RemoteMtx.Unlock()
	var peers []PeerInfo
	for k, v := range nd.RemoteCons {
		var newPeer PeerInfo
		newPeer.PeerNumber = k
		newPeer.RemoteHost = v.RemoteAddr().String()
		peers = append(peers, newPeer)
	}
	return peers
}

// ConnectedToPeer checks whether you're connected to a specific peer
func (nd *LitNode) ConnectedToPeer(peer uint32) bool {
	nd.RemoteMtx.Lock()
	_, ok := nd.RemoteCons[peer]
	nd.RemoteMtx.Unlock()
	return ok
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

// SendChat sends a text string to a peer
func (nd *LitNode) SendChat(peer uint32, chat string) error {
	if !nd.ConnectedToPeer(peer) {
		return fmt.Errorf("Not connected to peer %d", peer)
	}

	outMsg := new(lnutil.LitMsg)
	outMsg.MsgType = lnutil.MSGID_TEXTCHAT
	outMsg.PeerIdx = peer
	outMsg.Data = []byte(chat)
	nd.OmniOut <- outMsg

	return nil
}
