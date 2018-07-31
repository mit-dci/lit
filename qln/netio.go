package qln

import (
	"fmt"
	"log"
	"net"
	"strings"
	"strconv"
	"time"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	nat "github.com/mit-dci/lit/nat"
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
	listener, err := lndc.NewListener(nd.IdKey(), lisIpPort)
	if err != nil {
		return "", err
	}

	var idPub [33]byte
	copy(idPub[:], idPriv.PubKey().SerializeCompressed())

	adr := lnutil.LitAdrFromPubkey(idPub)

	// Don't announce on the tracker if we are communicating via SOCKS proxy
	if nd.ProxyURL == "" {
		err = Announce(idPriv, lisIpPort, adr, nd.TrackerURL)
		if err != nil {
			log.Printf("Announcement error %s", err.Error())
		}
	}

	log.Printf("Listening on %s\n", listener.Addr().String())
	log.Printf("Listening with ln address: %s \n", adr)

	go func() {
		for {
			netConn, err := listener.Accept() // this blocks
			if err != nil {
				log.Printf("Listener error: %s\n", err.Error())
				continue
			}
			newConn, ok := netConn.(*lndc.Conn)
			if !ok {
				log.Printf("Got something that wasn't a LNDC")
				continue
			}
			log.Printf("Incoming connection from %x on %s\n",
				newConn.RemotePub().SerializeCompressed(), newConn.RemoteAddr().String())

			// don't save host/port for incoming connections
			peerIdx, err := nd.GetPeerIdx(newConn.RemotePub(), "")
			if err != nil {
				log.Printf("Listener error: %s\n", err.Error())
				continue
			}

			nickname := nd.GetNicknameFromPeerIdx(peerIdx)

			nd.RemoteMtx.Lock()
			var peer RemotePeer
			peer.Idx = peerIdx
			peer.Con = newConn
			peer.Nickname = nickname
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

// ParseAdrString splits a string like
// "ln1yrvw48uc3atg8e2lzs43mh74m39vl785g4ehem@myhost.co:8191 into a separate
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
func (nd *LitNode) DialPeer(connectAdr string) error {
	var err error

	// parse address and get pkh / host / port
	who, where := splitAdrString(connectAdr)

	// If we couldn't deduce a URL, look it up on the tracker
	if where == "" {
		where, _, err = Lookup(who, nd.TrackerURL, nd.ProxyURL)
		if err != nil {
			return err
		}
	}

	// get my private ID key
	idPriv := nd.IdKey()

	// Assign remote connection
	newConn, err := lndc.Dial(idPriv, where, who, net.Dial)
	if err != nil {
		return err
	}

	// if connect is successful, either query for already existing peer index, or
	// if the peer is new, make a new index, and save the hostname&port

	// figure out peer index, or assign new one for new peer.  Since
	// we're connecting out, also specify the hostname&port
	peerIdx, err := nd.GetPeerIdx(newConn.RemotePub(), newConn.RemoteAddr().String())
	if err != nil {
		return err
	}

	// also retrieve their nickname, if they have one
	nickname := nd.GetNicknameFromPeerIdx(uint32(peerIdx))

	nd.RemoteMtx.Lock()
	var p RemotePeer
	p.Con = newConn
	p.Idx = peerIdx
	p.Nickname = nickname
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
		if !nd.ConnectedToPeer(msg.Peer()) {
			log.Printf("message type %x to peer %d but not connected\n",
				msg.MsgType(), msg.Peer())
			continue
		}

		//rawmsg := append([]byte{msg.MsgType()}, msg.Data...)
		rawmsg := msg.Bytes() // automatically includes messageType
		nd.RemoteMtx.Lock()   // not sure this is needed...
		n, err := nd.RemoteCons[msg.Peer()].Con.Write(rawmsg)
		if err != nil {
			log.Printf("error writing to peer %d: %s\n", msg.Peer(), err.Error())
		} else {
			log.Printf("type %x %d bytes to peer %d\n", msg.MsgType(), n, msg.Peer())
		}
		nd.RemoteMtx.Unlock()
	}
}

type PeerInfo struct {
	PeerNumber uint32
	RemoteHost string
	LitAdr 	   string
	Nickname   string
}

func (nd *LitNode) GetConnectedPeerList() []PeerInfo {
	var peers []PeerInfo
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
	_, ok := nd.RemoteCons[peer]
	nd.RemoteMtx.Unlock()
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
