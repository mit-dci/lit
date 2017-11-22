package uspv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/adiabat/btcd/wire"
)

func (s *SPVCon) CreatePeerFile() error {
	if _, err := os.Stat(s.nodeFile); os.IsNotExist(err) {
		os.Mkdir("./peers", 0700)
		file, err := os.Create(s.nodeFile)
		file.Close()
		if err != nil {
			log.Println("File creation error. Exiting")
			return err
		}
		return nil
	}
	return nil
}

func (s *SPVCon) GetNodes() ([]wire.NetAddress, error) {
	var addresses []wire.NetAddress
	raw, err := ioutil.ReadFile(s.nodeFile)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(raw, &addresses)
	return addresses, nil
}

func (s *SPVCon) Connect(remoteNode string) error {
	var err error
	flag := 0
	log.Println("Its coming over here")
	log.Println(remoteNode)
	// slice of IP addrs returned from the DNS seed
	var seedAdrs []string
	if remoteNode == "" || remoteNode == "auto" || remoteNode == "1" ||
		remoteNode == "true" || remoteNode == "y" {
		// connect to DNS seed based node
		log.Printf("Attempting connection based on DNS seed\n")
		if len(s.Param.DNSSeeds) == 0 {
			// no available DNS seeds
			return fmt.Errorf(
				"Can't connect: No known DNS seeds for %s", s.Param.Name)
		}

		// before this part, I want to see if I can connect to stuff over in the peers file

		if _, err := os.Stat(s.nodeFile); os.IsNotExist(err) {
			log.Println("Peers file doesn't exist") // set flag to 1 sicne peers doesn't exist
		} else {
			log.Println("Reading from peers file")
			readvalues, err := s.GetNodes()
			if err != nil {
				log.Println(err)
				return err
			}
			for _, ve := range readvalues {
				// do what we want with the IPs, which is to try connecting to them.
				if ve.IP.String() != "<nil>" {
					log.Printf("Attempting to connect to %s:%d", ve.IP.String(), int(ve.Port))
					if strconv.Itoa(int(ve.Port)) == s.Param.DefaultPort { // to handle different protocols
						_, err := net.LookupHost(ve.IP.String())
						if err != nil {
							log.Println("Fatal Error while connecting to remote node. Trying again.")
							continue
						}
						remoteNode = "[" + ve.IP.String() + "]" + ":"+ s.Param.DefaultPort
						s.con, err = net.Dial("tcp", remoteNode)
						if err != nil {
							log.Println(err.Error())
							continue
						}
						flag = 1
						break
					}
				}
			}
		}
		// end the part where I connect to the stuff over at peers.json

		if flag != 1 {
			// go through seeds to get a list of IP addrs
			for _, seed := range s.Param.DNSSeeds {
				seedAdrs, err = net.LookupHost(seed)
				if err == nil {
					// got em
					log.Printf("Got %d IPs from DNS seed %s\n", len(seedAdrs), seed)
					break
				}
				// gotta keep trying for those IPs
				log.Println("DNS seed %s error", seed)
			}

			if len(seedAdrs) == 0 {
				// never got any IPs from DNS seeds; give up
				return fmt.Errorf(
					"Can't connect: No functioning DNS seeds for %s", s.Param.Name)
			}
			// now have some IPs, go through and try to connect to one.
			var connected bool
			for _, ip := range seedAdrs {
				log.Printf("Attempting connection to node at %s\n",
					ip+":"+s.Param.DefaultPort)
				s.con, err = net.Dial("tcp", ip+":"+s.Param.DefaultPort)
				if err != nil {
					log.Println(err.Error())
				} else {
					connected = true
					break
				}
			}
			if !connected {
				return fmt.Errorf("Tried all IPs from DNS seed, none worked")
			}
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
	}

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
	log.Printf("remote reports version %x (dec %d)\n",
		mv.ProtocolVersion, mv.ProtocolVersion)

	// set remote height
	s.remoteHeight = mv.LastBlock
	mva := wire.NewMsgVerAck()
	n, err = wire.WriteMessageWithEncodingN(
		s.con, mva, s.localVersion,
		wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
	if err != nil {
		return err
	}
	s.WBytes += uint64(n)

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

	if s.HardMode { // what about for non-hard?  send filter?
		/*
			Ignore filters now; switch to filters fed to SPVcon from TS
				filt, err := s.TS.GimmeFilter()
				if err != nil {
					return err
				}
				s.localFilter = filt
				//		s.Refilter(filt)
		*/
	}

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
