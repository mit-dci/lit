package lnp2p

//"crypto/ecdsa" // TODO Use ecdsa not btcec
import (
	"fmt"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnio"
	"net"
	"sync"
)

type privkey *btcec.PrivateKey

// PeerManager .
type PeerManager struct {
	idkey  privkey
	peerdb lnio.LitPeerStorage
	// TODO event bus

	peers   []*Peer
	peerMap map[lnio.LnAddr]*Peer

	mtx *sync.Mutex
}

// Peer .
type Peer struct {
	lnaddr   lnio.LnAddr
	nickname *string
	conn     *lndc.Conn
	//chans    map[lnio.ChannelHandle]*lnio.ChannelInfo
	// TODO
}

// ProxySettings .
type ProxySettings struct {
	// TODO
}

// GetPeerIdx is a convenience function for working with older code.
func (pm *PeerManager) GetPeerIdx(peer *Peer) int32 {
	for i, p := range pm.peers {
		if p == peer {
			return int32(i)
		}
	}
	return -1
}

func (pm *PeerManager) tryConnectAddress(addr string, proxy *ProxySettings) (*Peer, error) {

	// Figure out who we're trying to connect to.
	who, where := splitAdrString(addr)
	if where == "" {
		// TODO Do lookup.
	}

	lnwho := lnio.LnAddr(who)
	x, y := pm.tryConnectPeer(where, &lnwho, proxy)
	return x, y

}

func (pm *PeerManager) tryConnectPeer(netaddr string, lnaddr *lnio.LnAddr, proxy *ProxySettings) (*Peer, error) {

	// lnaddr check, to make sure that we do the right thing.
	if lnaddr == nil {
		return nil, fmt.Errorf("connection to a peer with unknown lnaddr not supported yet")
	}

	// Set up the connection.
	lndcconn, err := lndc.Dial(pm.idkey, netaddr, string(*lnaddr), net.Dial)
	if err != nil {
		return nil, err
	}

	pi, err := pm.peerdb.GetPeerInfo(*lnaddr)
	if err != nil {
		println(err)
		// TODO deal with this
	}

	// Now that we've got the connection, actually create the peer object.
	p := &Peer{
		nickname: pi.Nickname,
		conn:     lndcconn,
	}

	// Return
	return p, nil

}

func (pm *PeerManager) registerPeer(peer *Peer) {

	lnaddr := peer.lnaddr

	// We're making changes to the manager so keep stuff away while we set up.
	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	// Append peer to peer list and add to peermap
	pm.peers = append(pm.peers, peer)
	pm.peerMap[lnaddr] = peer

	// Figure out which channels were with this peer.

}
