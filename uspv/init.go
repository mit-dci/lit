package uspv

import (
	"bytes"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"li.lan/tx/lit/lnutil"
	"li.lan/tx/lit/portxo"
)

// --- Uwallet interface ----

func (s *SPVCon) GetPriv(k portxo.KeyGen) *btcec.PrivateKey {
	return s.TS.PathPrivkey(k)
}

func (s *SPVCon) GetPub(k portxo.KeyGen) *btcec.PublicKey {
	return s.TS.PathPubkey(k)
}

func (s *SPVCon) Params() *chaincfg.Params {
	return s.Param
}

func (s *SPVCon) PushTx(tx *wire.MsgTx) error {
	return s.NewOutgoingTx(tx)
}

func (s *SPVCon) LetMeKnow() chan lnutil.OutPointEvent {
	return s.TS.OPEventChan
}

// OpenSPV starts a
func OpenSPV(remoteNode string, hfn, dbfn string,
	inTs *TxStore, hard bool, iron bool, p *chaincfg.Params) (SPVCon, error) {
	// create new SPVCon
	var s SPVCon
	s.HardMode = hard
	s.Ironman = iron
	s.Param = p
	// I should really merge SPVCon and TxStore, they're basically the same
	inTs.Param = p
	s.OKTxids = make(map[chainhash.Hash]int32)
	s.TS = inTs // copy pointer of txstore into spvcon

	// open header file
	err := s.openHeaderFile(hfn)
	if err != nil {
		return s, err
	}
	// open db file
	err = inTs.OpenDB(dbfn)
	if err != nil {
		return s, err
	}
	// load known txids into ram
	//	txids, err := inTs.GetAllTxids()
	//	if err != nil {
	//		return s, err
	//	}
	//	s.OKMutex.Lock()
	//	for _, txid := range txids {
	//		s.OKTxids[*txid] = 0
	//	}
	//	s.OKMutex.Unlock()

	// open TCP connection
	s.con, err = net.Dial("tcp", remoteNode)
	if err != nil {
		return s, err
	}
	// assign version bits for local node
	s.localVersion = VERSION
	myMsgVer, err := wire.NewMsgVersionFromConn(s.con, 0, 0)
	if err != nil {
		return s, err
	}
	err = myMsgVer.AddUserAgent("test", "zero")
	if err != nil {
		return s, err
	}
	// must set this to enable SPV stuff
	myMsgVer.AddService(wire.SFNodeBloom)
	// set this to enable segWit
	myMsgVer.AddService(wire.SFNodeWitness)
	// this actually sends
	n, err := wire.WriteMessageWithEncodingN(s.con, myMsgVer, s.localVersion, s.Param.Net, wire.LatestEncoding)
	if err != nil {
		return s, err
	}
	s.WBytes += uint64(n)
	log.Printf("wrote %d byte version message to %s\n",
		n, s.con.RemoteAddr().String())
	n, m, b, err := wire.ReadMessageWithEncodingN(s.con, s.localVersion, s.Param.Net, wire.LatestEncoding)
	if err != nil {
		return s, err
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
	n, err = wire.WriteMessageWithEncodingN(s.con, mva, s.localVersion, s.Param.Net, wire.LatestEncoding)
	if err != nil {
		return s, err
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

	if hard { // what about for non-hard?  send filter?
		filt, err := s.TS.GimmeFilter()
		if err != nil {
			return s, err
		}
		s.localFilter = filt
		//		s.Refilter(filt)
	}

	return s, nil
}

func (s *SPVCon) openHeaderFile(hfn string) error {
	_, err := os.Stat(hfn)
	if err != nil {
		if os.IsNotExist(err) {
			var b bytes.Buffer
			err = s.TS.Param.GenesisBlock.Header.Serialize(&b)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(hfn, b.Bytes(), 0600)
			if err != nil {
				return err
			}
			log.Printf("created hardcoded genesis header at %s\n",
				hfn)
		}
	}
	s.headerFile, err = os.OpenFile(hfn, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	log.Printf("opened header file %s\n", s.headerFile.Name())
	return nil
}
