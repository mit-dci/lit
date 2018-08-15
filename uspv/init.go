package uspv

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/wire"
	"golang.org/x/net/proxy"
)

func IP4(ipAddress string) bool {
	parseIp := net.ParseIP(ipAddress)
	if parseIp.To4() == nil {
		return false
	}
	return true
}

func (s *SPVCon) parseRemoteNode(remoteNode string) (string, string, error) {
	colonCount := strings.Count(remoteNode, ":")
	var conMode string
	if colonCount == 0 {
		if IP4(remoteNode) || remoteNode == "localhost" { // need this to connect to locahost
			remoteNode = remoteNode + ":" + s.Param.DefaultPort
		}
		// only ipv4 clears this since ipv6 has colons
		conMode = "tcp4"
		return remoteNode, conMode, nil
	} else if colonCount == 1 && IP4(strings.Split(remoteNode, ":")[0]) {
		// custom port on ipv4
		return remoteNode, "tcp4", nil
	} else if colonCount >= 5 {
		// ipv6 without remote port
		// assume users don't give ports with ipv6 nodes
		if !strings.Contains(remoteNode, "[") && !strings.Contains(remoteNode, "]") {
			remoteNode = "[" + remoteNode + "]" + ":" + s.Param.DefaultPort
		}
		conMode = "tcp6"
		return remoteNode, conMode, nil
	} else {
		return "", "", fmt.Errorf("Invalid ip")
	}
}

// GetListOfNodes contacts all DNSSeeds for the coin specified and then contacts
// each one of them in order to receive a list of ips and then returns a combined
// list
func (s *SPVCon) GetListOfNodes() ([]string, error) {
	var listOfNodes []string // slice of IP addrs returned from the DNS seed
	log.Printf("Attempting to retrieve peers to connect to based on DNS Seed\n")

	for _, seed := range s.Param.DNSSeeds {
		temp, err := net.LookupHost(seed)
		// need this temp in order to capture the error from net.LookupHost
		// also need this to report the number of IPs we get from a seed
		if err != nil {
			log.Printf("Have difficulty trying to conenct to %s. Going to the next seed", seed)
			continue
		}
		listOfNodes = append(listOfNodes, temp...)
		log.Printf("Got %d IPs from DNS seed %s\n", len(temp), seed)
	}
	if len(listOfNodes) == 0 {
		return nil, fmt.Errorf("No peers found connected to DNS Seeds. Please provide a host to connect to.")
	}
	return listOfNodes, nil
}

// DialNode receives a list of node ips and then tries to connect to them one by one.
func (s *SPVCon) DialNode(listOfNodesParent []string) error {
	// now have some IPs, go through and try to connect to one.
	var err error
	var wg sync.WaitGroup
	// attempt sonnecting to only ot as many nodes specified by the user.
	var slice int
	slice = len(listOfNodesParent)
	if s.maxConnections < len(listOfNodesParent) {
		slice = s.maxConnections
	}
	listOfNodes := listOfNodesParent[:slice]
	queue := make(chan net.Conn, 1)
	wg.Add(len(listOfNodes))
	for i, ip := range listOfNodes[:slice] { // Maintaining 10 parallel connections should be enough?
		// try to connect to all nodes in this range
		var conString, conMode string
		// need to check whether conString is ipv4 or ipv6
		conString, conMode, err = s.parseRemoteNode(ip)
		log.Printf("Attempting connection to node at %s\n",
			conString)

		if s.ProxyURL != "" {
			log.Printf("Attempting to connect via proxy %s", s.ProxyURL)
			var d proxy.Dialer
			d, err = proxy.SOCKS5("tcp", s.ProxyURL, nil, proxy.Direct)
			if err != nil {
				return err
			}
			s.conns[0], err = d.Dial(conMode, conString)
		} else {
			d := net.Dialer{Timeout: time.Millisecond * 1000}
			// get only the fastest nodes, drop the other ones
			// disconnect nodes if they don't respond within 1 sec
			// put all ips into a go routine and collect them later
			go func(i int) {
				dummy, _ := d.Dial(conMode, conString)
				queue <- dummy
			}(i)
		}

		if err != nil {
			if i != len(listOfNodes)-1 {
				log.Println(err.Error())
				continue
			} else if i == len(listOfNodes)-1 {
				log.Println(err)
				// all nodes have been exhausted, we move on to the next one, if any.
				return fmt.Errorf(" Tried to connect to all available node Addresses. Failed")
			}
		}
	}
	go func() {
		for conn := range queue {
			if conn != nil {
				s.conns = append(s.conns, conn)
			}
			wg.Done()
		}
	}()
	wg.Wait()
	return nil
}

func (s *SPVCon) Handshake(peerIdx int) error {
	// assign version bits for local node
	s.localVersion = VERSION
	myMsgVer, err := wire.NewMsgVersionFromConn(s.conns[peerIdx], 0, 0)
	if err != nil {
		return err
	}
	err = myMsgVer.AddUserAgent("lit", "v0.1")
	if err != nil {
		return err
	}
	// must set this to enable SPV stuff
	myMsgVer.AddService(wire.SFNodeBloom)
	// set this to enable segWit
	myMsgVer.AddService(wire.SFNodeWitness)
	// this actually sends
	n, err := wire.WriteMessageWithEncodingN(
		s.conns[peerIdx], myMsgVer, s.localVersion,
		wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
	if err != nil {
		return err
	}
	s.WBytes += uint64(n)
	log.Printf("wrote %d byte version message to %s\n",
		n, s.conns[peerIdx].RemoteAddr().String())
	n, m, b, err := wire.ReadMessageWithEncodingN(
		s.conns[peerIdx], s.localVersion,
		wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
	if err != nil {
		return err
	}
	s.RBytes += uint64(n)
	log.Printf("got %d byte response %x\n command: %s\n", n, b, m.Command())

	mv, ok := m.(*wire.MsgVersion)
	if ok {
		log.Printf("connected to %s", mv.UserAgent)
	}

	if mv.ProtocolVersion < 70013 {
		//70014 -> core v0.13.1, so we should be fine
		return fmt.Errorf("Remote node version: %x too old, disconnecting.", mv.ProtocolVersion)
	}

	if !((strings.Contains(s.Param.Name, "lite") && strings.Contains(mv.UserAgent, "LitecoinCore")) || strings.Contains(mv.UserAgent, "Satoshi") || strings.Contains(mv.UserAgent, "btcd")) {
		// TODO: improve this filtering criterion
		return fmt.Errorf("Spam node. Returning!")
	}

	log.Printf("remote reports version %x (dec %d)\n",
		mv.ProtocolVersion, mv.ProtocolVersion)

	// set remote height
	s.remoteHeight = append(s.remoteHeight, mv.LastBlock)
	log.Printf("node reports %d as height of the last block", mv.LastBlock)
	// set remote version
	s.remoteVersion = append(s.remoteVersion, uint32(mv.ProtocolVersion))

	mva := wire.NewMsgVerAck()
	n, err = wire.WriteMessageWithEncodingN(
		s.conns[peerIdx], mva, s.localVersion,
		wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
	if err != nil {
		return err
	}
	s.WBytes += uint64(n)
	return nil
}

func ConnCheck(in []bool) bool {
	for _, val := range in {
		if val == true {
			return val
		}
	}
	return false
}

func (s *SPVCon) ConnectToMaxConns(listOfNodes []string) ([]string, error) {
	var empty []string
	var err error
	if !s.randomNodesOK { // conneect to user provided node
		err = s.DialNode(listOfNodes)
		return listOfNodes, err
	}
	for len(listOfNodes)-s.maxConnections > 0 && len(s.conns) < s.maxConnections {
		// make sure we get atleast one active connection from the DNS Seeds
		log.Printf("Active Conns: %d", len(s.conns))
		err = s.DialNode(listOfNodes)
		if err != nil {
			log.Println(err) // no need to take action on this error since this doesn't
			// affect what we do below
		}
		if len(s.conns) >= s.maxConnections {
			s.conns = s.conns[:s.maxConnections] // restrict number of maximum connections
			return listOfNodes, nil
		}
		listOfNodes = listOfNodes[s.maxConnections:]
	}
	return empty, fmt.Errorf("Couldn't connect to any node from the list of peers obtained, exiting!")
}

// Connect dials out and connects to full nodes. Calls GetListOfNodes to get the
// list of nodes if the user has specified a YupString. Else, moves on to dial
// the node to see if its up and establishes a connection followed by Handshake()
// which sends out wire messages, checks for version string to prevent spam, etc.
func (s *SPVCon) Connect(remoteNode string) error {
	var err error
	var listOfNodes []string
	var handshakeEstablished []bool
	var connEstablished []bool
	if lnutil.YupString(remoteNode) {
		s.randomNodesOK = true
		// if remoteNode is "yes" but no IP specified, use DNS seed
		listOfNodes, err = s.GetListOfNodes()
		if err != nil {
			log.Println(err)
			return err
			// automatically quit if there are no other hosts to connect to.
		}
	} else { // else connect to user-specified node
		listOfNodes = []string{remoteNode}
	}
	log.Println("LSIT OF NODES", listOfNodes)
	// Connect to maxConns nodes
	listOfNodes, err = s.ConnectToMaxConns(listOfNodes)
	if err != nil {
		log.Println(err)
		return err
	}
	// Now we have the connections, try handshakes
	// Some peers might have weird version numbers, etc, so we drop them
	// hence we collect maxConns from above since there are bound to be spam nodes
	// which would get dropped and reduce the number of active connections
	log.Printf("Trying to connect to %d node(s)", len(s.conns))
	for k := 0; k < len(s.conns); k++ {
		err := s.Handshake(k)
		if err != nil {
			s.conns = append(s.conns[:k], s.conns[(k+1):]...) // delete s.conns[k]
			// means we either have a spam node or didn't get a resonse. So we try again
			handshakeEstablished = append(handshakeEstablished, false)
			connEstablished = append(connEstablished, false)
			log.Printf("Handshake failed with node %d. Moving on. Error: %s", k, err.Error())
			if len(listOfNodes) == 1 { // when the list is empty, error out
				return fmt.Errorf("Couldn't establish connection with any remote node. Exiting.")
			}
			continue
		}
		handshakeEstablished = append(handshakeEstablished, true)
		connEstablished = append(connEstablished, true)
		// setup streams to receive and send wire messages
		// one for each connection
		s.inMsgQueue = make(chan wire.Message)
		go s.incomingMessageHandler(k)
		s.outMsgQueue = make(chan wire.Message)
		go s.outgoingMessageHandler(k)
	}
	if !ConnCheck(connEstablished) && !ConnCheck(handshakeEstablished) {
		// if no handhsake and connection established
		// this case happens when user provided node fails to connect
		return fmt.Errorf("Couldn't establish connection with any node. Exiting.")
	}
	//log.Println(handshakeEstablished, connEstablished)
	log.Println("Remote versions of connected nodes:", s.remoteVersion)

	if s.HardMode {
		s.blockQueue = make(chan HashAndHeight, 32) // queue depth 32 for hardmode.
	} else {
		// for SPV, concurrent in-flight merkleblocks makes us miss txs.
		// The BloomUpdateAll setting seems like it should prevent it, but it
		// doesn't; occasionally it misses transactions, seems like with low
		// block index.  Could be a bug somewhere.  1 at a time merkleblock
		// seems OK.
		s.blockQueue = make(chan HashAndHeight, 1) // queue depth 1 for spv
	}
	s.fPositives = make(chan int32, 4000) // a block full, approx
	s.inWaitState = make(chan bool, 1)
	go s.fPositiveHandler()

	// if s.HardMode { // what about for non-hard?  send filter?
	// 	Ignore filters now; switch to filters fed to SPVcon from TS
	// 		filt, err := s.TS.GimmeFilter()
	// 		if err != nil {
	// 			return err
	// 		}
	// 		s.localFilter = filt
	// 		//		s.Refilter(filt)
	// }

	return nil
}

/*
Truncated header files
Like a regular header but the first 80 bytes is mostly empty.
The very first 4 bytes (big endian) says what height the empty 80 bytes
replace.  The next header, starting at offset 80, needs to be valid.
*/
func (s *SPVCon) openHeaderFile(hfn string) error {
	_, err := os.Stat(hfn)
	if err != nil {
		if os.IsNotExist(err) {
			var b bytes.Buffer
			// if StartHeader is defined, start with hardcoded height
			if s.Param.StartHeight != 0 {
				hdr := s.Param.StartHeader
				_, err := b.Write(hdr[:])
				if err != nil {
					return err
				}
			} else {
				err = s.Param.GenesisBlock.Header.Serialize(&b)
				if err != nil {
					return err
				}
			}
			err = ioutil.WriteFile(hfn, b.Bytes(), 0600)
			if err != nil {
				return err
			}
			log.Printf("made genesis header %x\n", b.Bytes())
			log.Printf("made genesis hash %s\n", s.Param.GenesisHash.String())
			log.Printf("created hardcoded genesis header at %s\n", hfn)
		}
	}

	if s.Param.StartHeight != 0 {
		s.headerStartHeight = s.Param.StartHeight
	}

	s.headerFile, err = os.OpenFile(hfn, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	log.Printf("opened header file %s\n", s.headerFile.Name())
	return nil
}
