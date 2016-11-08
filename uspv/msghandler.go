package uspv

import (
	"fmt"
	"log"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/bloom"
)

func (s *SPVCon) incomingMessageHandler() {
	for {
		n, xm, _, err := wire.ReadMessageWithEncodingN(
			s.con, s.localVersion, s.Param.Net, wire.LatestEncoding)
		if err != nil {
			log.Printf("ReadMessageWithEncodingN error.  Disconnecting: %s\n", err.Error())
			return
		}
		s.RBytes += uint64(n)
		//		log.Printf("Got %d byte %s message\n", n, xm.Command())
		switch m := xm.(type) {
		case *wire.MsgVersion:
			log.Printf("Got version message.  Agent %s, version %d, at height %d\n",
				m.UserAgent, m.ProtocolVersion, m.LastBlock)
			s.remoteVersion = uint32(m.ProtocolVersion) // weird cast! bug?
		case *wire.MsgVerAck:
			log.Printf("Got verack.  Whatever.\n")
		case *wire.MsgAddr:
			log.Printf("got %d addresses.\n", len(m.AddrList))
		case *wire.MsgPing:
			// log.Printf("Got a ping message.  We should pong back or they will kick us off.")
			go s.PongBack(m.Nonce)
		case *wire.MsgPong:
			log.Printf("Got a pong response. OK.\n")
		case *wire.MsgBlock:
			s.IngestBlock(m)
		case *wire.MsgMerkleBlock:
			s.IngestMerkleBlock(m)
		case *wire.MsgHeaders: // concurrent because we keep asking for blocks
			go s.HeaderHandler(m)
		case *wire.MsgTx: // not concurrent! txs must be in order
			s.TxHandler(m)
		case *wire.MsgReject:
			log.Printf("Rejected! cmd: %s code: %s tx: %s reason: %s",
				m.Cmd, m.Code.String(), m.Hash.String(), m.Reason)
		case *wire.MsgInv:
			s.InvHandler(m)
		case *wire.MsgNotFound:
			log.Printf("Got not found response from remote:")
			for i, thing := range m.InvList {
				log.Printf("\t$d) %s: %s", i, thing.Type, thing.Hash)
			}
		case *wire.MsgGetData:
			s.GetDataHandler(m)

		default:
			log.Printf("Got unknown message type %s\n", m.Command())
		}
	}
	return
}

// this one seems kindof pointless?  could get ridf of it and let
// functions call WriteMessageWithEncodingN themselves...
func (s *SPVCon) outgoingMessageHandler() {
	for {
		msg := <-s.outMsgQueue
		n, err := wire.WriteMessageWithEncodingN(s.con, msg, s.localVersion, s.Param.Net, wire.LatestEncoding)
		if err != nil {
			log.Printf("Write message error: %s", err.Error())
		}
		s.WBytes += uint64(n)
	}
	return
}

// fPositiveHandler monitors false positives and when it gets enough of them,
func (s *SPVCon) fPositiveHandler() {
	var fpAccumulator int32
	for {
		fpAccumulator += <-s.fPositives // blocks here
		if fpAccumulator > 7 {
			filt, err := s.TS.GimmeFilter()
			if err != nil {
				log.Printf("Filter creation error: %s\n", err.Error())
				log.Printf("uhoh, crashing filter handler")
				return
			}
			// send filter
			s.Refilter(filt)
			fmt.Printf("sent filter %x\n", filt.MsgFilterLoad().Filter)

			// clear the channel
		finClear:
			for {
				select {
				case x := <-s.fPositives:
					fpAccumulator += x
				default:
					break finClear
				}
			}

			fmt.Printf("reset %d false positives\n", fpAccumulator)
			// reset accumulator
			fpAccumulator = 0
		}
	}
}

func (s *SPVCon) HeaderHandler(m *wire.MsgHeaders) {
	moar, err := s.IngestHeaders(m)
	if err != nil {
		log.Printf("Header error: %s\n", err.Error())
		return
	}
	// more to get? if so, ask for them and return
	if moar {
		err = s.AskForHeaders()
		if err != nil {
			log.Printf("AskForHeaders error: %s", err.Error())
		}
		return
	}
	// no moar, done w/ headers, send filter and get blocks
	if !s.HardMode { // don't send this in hardmode! that's the whole point
		filt, err := s.TS.GimmeFilter()
		if err != nil {
			log.Printf("AskForBlocks error: %s", err.Error())
			return
		}
		// send filter
		s.SendFilter(filt)
		fmt.Printf("sent filter %x\n", filt.MsgFilterLoad().Filter)
	}
	dbTip, err := s.TS.GetDBSyncHeight()
	if err != nil {
		log.Printf("AskForBlocks error: %s", err.Error())
		return
	}
	err = s.AskForBlocks(dbTip)
	if err != nil {
		log.Printf("AskForBlocks error: %s", err.Error())
		return
	}
}

// TxHandler takes in transaction messages that come in from either a request
// after an inv message or after a merkle block message.
func (s *SPVCon) TxHandler(m *wire.MsgTx) {
	log.Printf("received msgtx %s\n", m.TxHash().String())
	// check if we have a height for this tx.
	s.OKMutex.Lock()
	height, ok := s.OKTxids[m.TxHash()]
	s.OKMutex.Unlock()
	// if we don't have a height for this / it's not in the map, discard.
	// currently CRASHES when this happens because I want to see if it ever does.
	// it shouldn't if things are working properly.
	if !ok {
		log.Printf("Tx %s unknown, will not ingest\n", m.TxHash().String())
		panic("unknown tx")
		return
	}

	// check for double spends ...?
	//	allTxs, err := s.TS.GetAllTxs()
	//	if err != nil {
	//		log.Printf("Can't get txs from db: %s", err.Error())
	//		return
	//	}
	//	dubs, err := CheckDoubleSpends(m, allTxs)
	//	if err != nil {
	//		log.Printf("CheckDoubleSpends error: %s", err.Error())
	//		return
	//	}
	//	if len(dubs) > 0 {
	//		for i, dub := range dubs {
	//			fmt.Printf("dub %d known tx %s and new tx %s are exclusive!!!\n",
	//				i, dub.String(), m.TxSha().String())
	//		}
	//	}

	utilTx := btcutil.NewTx(m)
	if !s.HardMode || s.localFilter.MatchTxAndUpdate(utilTx) {
		hits, err := s.TS.Ingest(m, height)
		if err != nil {
			log.Printf("Incoming Tx error: %s\n", err.Error())
			return
		}

		if hits == 0 {
			log.Printf("tx %s had no hits, filter false positive.",
				m.TxHash().String())
			s.fPositives <- 1 // add one false positive to chan
		} else {
			// there was a hit, send new filter I guess.  Feels like
			// you shouldn't have to..?
			// make new filter
			filt, err := s.TS.GimmeFilter()
			if err != nil {
				log.Printf("Incoming Tx error: %s\n", err.Error())
				return
			}
			// send filter
			s.Refilter(filt)
		}
		log.Printf("tx %s ingested and matches %d utxo/adrs.",
			m.TxHash().String(), hits)
	}
	if utilTx.Hash().String() == "c5cbac2c838a2ca75d5294afe13726813669cd81c75f87e25db232e42e70ee7e" {
		//		panic("c5cbac2c")
		log.Printf("ZZO got c5cbac2c spends 9f5e344d84a70101007944bcd109f6544df4df1cdca506c23ae94b45ce0b98f2")
	}

}

// GetDataHandler responds to requests for tx data, which happen after
// advertising our txs via an inv message
func (s *SPVCon) GetDataHandler(m *wire.MsgGetData) {
	log.Printf("got GetData.  Contains:\n")
	var sent int32
	for i, thing := range m.InvList {
		log.Printf("\t%d)%s : %s",
			i, thing.Type.String(), thing.Hash.String())

		// separate wittx and tx.  needed / combine?
		// does the same thing right now
		if thing.Type == wire.InvTypeWitnessTx {
			tx, err := s.TS.GetTx(&thing.Hash)
			if err != nil {
				log.Printf("error getting tx %s: %s",
					thing.Hash.String(), err.Error())
				continue
			}
			s.outMsgQueue <- tx
			sent++
			continue
		}
		if thing.Type == wire.InvTypeTx {
			tx, err := s.TS.GetTx(&thing.Hash)
			if err != nil {
				log.Printf("error getting tx %s: %s",
					thing.Hash.String(), err.Error())
				continue
			}
			// Shouldn't connect to non-witness aware nodes anyway so
			// probably remove this and just return an error?
			// Right now returns a witness tx even if asked for regular tx.
			//			tx.DeWitnessify()
			//			tx.Flags = 0x00 // dewitnessify
			s.outMsgQueue <- tx
			sent++
			continue
		}
		// didn't match, so it's not something we're responding to
		log.Printf("We only respond to tx requests, ignoring")
	}
	log.Printf("sent %d of %d requested items", sent, len(m.InvList))
}

func (s *SPVCon) InvHandler(m *wire.MsgInv) {
	log.Printf("got inv.  Contains:\n")
	for i, thing := range m.InvList {
		log.Printf("\t%d)%s : %s",
			i, thing.Type.String(), thing.Hash.String())
		if thing.Type == wire.InvTypeTx {
			// ignore tx invs in ironman mode, or if we already have it
			if !s.Ironman {
				// new tx, OK it at 0 and request
				// also request if we already have it; might have new witness?
				// needed for confirmed channels...
				s.OKTxid(&thing.Hash, 0) // unconfirmed
				s.AskForTx(thing.Hash)
			}
		}
		if thing.Type == wire.InvTypeBlock { // new block what to do?
			select {
			case <-s.inWaitState:
				// start getting headers
				fmt.Printf("asking for headers due to inv block\n")
				err := s.AskForHeaders()
				if err != nil {
					log.Printf("AskForHeaders error: %s", err.Error())
				}
			default:
				// drop it as if its component particles had high thermal energies
				fmt.Printf("inv block but ignoring; not synched\n")
			}
		}
	}
}

func (s *SPVCon) PongBack(nonce uint64) {
	mpong := wire.NewMsgPong(nonce)

	s.outMsgQueue <- mpong
	return
}

func (s *SPVCon) SendFilter(f *bloom.Filter) {
	s.outMsgQueue <- f.MsgFilterLoad()

	return
}
