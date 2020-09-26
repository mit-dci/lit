package lnp2p

//"crypto/ecdsa" // TODO Use ecdsa not koblitz
import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"sync"
	"time"
	"math"

	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/nat"
	"github.com/mit-dci/lit/portxo"
)

type privkey *koblitz.PrivateKey
type pubkey *koblitz.PublicKey

// MaxNodeCount is the size of the peerIdx->LnAddr array.
// TEMP This shouldn't be necessary.
const MaxNodeCount = 1024

// PeerManager .
type PeerManager struct {

	// Biographical.
	idkey       privkey
	peerdb      lncore.LitPeerStorage
	ebus        *eventbus.EventBus
	mproc       MessageProcessor
	netsettings *NetSettings

	// Peer tracking.
	peers   []lncore.LnAddr // compatibility
	peerMap map[lncore.LnAddr]*Peer

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
func NewPeerManager(rootkey *hdkeychain.ExtendedKey, pdb lncore.LitPeerStorage, trackerURL string, bus *eventbus.EventBus, ns *NetSettings) (*PeerManager, error) {
	k, err := computeIdentKeyFromRoot(rootkey)
	if err != nil {
		return nil, err
	}

	pm := &PeerManager{
		idkey:          k,
		peerdb:         pdb,
		ebus:           bus,
		mproc:          NewMessageProcessor(),
		netsettings:    ns,
		peers:          make([]lncore.LnAddr, MaxNodeCount),
		peerMap:        map[lncore.LnAddr]*Peer{},
		listeningPorts: map[int]*listeningthread{},
		sending:        false,
		trackerURL:     trackerURL,
		outqueue:       make(chan outgoingmsg, outgoingbuf),
		mtx:            &sync.Mutex{},
	}


	// Clear ChunksOfMsg in case of incomplete chunks transmittion.
	// Try to clean the map every 5 minutes. Therefore message have to
	// be transmitted within a 5 minutes.
	go func(){

		for {

			time.Sleep(5 * time.Minute)
	
			for k := range pm.mproc.ChunksOfMsg {
				tdelta := time.Now().UnixNano() - k
				if tdelta > 3*int64(math.Pow10(11)) {
					delete(pm.mproc.ChunksOfMsg, k)
				}
			}
		}
	}()

	return pm, nil
}

// GetMessageProcessor gets the message processor for this peer manager that's passed incoming messasges from peers.
func (pm *PeerManager) GetMessageProcessor() *MessageProcessor {
	return &pm.mproc
}

// GetExternalAddress returns the human-readable LN address
func (pm *PeerManager) GetExternalAddress() string {
	idk := pm.idkey // lol
	c := koblitz.PublicKey(ecdsa.PublicKey(idk.PublicKey))
	addr := convertPubkeyToLitAddr(pubkey(&c))
	return string(addr)
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
func (pm *PeerManager) GetPeerIdx(peer *Peer) uint32 {
	if peer.idx == nil {
		return 0
	}
	return *peer.idx
}

// GetPeer returns the peer with the given lnaddr.
func (pm *PeerManager) GetPeer(lnaddr lncore.LnAddr) *Peer {
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

// TryConnectAddress attempts to connect to the specified LN address.
func (pm *PeerManager) TryConnectAddress(addr string) (*Peer, error) {

	// Figure out who we're trying to connect to.
	who, where := splitAdrString(addr)
	if where == "" {
		ipv4, _, err := lnutil.Lookup(addr, pm.trackerURL, "")
		if err != nil {
			return nil, err
		}
		where = fmt.Sprintf("%s:2448", ipv4)
	}

	lnwho, err := lncore.ParseLnAddr(who)
	if err != nil {
		return nil, err
	}

	x, y := pm.tryConnectPeer(where, &lnwho)
	return x, y

}

func (pm *PeerManager) tryConnectPeer(netaddr string, lnaddr *lncore.LnAddr) (*Peer, error) {

	// lnaddr check, to make sure that we do the right thing.
	if lnaddr == nil {
		return nil, fmt.Errorf("connection to a peer with unknown lnaddr not supported yet")
	}

	dialer := net.Dial

	// Use a proxy server if applicable.
	ns := pm.netsettings
	if ns != nil && ns.ProxyAddr != nil {
		d, err := connectToProxyTCP(*ns.ProxyAddr, ns.ProxyAuth)
		if err != nil {
			return nil, err
		}
		dialer = d
	}

	// Create the connection.
	lndcconn, err := lndc.Dial(pm.idkey, netaddr, string(*lnaddr), dialer)
	if err != nil {
		return nil, err
	}

	// Try to set up the new connection.
	p, err := pm.handleNewConnection(lndcconn, *lnaddr)
	if err != nil {
		return nil, err
	}

	// Now start listening for inbound traffic.
	// (it *also* took me a while to realize I forgot *this*)
	go processConnectionInboundTraffic(p, pm)

	// Return
	return p, nil

}

func (pm *PeerManager) handleNewConnection(conn *lndc.Conn, expectedAddr lncore.LnAddr) (*Peer, error) {

	// Now that we've got the connection, actually create the peer object.
	pk := pubkey(conn.RemotePub())
	rlitaddr := convertPubkeyToLitAddr(pk)

	if rlitaddr != expectedAddr {
		conn.Close()
		return nil, fmt.Errorf("peermgr: Connection init error, expected addr %s got addr %s", expectedAddr, rlitaddr)
	}

	p := &Peer{
		lnaddr:   rlitaddr,
		nickname: nil,
		conn:     conn,
		idpubkey: pk,

		// TEMP
		idx: nil,
	}

	pi, err := pm.peerdb.GetPeerInfo(expectedAddr)
	if err != nil {
		logging.Errorf("peermgr: Problem loading peer info from DB: %s\n", err.Error())
		// don't kill the connection?
	}

	if pi == nil {
		pidx, err := pm.peerdb.GetUniquePeerIdx()
		if err != nil {
			logging.Errorf("Problem getting unique peeridx from DB: %s\n", err.Error())
		} else {
			p.idx = &pidx
		}
		raddr := conn.RemoteAddr().String()
		pi = &lncore.PeerInfo{
			LnAddr:   &rlitaddr,
			Nickname: nil,
			NetAddr:  &raddr,
			PeerIdx:  pidx,
		}
		err = pm.peerdb.AddPeer(p.GetLnAddr(), *pi)
		if err != nil {
			logging.Errorf("Error saving new peer to DB: %s\n", err.Error())
		}
	} else {
		p.nickname = pi.Nickname
		// TEMP
		p.idx = &pi.PeerIdx
	}

	// Register the peer we just connected to!
	// (it took me a while to realize I forgot this)
	pm.registerPeer(p)

	// Now actually return the peer.
	return p, nil

}

func (pm *PeerManager) registerPeer(peer *Peer) {

	lnaddr := peer.lnaddr

	// We're making changes to the manager so keep stuff away while we set up.
	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	logging.Infof("peermgr: New peer %s\n", peer.GetLnAddr())

	// Append peer to peer list and add to peermap
	pm.peers[int(*peer.idx)] = lnaddr // TEMP This idx logic is a litte weird.
	pm.peerMap[lnaddr] = peer
	peer.pmgr = pm

	// Announce the peer has been added.
	e := NewPeerEvent{
		Addr:            lnaddr,
		Peer:            peer,
		RemoteInitiated: false,

		// TODO Remove these.
		RemotePub: peer.idpubkey,
		Conn:      peer.conn,
	}
	pm.ebus.Publish(e)

}

func (pm *PeerManager) unregisterPeer(peer *Peer) {

	// Again, sensitive changes we should get a lock to do first.
	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	logging.Infof("peermgr: Unregistering peer: %s\n", peer.GetLnAddr())

	// Remove the peer idx entry.
	idx := pm.GetPeerIdx(peer)
	pm.peers[idx] = ""

	// Remove the actual peer entry.
	pm.peerMap[peer.GetLnAddr()] = nil

	// More cleanup.
	peer.conn = nil
	peer.idx = nil
	peer.pmgr = nil

}

// DisconnectPeer disconnects a peer from ourselves and does relevant cleanup.
func (pm *PeerManager) DisconnectPeer(peer *Peer) error {

	err := peer.conn.Close()
	if err != nil {
		return err
	}

	// This will cause the peer disconnect event to be raised when the reader
	// goroutine started to exit and run the unregistration

	return nil

}

// ListenOnPort attempts to start a goroutine lisening on the port.
func (pm *PeerManager) ListenOnPort(port int) error {

	// Do NAT setup stuff.
	ns := pm.netsettings
	if ns != nil && ns.NatMode != nil {

		// Do some type juggling.
		lisPort := uint16(port)

		// Actually figure out what we're going to do.
		if *ns.NatMode == "upnp" {
			// Universal Plug-n-Play
			logging.Infof("Attempting port forwarding via UPnP...")
			err := nat.SetupUpnp(lisPort)
			if err != nil {
				return err
			}
		} else if *ns.NatMode == "pmp" {
			// NAT Port Mapping Protocol
			timeout := time.Duration(10 * time.Second)
			logging.Infof("Attempting port forwarding via PMP...")
			_, err := nat.SetupPmp(timeout, lisPort)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("invalid NAT type: %s", *ns.NatMode)
		}
	}

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
	// TODO Listen on proxy if possible?
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
	if !pm.mproc.IsActive() {
		pm.mproc.Activate()
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

	pi, err := pm.peerdb.GetPeerInfo(peer.GetLnAddr())
	if err != nil {
		return err
	}

	pi.PeerIdx = idx

	return pm.peerdb.UpdatePeer(peer.GetLnAddr(), pi)

}
