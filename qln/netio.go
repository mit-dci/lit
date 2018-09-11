package qln

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	nat "github.com/mit-dci/lit/nat"
	shortadr "github.com/mit-dci/lit/shortadr"
)

// Gets the list of ports where LitNode is listening for incoming connections,
// & the connection key
func (nd *LitNode) GetLisPorts() []string {
	nd.RemoteMtx.Lock()
	ports := nd.LisIpPorts
	nd.RemoteMtx.Unlock()
	return ports
}

func (nd *LitNode) FindPeerIndexByAddress(lnAdr string) (uint32, error) {
	nd.RemoteMtx.Lock()
	defer nd.RemoteMtx.Unlock()
	for idx, peer := range nd.RemoteCons {
		var pubKey [33]byte
		copy(pubKey[:], peer.Con.RemotePub().SerializeCompressed())
		adr := lnutil.LitAdrFromPubkey(pubKey)
		if adr == lnAdr {
			return idx, nil
		}
	}

	return 0, fmt.Errorf("Node %s not found", lnAdr)
}

// TCPListener starts a litNode listening for incoming LNDC connections
func (nd *LitNode) TCPListener(lisIpPort string,
	shortArg bool, shortZeros uint8) (string, error) {
	idPriv := nd.IdKey()

	// do UPnP / pmp port forwarding
	// fatal if we aren't able to port forward via upnp
	if len(nd.Nat) > 0 {
		listenPort, err := strconv.Atoi(lisIpPort[1:])
		if err != nil {
			logging.Error("Invalid port number, returning")
			return "", err
		}
		if nd.Nat == "upnp" {
			logging.Info("Port forwarding via UPnP on port", lisIpPort[1:])
			err := nat.SetupUpnp(uint16(listenPort))
			if err != nil {
				fmt.Printf("Unable to setup Upnp %v\n", err)
				logging.Fatal(err) // error out if we can't connect via UPnP
			}
			logging.Info("Forwarded port via UPnP")
		} else if nd.Nat == "pmp" {
			discoveryTimeout := time.Duration(10 * time.Second)
			logging.Info("Port forwarding via NAT Pmp on port", lisIpPort[1:])
			_, err := nat.SetupPmp(discoveryTimeout, uint16(listenPort))
			if err != nil {
				err := fmt.Errorf("Unable to discover a "+
					"NAT-PMP enabled device on the local "+
					"network: %v", err)
				logging.Fatal(err) // error out if we can't connect via Pmp
			} else {
				logging.Error("Invalid NAT punching option")
			}
		}
	}

	var idPub [33]byte
	copy(idPub[:], idPriv.PubKey().SerializeCompressed())

	stop := make([]chan bool, nd.MaxThreads) // open 10 channels, maybe receive this from the user as well?
	MaxUint64 := uint64(1<<64 - 1)           // nonce range to iterate over
	NoOfWorkers := uint64(nd.MaxThreads)     // 10 for testing, get from user, max 1 million
	Start := MaxUint64 / NoOfWorkers         // split each routine to look at a specific range

	logging.Infof("Spinning up %d threads for generatign short address", nd.MaxThreads)
	// used to assign ids to the channels
	i := uint64(0)
	// default nonce value, if zero, we proceed with the normal listening address case
	bestNonce := uint64(0)
	// the firstHash with higher difficulty than specified powbytes
	var bestHash [20]byte
	// target powbytes that we must aim at
	powbytes := 8 * shortZeros

	if shortArg {
		shortAddressReply := make(chan shortadr.ShortReply)
		for ; i < NoOfWorkers; i++ {
			go shortadr.ShortAdrWorker(idPub, i, Start*i, powbytes, shortAddressReply, stop[i])
		}
		for reply := range shortAddressReply {
			if shortadr.CheckWork(reply.BestHash)/8 >= shortZeros {
				bestNonce = reply.BestNonce
				bestHash = reply.BestHash
				break // break since we got a nocne with lower difficulty
			}
		}
		go func() {
			for ; i < NoOfWorkers; i++ {
				stop[i] <- true // TODO: this still runs the chans till one nonce each is found.
			}
		}()
	}
	listener, err := lndc.NewListener(nd.IdKey(), lisIpPort, bestNonce)
	if err != nil {
		return "", err
	}
	//pub, id, nonce uint64, bestBits uint8, bestNonce chan uint64, stop chan bool
	//adr := lnutil.LitAdrFromPubkey(idPub)
	var adr string
	if bestNonce != 0 {
		adr = lnutil.LitShortAdrFromPubkey(bestHash[:]) // shorter adr if nonce is non zero
	} else {
		adr = lnutil.LitAdrFromPubkey(idPub) // default scenario
	}

	// Don't announce on the tracker if we are communicating via SOCKS proxy
	if nd.ProxyURL == "" {
		// this should happen asynchronously
		go GoAnnounce(idPriv, lisIpPort, adr, nd.TrackerURL)
	}

	logging.Infof("Listening on %s\n", listener.Addr().String())
	logging.Infof("Listening with ln address: %s and nonce: %d\n", adr, bestNonce)
	go func() {
		for {
			netConn, err := listener.Accept() // this blocks
			if err != nil {
				logging.Errorf("Listener error: %s\n", err.Error())
				continue
			}
			newConn, ok := netConn.(*lndc.Conn)
			if !ok {
				logging.Errorf("Got something that wasn't a LNDC")
				continue
			}
			logging.Infof("Incoming connection from %x on %s\n",
				newConn.RemotePub().SerializeCompressed(), newConn.RemoteAddr().String())

			// don't save host/port for incoming connections
			peerIdx, err := nd.GetPeerIdx(newConn.RemotePub(), "")
			if err != nil {
				logging.Errorf("Listener error: %s\n", err.Error())
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

func GoAnnounce(priv *btcec.PrivateKey, litport string, litadr string, trackerURL string) {
	err := lnutil.Announce(priv, litport, litadr, trackerURL)
	if err != nil {
		logging.Errorf("Announcement error %s", err.Error())
	}
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
func (nd *LitNode) DialPeer(connectAdr string) (uint32, error) {
	var err error

	// parse address and get pkh / host / port
	who, where := lnutil.ParseAdrString(connectAdr)

	// If we couldn't deduce a URL, look it up on the tracker
	if where == "" {
		where, _, err = lnutil.Lookup(who, nd.TrackerURL, nd.ProxyURL)
		if err != nil {
			return 0, err
		}
	}

	// get my private ID key
	idPriv := nd.IdKey()

	// Assign remote connection
	newConn, nonce, err := lndc.Dial(idPriv, where, who, net.Dial)
	if err != nil {
		return 0, err
	}

	// if connect is successful, either query for already existing peer index, or
	// if the peer is new, make a new index, and save the hostname&port

	// figure out peer index, or assign new one for new peer.  Since
	// we're connecting out, also specify the hostname&port
	peerIdx, err := nd.GetPeerIdx(newConn.RemotePub(), newConn.RemoteAddr().String())
	if err != nil {
		return 0, err
	}

	// also retrieve their nickname, if they have one
	nickname := nd.GetNicknameFromPeerIdx(uint32(peerIdx))

	nd.RemoteMtx.Lock()
	var p RemotePeer
	p.Con = newConn
	p.Idx = peerIdx
	p.Nonce = nonce
	p.Nickname = nickname
	nd.RemoteCons[peerIdx] = &p
	nd.RemoteMtx.Unlock()

	// each connection to a peer gets its own LNDCReader
	go nd.LNDCReader(&p)

	return peerIdx, nil
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
		if v.Nonce != 0 {
			// Nonce is non zero, means we're connecting to a short address
			hash := shortadr.HashOnce(pubArr, 0, v.Nonce) // get the hash with zeros
			newPeer.LitAdr = lnutil.LitShortAdrFromPubkey(hash[:])
		}
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
