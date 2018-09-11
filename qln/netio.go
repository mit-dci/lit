package qln

import (
	"fmt"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/lnio"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"log"
	"strings"
)

// GetLisAddressAndPorts .
// Gets the list of ports where LitNode is listening for incoming connections,
// & the connection key
func (nd *LitNode) GetLisAddressAndPorts() (string, []string) {
	return nd.PeerMan.GetExternalAddress(), nd.PeerMan.GetListeningAddrs()
}

// FindPeerIndexByAddress finds the peer index by address.
// TODO Remove this function.
func (nd *LitNode) FindPeerIndexByAddress(lnAdr string) (uint32, error) {
	pm := nd.PeerMan
	p := pm.GetPeer(lnio.LnAddr(lnAdr))
	if p != nil {
		return p.GetIdx(), nil
	}

	return 0, fmt.Errorf("Node %s not found", lnAdr)
}

// TCPListener starts a litNode listening for incoming LNDC connections.
func (nd *LitNode) TCPListener(lisIpPort string) (string, error) {

	err := nd.PeerMan.ListenOnPort(lisIpPort)
	if err != nil {
		return "", err
	}

	lnaddr := nd.PeerMan.GetExternalAddress()

	log.Printf("Listening on %s\n", lisIpPort)
	log.Printf("Listening with ln address: %s \n", lnaddr)

	// Don't announce on the tracker if we are communicating via SOCKS proxy
	if nd.ProxyURL == "" {
		go GoAnnounce(nd.IdKey(), lisIpPort, lnaddr, nd.TrackerURL)
	}

	return lnaddr, nil

}

func GoAnnounce(priv *btcec.PrivateKey, litport string, litadr string, trackerURL string) {
	err := lnutil.Announce(priv, litport, litadr, trackerURL)
	if err != nil {
		logging.Errorf("Announcement error %s", err.Error())
	}
}

// ParseAdrString splits a string like
// "ln1yrvw48uc3atg8e2lzs43mh74m39vl785g4ehem@myhost.co:8191" into a separate
// pkh part and network part, adding the network part if needed
func splitAdrString(adr string) (string, string) {

	if !strings.ContainsRune(adr, ':') && strings.ContainsRune(adr, '@') {
		adr += ":2448"
	}

	idHost := strings.Split(adr, "@")

	if len(idHost) == 1 {
		return idHost[0], ""
	}

	return idHost[0], idHost[1]
}

// DialPeer makes an outgoing connection to another node.
// TODO Remove this.
func (nd *LitNode) DialPeer(connectAdr string) error {

	_, err := nd.PeerMan.TryConnectAddress(connectAdr, nil)
	if err != nil {
		return err
	}

	// TEMP The event handler handles actually setting up the peer in the LitNode

	return nil
}

// OutMessager takes messages from the outbox and sends them to the ether. net.
func (nd *LitNode) OutMessager() {
	for {
		msg := <-nd.OmniOut
		if !nd.ConnectedToPeer(msg.Peer()) {
			logging.Errorf("message type %x to peer %d but not connected\n",
				msg.MsgType(), msg.Peer())
			continue
		}

		//rawmsg := append([]byte{msg.MsgType()}, msg.Data...)
		rawmsg := msg.Bytes() // automatically includes messageType
		nd.RemoteMtx.Lock()   // not sure this is needed...
		n, err := nd.RemoteCons[msg.Peer()].Con.Write(rawmsg)
		if err != nil {
			logging.Errorf("error writing to peer %d: %s\n", msg.Peer(), err.Error())
		} else {
			logging.Infof("type %x %d bytes to peer %d\n", msg.MsgType(), n, msg.Peer())
		}
		nd.RemoteMtx.Unlock()
	}
}

type PeerInfo struct {
	PeerNumber uint32
	RemoteHost string
	LitAdr     string
	Nickname   string
}

func (nd *LitNode) GetConnectedPeerList() []PeerInfo {
	var peers []PeerInfo
	nd.RemoteMtx.Lock()
	defer nd.RemoteMtx.Unlock()
	for k, v := range nd.RemoteCons {
		var newPeer PeerInfo
		var pubArr [33]byte
		copy(pubArr[:], v.Con.RemotePub().SerializeCompressed())
		newPeer.PeerNumber = k
		newPeer.RemoteHost = v.Con.RemoteAddr().String()
		newPeer.Nickname = v.Nickname
		newPeer.LitAdr = lnutil.LitAdrFromPubkey(pubArr)
		peers = append(peers, newPeer)
	}
	return peers
}

// ConnectedToPeer checks whether you're connected to a specific peer
func (nd *LitNode) ConnectedToPeer(peer uint32) bool {
	nd.RemoteMtx.Lock()
	defer nd.RemoteMtx.Unlock()
	_, ok := nd.RemoteCons[peer]
	return ok
}

// IdKey returns the identity private key
func (nd *LitNode) IdKey() *btcec.PrivateKey {
	return nd.IdentityKey
}

// SendChat sends a text string to a peer
func (nd *LitNode) SendChat(peer uint32, chat string) error {
	if !nd.ConnectedToPeer(peer) {
		return fmt.Errorf("Not connected to peer %d", peer)
	}

	outMsg := lnutil.NewChatMsg(peer, chat)

	nd.OmniOut <- outMsg

	return nil
}
