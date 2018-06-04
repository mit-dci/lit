package uspv

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"github.com/adiabat/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
)

func (s *SPVCon) GetSeedAdrs(seed string) ([]string, error) {
	var err error
	var seedAdrs []string // slice of IP addrs returned from the DNS seed
	log.Printf("Attempting to retrieve peers to connect to based on DNS Seed\n")

	seedAdrs, err = net.LookupHost(seed)
	if err == nil {
		log.Printf("Got %d IPs from DNS seed %s\n", len(seedAdrs), seed)
		// got seedAdrs, but it may be empty
	} else {
		return nil, fmt.Errorf("Have difficulty trying to conenct to %s. Going to the next seed", seed)
	}

	if len(seedAdrs) == 0 {
		// check whether seedAdrs is not empty
		return nil, fmt.Errorf("No peers found conencted to %s. Continuing.", seed)
	}
	return seedAdrs, nil
	// return the seedAdrs here
}

func (s *SPVCon) DialNode(seedAdrs []string) error {
	// now have some IPs, go through and try to connect to one.
	var err error
	for i, ip := range seedAdrs {
		// try to connect to all nodes in this range
		log.Printf("Attempting connection to node at %s\n",
			ip+":"+s.Param.DefaultPort)
		s.con, err = net.Dial("tcp", ip+":"+s.Param.DefaultPort)
		if err != nil && i != len(seedAdrs)-1 {
			log.Println(err.Error())
			continue
		} else if i == len(seedAdrs)-1 {
			// all nodes have been exhausted, we move on to the next one, if any.
			return fmt.Errorf(" Tried to connect to all seed Addresses from peer. Failed")
		}
		break
	}
	return nil
}

func (s *SPVCon) Handshake(seedAdrs []string) error {
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

	if !(strings.Contains(mv.UserAgent, "Satoshi") || strings.Contains(mv.UserAgent, "btcd")) && (len(seedAdrs) != 0) {
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

// Connect dials out and connects to full nodes.
func (s *SPVCon) Connect(remoteNode string) error {
	var err error
	if lnutil.YupString(remoteNode) {
		// if remoteNode is "yes" but no IP specified, use DNS seed
		if len(s.Param.DNSSeeds) == 0 {
			// no available DNS seeds
			return fmt.Errorf(
				"Can't connect: No known DNS seeds for %s", s.Param.Name)
		}
		listofDNSSeeds := s.Param.DNSSeeds
		for len(listofDNSSeeds) != 0 {
			seed := listofDNSSeeds[0]
			seedAdrs, err := s.GetSeedAdrs(seed)
			if err != nil {
				return err
			}
			err = s.DialNode(seedAdrs)
			if err != nil {
				listofDNSSeeds = listofDNSSeeds[1:]
				continue
			}
			err = s.Handshake(seedAdrs)
			if err != nil {
				// means we either have a sapm node or didn't get a resonse. So we Try again
				log.Println(err)
				log.Println("Couldn't establish connection with node. Proceeding to the next one")
				continue
			}
			break
		}
	} else { // else connect to user-specified node
		if !strings.Contains(remoteNode, ":") {
			remoteNode = remoteNode + ":" + s.Param.DefaultPort
		}
		// open TCP connection to specified host
		s.con, err = net.Dial("tcp", remoteNode)
		if err != nil {
			return err
		}
		err := s.Handshake(nil)
		if err != nil {
			return err
		}
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
