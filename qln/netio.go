package qln

import (
	"fmt"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
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
	p := pm.GetPeer(lncore.LnAddr(lnAdr))
	if p != nil {
		return p.GetIdx(), nil
	}

	return 0, fmt.Errorf("Node %s not found", lnAdr)
}

// TCPListener starts a litNode listening for incoming LNDC connections.
func (nd *LitNode) TCPListener(lisIpPort int) (string, error) {

	logging.Debugf("Trying to listen on %s\n", lisIpPort)

	err := nd.PeerMan.ListenOnPort(lisIpPort)
	if err != nil {
		return "", err
	}

	lnaddr := nd.PeerMan.GetExternalAddress()

	logging.Infof("Listening with ln address: %s \n", lnaddr)

	// Don't announce on the tracker if we are communicating via SOCKS proxy
	if nd.ProxyURL == "" {
		go GoAnnounce(nd.IdKey(), lisIpPort, lnaddr, nd.TrackerURL)
	}

	return lnaddr, nil

}

func GoAnnounce(priv *koblitz.PrivateKey, litport int, litadr string, trackerURL string) {
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

func (nd *LitNode) tmpSendLitMsg(msg lnutil.LitMsg) {

	if !nd.ConnectedToPeer(msg.Peer()) {
		logging.Debugf("message type %x to peer %d but not connected\n",
			msg.MsgType(), msg.Peer())
		return
	}

	buf := msg.Bytes()

	// Just wrap it and forward it off to the underlying infrastructure.
	// There's some byte fenagling that we have to do to get all this to work right.
	np := nd.PeerMan.GetPeerByIdx(int32(msg.Peer()))
	np.SendImmediateMessage(LitMsgWrapperMessage{buf[0], buf[1:]}) // being blocking might not be a huge issue here possibly?

}

type SimplePeerInfo struct {
	PeerNumber int32
	RemoteHost string
	Nickname   string
	LitAdr     string
}

func (nd *LitNode) GetConnectedPeerList() []SimplePeerInfo {
	peers := make([]SimplePeerInfo, 0)
	for k, _ := range nd.PeerMap {
		spi := SimplePeerInfo{
			PeerNumber: nd.PeerMan.GetPeerIdx(k),
			RemoteHost: k.GetRemoteAddr(),
			Nickname:   k.GetNickname(),
			LitAdr:     string(k.GetLnAddr()),
		}
		peers = append(peers, spi)
	}
	return peers
}

// ConnectedToPeer checks whether you're connected to a specific peer
func (nd *LitNode) ConnectedToPeer(peer uint32) bool {
	// TODO Upgrade this to the new system.
	nd.RemoteMtx.Lock()
	defer nd.RemoteMtx.Unlock()
	_, ok := nd.RemoteCons[peer]
	return ok
}

// IdKey returns the identity private key
func (nd *LitNode) IdKey() *koblitz.PrivateKey {
	return nd.IdentityKey
}

// SendChat sends a text string to a peer
func (nd *LitNode) SendChat(peer uint32, chat string) error {
	if !nd.ConnectedToPeer(peer) {
		return fmt.Errorf("Not connected to peer %d", peer)
	}

	outMsg := lnutil.NewChatMsg(peer, chat)

	nd.tmpSendLitMsg(outMsg)

	return nil
}
