package uspv

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/btcsuite/btcd/wire"
)

// Connect dials out and connects to full nodes.
func (s *SPVCon) Connect(remoteNode string) error {
	var err error
	// open TCP connection
	s.con, err = net.Dial("tcp", remoteNode)
	if err != nil {
		return err
	}
	// assign version bits for local node
	s.localVersion = VERSION
	myMsgVer, err := wire.NewMsgVersionFromConn(s.con, 0, 0)
	if err != nil {
		return err
	}
	err = myMsgVer.AddUserAgent("test", "zero")
	if err != nil {
		return err
	}
	// must set this to enable SPV stuff
	myMsgVer.AddService(wire.SFNodeBloom)
	// set this to enable segWit
	myMsgVer.AddService(wire.SFNodeWitness)
	// this actually sends
	n, err := wire.WriteMessageWithEncodingN(
		s.con, myMsgVer, s.localVersion, s.Param.Net, wire.LatestEncoding)
	if err != nil {
		return err
	}
	s.WBytes += uint64(n)
	log.Printf("wrote %d byte version message to %s\n",
		n, s.con.RemoteAddr().String())
	n, m, b, err := wire.ReadMessageWithEncodingN(
		s.con, s.localVersion, s.Param.Net, wire.LatestEncoding)
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
		s.con, mva, s.localVersion, s.Param.Net, wire.LatestEncoding)
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
//

func (s *SPVCon) openHeaderFile(hfn string) error {
	_, err := os.Stat(hfn)
	if err != nil {
		if os.IsNotExist(err) {
			var b bytes.Buffer
			// if testnet, start with hardcoded height
			if s.Param.Name == "testnet3" {
				// hard-coded millionth block header (actually 1032192)
				hdr, err := hex.DecodeString("00000020da33925b1f7a55e9fa8e6c955a20ea094148b60c5c88f69a4f500000000000003673b7b6ce8157d3cfcaf415b6740918df7610a8769d70334aa9abd9c941b25e7621215880ba371a85bf9646")
				if err != nil {
					return err
				}
				_, err = b.Write(hdr)
				if err != nil {
					return err
				}
			} else {
				// not testnet3, start from beginning.
				err = s.Param.GenesisBlock.Header.Serialize(&b)
				if err != nil {
					return err
				}
			}
			err = ioutil.WriteFile(hfn, b.Bytes(), 0600)
			if err != nil {
				return err
			}
			log.Printf("made genesis block %x\n", b.Bytes())
			log.Printf("made genesis header %s\n", s.Param.GenesisHash.String())
			log.Printf("created hardcoded genesis header at %s\n", hfn)
		}
	}
	if s.Param.Name == "testnet3" {
		s.headerStartHeight = 1032192
	}

	s.headerFile, err = os.OpenFile(hfn, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	log.Printf("opened header file %s\n", s.headerFile.Name())
	return nil
}
