package lnp2p

//"crypto/ecdsa" // TODO Use ecdsa not btcec
import (
	"fmt"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnio"
	"github.com/mit-dci/lit/portxo"
	"net"
	"sync"
)

type privkey *btcec.PrivateKey
type pubkey *btcec.PublicKey

// PeerManager .
type PeerManager struct {
	idkey  privkey
	peerdb lnio.LitPeerStorage
	ebus   *eventbus.EventBus

	peers   []lnio.LnAddr // compatibility
	peerMap map[lnio.LnAddr]*Peer

	listeningPorts map[string]*listeningthread

	mtx *sync.Mutex
}

// ProxySettings .
type ProxySettings struct {
	// TODO
}

// NewPeerManager creates a peer manager from a root key
func NewPeerManager(rootkey *hdkeychain.ExtendedKey, pdb lnio.LitPeerStorage, bus *eventbus.EventBus) (*PeerManager, error) {
	k, err := computeIdentKeyFromRoot(rootkey)
	if err != nil {
		return nil, err
	}

	pm := &PeerManager{
		idkey:          k,
		peerdb:         pdb,
		ebus:           bus,
		peers:          []lnio.LnAddr{},
		peerMap:        map[lnio.LnAddr]*Peer{},
		listeningPorts: map[string]*listeningthread{},
		mtx:            &sync.Mutex{},
	}

	return pm, nil
}

func computeIdentKeyFromRoot(rootkey *hdkeychain.ExtendedKey) (privkey, error) {
	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31  // from bip44, but not actually sensical in this context
	kg.Step[1] = 513 | 1<<31 // magic
	kg.Step[2] = 9 | 1<<31   // magic
	kg.Step[3] = 0 | 1<<31
	kg.Step[4] = 0 | 1<<31
	k, err := kg.DerivePrivateKey(rootkey)
	if err != nil {
		return nil, err
	}
	return privkey(k), nil
}

// GetPeerIdx is a convenience function for working with older code.
func (pm *PeerManager) GetPeerIdx(peer *Peer) int32 {
	for i, p := range pm.peers {
		if pm.peerMap[p] == peer {
			return int32(i)
		}
	}
	return -1
}

// GetPeer returns the peer with the given lnaddr.
func (pm *PeerManager) GetPeer(lnaddr lnio.LnAddr) *Peer {
	p, ok := pm.peerMap[lnaddr]
	if !ok {
		return nil
	}
	return p
}

// GetPeerByIdx is a compatibility function for getting a peer by its "peer id".
func (pm *PeerManager) GetPeerByIdx(id int32) *Peer {
	if id < 0 || id >= int32(len(pm.peers)) {
		return nil
	}
	return pm.peerMap[pm.peers[id]]
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
	pidx := uint32(len(pm.peers))
	pm.peers = append(pm.peers, lnaddr)
	pm.peerMap[lnaddr] = peer
	peer.idx = &pidx

	// Announce the peer has been added.
	e := NewPeerEvent{
		Addr:            lnaddr,
		Peer:            peer,
		RemoteInitiated: false,
	}
	pm.ebus.Publish(e)

}

// ListenOnPort attempts to start a goroutine lisening on the port.
func (pm *PeerManager) ListenOnPort(addr string) error {

	threadobj := &listeningthread{
		listener: nil,
	}

	// Publish the new thread
	res, err := pm.ebus.Publish(NewListeningPortEvent{addr})
	if err != nil {
		return err
	}

	if !res {
		return fmt.Errorf("listen cancelled by event handler")
	}

	// TODO UPnP and PMP NAT traversal.

	// Try to start listening.
	listener, err := lndc.NewListener(pm.idkey, addr)
	if err != nil {
		pm.ebus.Publish(StopListeningPortEvent{
			ListenAddr: addr,
			Reason:     "initfail",
		})
		return err
	}

	// Install the thread object.
	pm.mtx.Lock()
	pm.listeningPorts[addr] = threadobj
	pm.mtx.Unlock()

	// Actually start it
	go acceptConnections(listener, addr, pm)
	return nil

}

// StopListening closes the socket listened on the given address, stopping the goroutine.
func (pm *PeerManager) StopListening(addr string) error {

	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	// This will interrupt the .Accept() call in the other goroutine, and handle cleanup for us.
	lt, ok := pm.listeningPorts[addr]
	if !ok {
		return fmt.Errorf("not listening")
	}

	lt.listener.Close()
	return nil

}
