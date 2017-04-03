package qln

import (
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcec"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

// Gets the list of ports where LitNode is listening for incoming connections,
// & the connection key
func (nd *LitNode) GetLisAddressAndPorts() (
	string, []string) {

	idPriv := nd.IdKey()
	var idPub [33]byte
	copy(idPub[:], idPriv.PubKey().SerializeCompressed())

	lisAdr := lnutil.LitAdrFromPubkey(idPub)

	nd.RemoteMtx.Lock()
	ports := nd.LisIpPorts
	nd.RemoteMtx.Unlock()

	return lisAdr, ports
}

// TCPListener starts a litNode listening for incoming LNDC connections
func (nd *LitNode) TCPListener(
	lisIpPort string) (string, error) {
	idPriv := nd.IdKey()
	listener, err := lndc.NewListener(nd.IdKey(), lisIpPort)
	if err != nil {
		return "", err
	}

	var idPub [33]byte
	copy(idPub[:], idPriv.PubKey().SerializeCompressed())

	adr := lnutil.LitAdrFromPubkey(idPub)

	fmt.Printf("Listening on %s\n", listener.Addr().String())
	fmt.Printf("Listening with ln address: %s \n", adr)

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
			fmt.Printf("Incomming connection from %x on %s\n",
				newConn.RemotePub.SerializeCompressed(), newConn.RemoteAddr().String())

			// don't save host/port for incomming connections
			peerIdx, err := nd.GetPeerIdx(newConn.RemotePub, "")
			if err != nil {
				log.Printf("Listener error: %s\n", err.Error())
				continue
			}

			nd.RemoteMtx.Lock()
			var peer RemotePeer
			peer.Idx = peerIdx
			peer.Con = newConn
			nd.RemoteCons[peerIdx] = &peer
			nd.RemoteMtx.Unlock()

			// each connection to a peer gets its own LNDCReader
			go nd.LNDCReader(&peer)
		}
	}()
	nd.RemoteMtx.Lock()
	nd.LisIpPorts = append(nd.LisIpPorts, lisIpPort)
	nd.RemoteMtx.Unlock()
	return adr, nil
}

// DialPeer makes an outgoing connection to another node.
func (nd *LitNode) DialPeer(connectAdr string) error {

	// parse address and get pkh / host / port
	who, where := lndc.SplitAdrString(connectAdr)

	// sanity check the "who" pkh string
	if !lnutil.LitAdrOK(who) {
		return fmt.Errorf("ln address %s invalid", who)
	}

	// get my private ID key
	idPriv := nd.IdKey()

	// Assign remote connection
	newConn := new(lndc.LNDConn)

	err := newConn.Dial(idPriv, where, who)
	if err != nil {
		return err
	}

	// if connect is successful, either query for already existing peer index, or
	// if the peer is new, make an new index, and save the hostname&port

	// figure out peer index, or assign new one for new peer.  Since
	// we're connecting out, also specify the hostname&port
	peerIdx, err := nd.GetPeerIdx(newConn.RemotePub, newConn.RemoteAddr().String())
	if err != nil {
		return err
	}

	nd.RemoteMtx.Lock()
	var p RemotePeer
	p.Con = newConn
	p.Idx = peerIdx
	nd.RemoteCons[peerIdx] = &p
	nd.RemoteMtx.Unlock()

	// each connection to a peer gets its own LNDCReader
	go nd.LNDCReader(&p)

	return nil
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
		n, err := nd.RemoteCons[msg.PeerIdx].Con.Write(rawmsg)
		if err != nil {
			fmt.Printf("error writing to peer %d: %s\n", msg.PeerIdx, err.Error())
		} else {
			fmt.Printf("type %x %d bytes to peer %d\n", msg.MsgType, n, msg.PeerIdx)
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
	nd.RemoteMtx.Unlock() //TODO: This unlock is in the wrong place...?
	var peers []PeerInfo
	for k, v := range nd.RemoteCons {
		var newPeer PeerInfo
		newPeer.PeerNumber = k
		newPeer.RemoteHost = v.Con.RemoteAddr().String()
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
	return nd.SubWallet.GetPriv(kg)
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
