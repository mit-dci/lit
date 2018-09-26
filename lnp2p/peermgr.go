package lnp2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/nat"
	"github.com/mit-dci/lit/portxo"
	"golang.org/x/net/proxy"
)

type privkey *koblitz.PrivateKey
type pubkey *koblitz.PublicKey

// MaxNodeCount is the size of the peerIdx->Addr array.
// TEMP This shouldn't be necessary.
const MaxNodeCount = 1024

// PeerManager .
type PeerManager struct {

	// Biographical.
	idkey  privkey
	peerdb lncore.LitPeerStorage
	ebus   *eventbus.EventBus
	MProc  MessageProcessor

	// Peer tracking.
	peers   []string // compatibility
	peerMap map[string]*Peer

	// Accepting connections.
	listeningPorts map[int]*listeningthread

	// Outgoing messages.
	sending  bool
	outqueue chan outgoingmsg

	// Tracker
	trackerURL string

	// Sync.
	mtx *sync.Mutex
}

const outgoingbuf = 16

// NetSettings is a container struct for misc network settings like NAT
// holepunching and proxies.
type NetSettings struct {
	NatMode *string `json:"natmode"`

	ProxyAddr *string `json:"proxyserv"`
	ProxyAuth *string `json:"proxyauth"`
}

// NewPeerManager creates a peer manager from a root key
func NewPeerManager(rootkey *hdkeychain.ExtendedKey, pdb lncore.LitPeerStorage, trackerURL string, bus *eventbus.EventBus) (*PeerManager, error) {
	k, err := computeIdentKeyFromRoot(rootkey)
	if err != nil {
		return nil, err
	}

	pm := &PeerManager{
		idkey:          k,
		peerdb:         pdb,
		ebus:           bus,
		MProc:          NewMessageProcessor(),
		peers:          make([]string, MaxNodeCount),
		peerMap:        map[string]*Peer{},
		listeningPorts: map[int]*listeningthread{},
		sending:        false,
		trackerURL:     trackerURL,
		outqueue:       make(chan outgoingmsg, outgoingbuf),
		mtx:            &sync.Mutex{},
	}

	return pm, nil
}

// GetExternalAddress returns the human-readable LN address
func (pm *PeerManager) GetExternalAddress() string {
	idk := pm.idkey // lol
	c := koblitz.PublicKey(ecdsa.PublicKey(idk.PublicKey))
	addr := lnutil.ConvertPubkeyToLitAddr(pubkey(&c))
	return addr
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

// GetPeer returns the peer with the given addr.
func (pm *PeerManager) GetPeer(addr string) *Peer {
	p, ok := pm.peerMap[addr]
	logging.Infof("%v -> %v (%t)\n", addr, p, ok)
	if !ok {
		return nil
	}
	return p
}

// GetPeerByIdx is a compatibility function for getting a peer by its "peer id".
func (pm *PeerManager) GetPeerByIdx(id int32) *Peer {
	if id <= 0 || id >= int32(len(pm.peers)) {
		return nil
	}
	return pm.peerMap[pm.peers[id]]
}

// TryConnectAddress is a handler function for tryConnectPeer and
// attempts to connect to the specified LN address.
func (pm *PeerManager) TryConnectAddress(addr string, settings *NetSettings) (*Peer, error) {

	// Figure out who we're trying to connect to.
	who, where := lnutil.ParseAdrString(addr)
	if where == "" {
		ipv4, _, err := lnutil.Lookup(addr, pm.trackerURL, "")
		if err != nil {
			return nil, err
		}
		where = fmt.Sprintf("%s:2448", ipv4)
	}

	return pm.tryConnectPeer(where, who, settings)
}

func connectToProxyTCP(addr string, auth *string) (func(string, string) (net.Conn, error), error) {
	// Authentication is good.  Use it if it's there.
	var pAuth *proxy.Auth
	if auth != nil {
		parts := strings.SplitN(*auth, ":", 2)
		pAuth = &proxy.Auth{
			User:     parts[0],
			Password: parts[1],
		}
	}

	// Actually attempt to connect to the SOCKS Proxy.
	d, err := proxy.SOCKS5("tcp", addr, pAuth, proxy.Direct)
	if err != nil {
		return nil, err
	}

	return d.Dial, nil
}

func (pm *PeerManager) tryConnectPeer(netaddr string, addr string, settings *NetSettings) (*Peer, error) {

	// if its nil, return since we don't support that yet
	if len(addr) == 0 {
		return nil, fmt.Errorf("connection to a peer with unknown address not supported yet")
	}

	// Do NAT setup stuff.
	if settings != nil && settings.NatMode != nil {

		// Do some type juggling.
		x, err := strconv.Atoi(netaddr[1:])
		if err != nil {
			return nil, err
		}
		lisPort := uint16(x) // more type juggling

		// choose between upnp and natpmp modes
		if *settings.NatMode == "upnp" {
			// Universal Plug-n-Play
			logging.Infof("Attempting port forwarding via UPnP...")
			err = nat.SetupUpnp(lisPort)
			if err != nil {
				return nil, err
			}
		} else if *settings.NatMode == "pmp" {
			// NAT Port Mapping Protocol
			timeout := time.Duration(10 * time.Second)
			logging.Infof("Attempting port forwarding via PMP...")
			_, err = nat.SetupPmp(timeout, lisPort)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid NAT type: %s", *settings.NatMode)
		}
	}

	dialer := net.Dial

	// Use a proxy server if applicable.
	if settings != nil && settings.ProxyAddr != nil {
		d, err := connectToProxyTCP(*settings.ProxyAddr, settings.ProxyAuth)
		if err != nil {
			return nil, err
		}
		dialer = d
	}

	// Set up the connection.
	lndcconn, err := lndc.Dial(pm.idkey, netaddr, addr, dialer)
	if err != nil {
		return nil, err
	}

	pi, err := pm.peerdb.GetPeerInfo(addr)
	if err != nil {
		logging.Errorf("Problem loading peer info from DB: %s\n", err.Error())
		return nil, err
	}

	// Now that we've got the connection, actually create the peer object.
	pk := pubkey(lndcconn.RemotePub())
	rlitaddr := lnutil.ConvertPubkeyToLitAddr(pk)
	p := &Peer{
		Addr:     rlitaddr,
		Nickname: "",
		conn:     lndcconn,
		Pubkey:   pk,
	}

	if len(pi.Addr) != 0 {
		pidx, err := pm.peerdb.GetUniquePeerIdx()
		if err != nil {
			logging.Errorf("Problem getting unique peeridx from DB: %s\n", err.Error())
		} else {
			p.Idx = pidx
		}
		raddr := lndcconn.RemoteAddr().String()
		pi = lncore.PeerInfo{
			Addr:     rlitaddr,
			Nickname: "",
			NetAddr:  raddr,
			PeerIdx:  pidx,
		}
		err = pm.peerdb.AddPeer(p.Addr, pi)
		if err != nil {
			logging.Errorf("Error saving new peer to DB: %s\n", err.Error())
		}
	} else {
		p.Nickname = pi.Nickname
		p.Idx = pi.PeerIdx
	}

	if len(pi.Addr) == 0 { // TODO: Remove this
		pidx, err := pm.peerdb.GetUniquePeerIdx()
		if err != nil {
			logging.Errorf("Problem getting unique peeridx from DB: %s\n", err.Error())
		} else {
			p.Idx = pidx
		}
		raddr := lndcconn.RemoteAddr().String()
		pi = lncore.PeerInfo{
			Addr:     rlitaddr,
			Nickname: "",
			NetAddr:  raddr,
			PeerIdx:  pidx,
		}
		err = pm.peerdb.AddPeer(p.Addr, pi)
		if err != nil {
			logging.Errorf("Error saving new peer to DB: %s\n", err.Error())
		}
	} else {
		p.Nickname = pi.Nickname
		// TEMP
		p.Idx = pi.PeerIdx
	}

	// Register the peer we just connected to!
	// (it took me a while to realize I forgot this)
	pm.registerPeer(p)

	// Now start listening for inbound traffic.
	// (it *also* took me a while to realize I forgot *this*)
	go processConnectionInboundTraffic(p, pm)

	// Return
	return p, nil

}

func (pm *PeerManager) registerPeer(peer *Peer) {

	// We're making changes to the manager so keep stuff away while we set up.
	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	logging.Infof("peermgr: New peer %s\n", peer.Addr)

	// Append peer to peer list and add to peermap
	pm.peers[peer.Idx] = peer.Addr // TEMP This idx logic is a litte weird.
	pm.peerMap[peer.Addr] = peer
	peer.pmgr = pm

	// Announce the peer has been added.
	e := NewPeerEvent{
		Addr:            peer.Addr,
		Peer:            peer,
		RemoteInitiated: false,

		// TODO Remove these.
		RemotePub: peer.Pubkey,
		Conn:      peer.conn,
	}
	pm.ebus.Publish(e)

}

func (pm *PeerManager) unregisterPeer(peer *Peer) {

	// Again, sensitive changes we should get a lock to do first.
	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	logging.Infof("peermgr: Unregistering peer: %s\n", peer.Addr)

	// Remove the peer idx entry.
	idx := peer.Idx
	pm.peers[idx] = ""

	// Remove the actual peer entry.
	pm.peerMap[peer.Addr] = nil

	// More cleanup.
	peer.conn = nil
	peer.Idx = 0
	peer.pmgr = nil

}

// DisconnectPeer disconnects a peer from ourselves and does relevant cleanup.
func (pm *PeerManager) DisconnectPeer(peer *Peer) error {
	// This will cause the peer disconnect event to be raised when the reader
	// goroutine started to exit and run the unregistration
	return peer.conn.Close()
}

// ListenOnPort attempts to start a goroutine lisening on the port.
func (pm *PeerManager) ListenOnPort(port int) error {

	threadobj := &listeningthread{
		listener: nil,
	}

	// Publish the new thread
	res, err := pm.ebus.Publish(NewListeningPortEvent{port})
	if err != nil {
		return err
	}

	if !res {
		return fmt.Errorf("listen cancelled by event handler")
	}

	// Try to start listening.
	logging.Info("PORT: ", port)
	listener, err := lndc.NewListener(pm.idkey, port)
	if err != nil {
		logging.Errorf("listening failed: %s\n", err.Error())
		logging.Info(err)
		pm.ebus.Publish(StopListeningPortEvent{
			Port:   port,
			Reason: "initfail",
		})
		return err
	}

	threadobj.listener = listener

	// Install the thread object.
	pm.mtx.Lock()
	pm.listeningPorts[port] = threadobj
	pm.mtx.Unlock()

	// Activate the MessageProcessor if we haven't yet.
	if !pm.MProc.IsActive() {
		pm.MProc.Activate()
	}

	// Actually start it
	go acceptConnections(listener, port, pm)

	return nil
}

// GetListeningAddrs returns the listening addresses.
func (pm *PeerManager) GetListeningAddrs() []string {
	pm.mtx.Lock()
	defer pm.mtx.Unlock()
	ports := make([]string, 0)
	for _, t := range pm.listeningPorts {
		ports = append(ports, t.listener.Addr().String())
	}
	return ports
}

// StopListening closes the socket listened on the given address, stopping the goroutine.
func (pm *PeerManager) StopListening(port int) error {

	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	// This will interrupt the .Accept() call in the other goroutine, and handle cleanup for us.
	lt, ok := pm.listeningPorts[port]
	if !ok {
		return fmt.Errorf("not listening")
	}

	lt.listener.Close()
	return nil

}

// StartSending starts a goroutine to start sending queued messages out to peers.
func (pm *PeerManager) StartSending() error {
	if pm.sending {
		return fmt.Errorf("already sending")
	}
	pm.sending = true
	go sendMessages(pm.outqueue)
	return nil
}

// StopSending has us stop sending new messages to peers.
func (pm *PeerManager) StopSending() error {
	if !pm.sending {
		return fmt.Errorf("not sending")
	}
	fc := make(chan error)
	pm.outqueue <- outgoingmsg{nil, nil, &fc} // sends a message to stop the goroutine

	<-fc // wait for the queue to flush
	pm.sending = false

	return nil
}

func (pm *PeerManager) queueMessageToPeer(peer *Peer, msg Message, ec *chan error) error {
	if !pm.sending {
		return fmt.Errorf("sending is disabled on this peer manager, need to start it?")
	}
	pm.outqueue <- outgoingmsg{peer, &msg, ec}
	return nil
}

// TmpHintPeerIdx sets the peer idx hint for a particular peer.
// TEMP This should be removed at some point in the future.
func (pm *PeerManager) TmpHintPeerIdx(peer *Peer, idx uint32) error {

	pi, err := pm.peerdb.GetPeerInfo(peer.Addr)
	if err != nil {
		return err
	}

	pi.PeerIdx = idx

	return pm.peerdb.UpdatePeer(peer.Addr, pi)
}
