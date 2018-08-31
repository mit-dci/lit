package lnp2p

//"crypto/ecdsa" // TODO Use ecdsa not btcec
import (
	"crypto/ecdsa"
	"fmt"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnio"
	"github.com/mit-dci/lit/portxo"
	"log"
	"net"
	"sync"
)

type privkey *btcec.PrivateKey
type pubkey *btcec.PublicKey

// PeerManager .
type PeerManager struct {

	// Biographical.
	idkey  privkey
	peerdb lnio.LitPeerStorage
	ebus   *eventbus.EventBus
	mproc  MessageProcessor

	// Peer tracking.
	peers   []lnio.LnAddr // compatibility
	peerMap map[lnio.LnAddr]*Peer

	// Accepting connections.
	listeningPorts map[string]*listeningthread

	// Outgoing messages.
	sending  bool
	outqueue chan outgoingmsg

	// Sync.
	mtx *sync.Mutex
}

const outgoingbuf = 16

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
		mproc:          NewMessageProcessor(),
		peers:          make([]lnio.LnAddr, 1),
		peerMap:        map[lnio.LnAddr]*Peer{},
		listeningPorts: map[string]*listeningthread{},
		sending:        true,
		outqueue:       make(chan outgoingmsg, outgoingbuf),
		mtx:            &sync.Mutex{},
	}

	return pm, nil
}

// GetMessageProcessor gets the message processor for this peer manager that's passed incoming messasges from peers.
func (pm *PeerManager) GetMessageProcessor() *MessageProcessor {
	return &pm.mproc
}

// GetExternalAddress returns the human-readable LN address
func (pm *PeerManager) GetExternalAddress() string {
	idk := pm.idkey // lol
	c := btcec.PublicKey(ecdsa.PublicKey(idk.PublicKey))
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
func (pm *PeerManager) GetPeerIdx(peer *Peer) int32 {
	for i, p := range pm.peers {
		if pm.peerMap[p] == peer {
			return int32(i)
		}
	}
	return 0
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
	if id <= 0 || id >= int32(len(pm.peers)) {
		return nil
	}
	return pm.peerMap[pm.peers[id]]
}

// TryConnectAddress attempts to connect to the specified LN address.
func (pm *PeerManager) TryConnectAddress(addr string, proxy *ProxySettings) (*Peer, error) {

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
		log.Printf("Problem loading peer info from DB: %s\n", err.Error())
		// don't kill the connection?
	}

	pnick := ""
	if pi != nil {
		pnick = *pi.Nickname
	}

	// Now that we've got the connection, actually create the peer object.
	pk := pubkey(lndcconn.RemotePub())
	p := &Peer{
		lnaddr:   convertPubkeyToLitAddr(pk),
		nickname: &pnick,
		conn:     lndcconn,
		idpubkey: pk,
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

	lnaddr := peer.lnaddr

	// We're making changes to the manager so keep stuff away while we set up.
	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	log.Printf("peermgr: New peer %s\n", peer.GetLnAddr())

	// Append peer to peer list and add to peermap
	pidx := uint32(len(pm.peers))
	pm.peers = append(pm.peers, lnaddr)
	pm.peerMap[lnaddr] = peer
	peer.idx = &pidx
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

	log.Printf("peermgr: Unregistering peer: %s\n", peer.GetLnAddr())

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
		log.Printf("listening failed: %s\n", err.Error())
		pm.ebus.Publish(StopListeningPortEvent{
			ListenAddr: addr,
			Reason:     "initfail",
		})
		return err
	}

	threadobj.listener = listener

	// Install the thread object.
	pm.mtx.Lock()
	pm.listeningPorts[addr] = threadobj
	pm.mtx.Unlock()

	// Actually start it
	go acceptConnections(listener, addr, pm)
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

/*
	TODO Implement this stuff again.

	// do UPnP / pmp port forwarding
	// fatal if we aren't able to port forward via upnp
	if len(nd.Nat) > 0 {
		listenPort, err := strconv.Atoi(lisIpPort[1:])
		if err != nil {
			log.Println("Invalid port number, returning")
			return "", err
		}
		if nd.Nat == "upnp" {
			log.Println("Port forwarding via UPnP on port", lisIpPort[1:])
			err := nat.SetupUpnp(uint16(listenPort))
			if err != nil {
				fmt.Printf("Unable to setup Upnp %v\n", err)
				log.Fatal(err) // error out if we can't connect via UPnP
			}
			log.Println("Forwarded port via UPnP")
		} else if nd.Nat == "pmp" {
			discoveryTimeout := time.Duration(10 * time.Second)
			log.Println("Port forwarding via NAT Pmp on port", lisIpPort[1:])
			_, err := nat.SetupPmp(discoveryTimeout, uint16(listenPort))
			if err != nil {
				err := fmt.Errorf("Unable to discover a "+
					"NAT-PMP enabled device on the local "+
					"network: %v", err)
				log.Fatal(err) // error out if we can't connect via Pmp
			} else {
				log.Println("Invalid NAT punching option")
			}
		}
	}
*/

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
	pm.outqueue <- outgoingmsg{nil, nil} // stops the sending goroutine
	pm.sending = false
	return nil
}

func (pm *PeerManager) queueMessageToPeer(peer *Peer, msg Message) error {
	if !pm.sending {
		return fmt.Errorf("sending is disabled on this peer manager, need to start it?")
	}
	pm.outqueue <- outgoingmsg{peer, &msg}
	return nil
}
