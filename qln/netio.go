package qln

import (
	"fmt"
	."github.com/mit-dci/lit/logs"
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
			Log.Error("Invalid port number, returning")
			return "", err
		}
		if nd.Nat == "upnp" {
			Log.Debug("Port forwarding via UPnP on port", lisIpPort[1:])
			err := nat.SetupUpnp(uint16(listenPort))
			if err != nil {
				fmt.Printf("Unable to setup Upnp %v\n", err)
				Log.Error(err) // error out if we can't connect via UPnP
			}
			Log.Debug("Forwarded port via UPnP")
		} else if nd.Nat == "pmp" {
			discoveryTimeout := time.Duration(10 * time.Second)
			Log.Debug("Port forwarding via NAT Pmp on port", lisIpPort[1:])
			_, err := nat.SetupPmp(discoveryTimeout, uint16(listenPort))
			if err != nil {
				err := fmt.Errorf("Unable to discover a "+
					"NAT-PMP enabled device on the local "+
					"network: %v", err)
				Log.Fatal(err) // error out if we can't connect via Pmp
			} else {
				Log.Error("Invalid NAT punching option")
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
			Log.Errorf("Announcement error %s", err.Error())
		}
	}

	Log.Infof("Listening on %s\n", listener.Addr().String())
	Log.Infof("Listening with ln address: %s \n", adr)

	go func() {
		for {
			netConn, err := listener.Accept() // this blocks
			if err != nil {
				Log.Errorf("Listener error: %s\n", err.Error())
				continue
			}
			newConn, ok := netConn.(*lndc.LNDConn)
			if !ok {
				Log.Error("Got something that wasn't a LNDC")
				continue
			}
			Log.Infof("Incoming connection from %x on %s\n",
				newConn.RemotePub.SerializeCompressed(), newConn.RemoteAddr().String())

			// don't save host/port for incoming connections
			peerIdx, err := nd.GetPeerIdx(newConn.RemotePub, "")
			if err != nil {
				Log.Errorf("Listener error: %s\n", err.Error())
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

// DialPeer makes an outgoing connection to another node.
func (nd *LitNode) DialPeer(connectAdr string) error {
	var err error

	// parse address and get pkh / host / port
	who, where := lndc.SplitAdrString(connectAdr)

	// sanity check the "who" pkh string
	if !lnutil.LitAdrOK(who) {
		return fmt.Errorf("ln address %s invalid", who)
	}

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
	newConn := new(lndc.LNDConn)

	// TODO: handle IPv6 connections
	err = newConn.Dial(idPriv, where, who, nd.ProxyURL)
	if err != nil {
		return err
	}

	// if connect is successful, either query for already existing peer index, or
	// if the peer is new, make a new index, and save the hostname&port

	// figure out peer index, or assign new one for new peer.  Since
	// we're connecting out, also specify the hostname&port
	peerIdx, err := nd.GetPeerIdx(newConn.RemotePub, newConn.RemoteAddr().String())
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
			Log.Infof("message type %x to peer %d but not connected\n",
				msg.MsgType(), msg.Peer())
			continue
		}

		//rawmsg := append([]byte{msg.MsgType()}, msg.Data...)
		rawmsg := msg.Bytes() // automatically includes messageType
		nd.RemoteMtx.Lock()   // not sure this is needed...
		n, err := nd.RemoteCons[msg.Peer()].Con.Write(rawmsg)
		if err != nil {
			Log.Errorf("error writing to peer %d: %s\n", msg.Peer(), err.Error())
		} else {
			Log.Infof("type %x %d bytes to peer %d\n", msg.MsgType(), n, msg.Peer())
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
		copy(pubArr[:], v.Con.RemotePub.SerializeCompressed())
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
