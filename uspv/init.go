package uspv

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

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
	log.Println(listOfNodes)
	return listOfNodes, nil
}

// DialNode receives a list of node ips and then tries to connect to them one by one.
func (s *SPVCon) DialNode(listOfNodes []string) error {
	// now have some IPs, go through and try to connect to one.
	var err error
	for i, ip := range listOfNodes {
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

			s.con, err = d.Dial(conMode, conString)
		} else {
			s.con, err = net.Dial(conMode, conString)
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
		break
	}
	return nil
}

func (s *SPVCon) Handshake(listOfNodes []string) error {
	// assign version bits for local node
	s.localVersion = VERSION
	myMsgVer, err := wire.NewMsgVersionFromConn(s.con, 0, 0)
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
		s.con, myMsgVer, s.localVersion,
		wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
	if err != nil {
		return err
	}
	s.WBytes += uint64(n)
	log.Printf("wrote %d byte version message to %s\n",
		n, s.con.RemoteAddr().String())
	n, m, b, err := wire.ReadMessageWithEncodingN(
		s.con, s.localVersion,
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

	if !((strings.Contains(s.Param.Name, "lite") && strings.Contains(mv.UserAgent, "LitecoinCore")) || strings.Contains(mv.UserAgent, "Satoshi") || strings.Contains(mv.UserAgent, "btcd")) && (len(listOfNodes) != 0) {
		// TODO: improve this filtering criterion
		return fmt.Errorf("Couldn't connect to this node. Returning!")
	}

	log.Printf("remote reports version %x (dec %d)\n",
		mv.ProtocolVersion, mv.ProtocolVersion)

	// set remote height
	s.remoteHeight = mv.LastBlock
	// set remote version
	s.remoteVersion = uint32(mv.ProtocolVersion)

	mva := wire.NewMsgVerAck()
	n, err = wire.WriteMessageWithEncodingN(
		s.con, mva, s.localVersion,
		wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
	if err != nil {
		return err
	}
	s.WBytes += uint64(n)
	return nil
}

// Connect dials out and connects to full nodes. Calls GetListOfNodes to get the
// list of nodes if the user has specified a YupString. Else, moves on to dial
// the node to see if its up and establishes a connection followed by Handshake()
// which sends out wire messages, checks for version string to prevent spam, etc.
func (s *SPVCon) Connect(remoteNode string) error {
	var err error
	var listOfNodes []string
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
	handShakeFailed := false //need to be in this scope to access it here
	connEstablished := false
	for len(listOfNodes) != 0 {
		err = s.DialNode(listOfNodes)
		if err != nil {
			log.Println(err)
			log.Printf("Couldn't dial node %s, Moving on", listOfNodes[0])
			listOfNodes = listOfNodes[1:]
			continue
		}
		err = s.Handshake(listOfNodes)
		if err != nil {
			handShakeFailed = true
			log.Printf("Handshake with %s failed. Moving on. Error: %s", listOfNodes[0], err.Error())
			if len(listOfNodes) == 1 { // when the list is empty, error out
				return fmt.Errorf("Couldn't establish connection with any remote node. Exiting.")
			}
			// means we either have a sapm node or didn't get a resonse. So we Try again
			log.Println(err)
			log.Println("Couldn't establish connection with node. Proceeding to the next one")
			listOfNodes = listOfNodes[1:]
			connEstablished = false
		} else {
			connEstablished = true
		}
		if connEstablished { // connection should be established, still checking for safety
			break
		} else {
			continue
		}
	}

	if !handShakeFailed && !connEstablished {
		// this case happens when user provided node fails to connect
		return fmt.Errorf("Couldn't establish connection with node. Exiting.")
	}
	if handShakeFailed && !connEstablished {
		// this case is when the last node fails and we continue, only to exit the
		// loop and execute below code, which is unnecessary.
		return fmt.Errorf("Couldn't establish connection with any remote node after an instance of handshake. Exiting.")
	}
	s.inMsgQueue = make(chan wire.Message)
	go s.incomingMessageHandler()
	s.outMsgQueue = make(chan wire.Message)
	go s.outgoingMessageHandler()

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
