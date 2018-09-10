package uspv

import (
	. "github.com/mit-dci/lit/logs"

	"github.com/mit-dci/lit/btcutil/bloom"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/wire"
)

func (s *SPVCon) incomingMessageHandler() {
	for {
		n, xm, _, err := wire.ReadMessageWithEncodingN(s.con, s.localVersion,
			wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
		if err != nil {
			s.con.Close() // close the connection to prevent spam messages from crashing lit.
			Log.Infof("ReadMessageWithEncodingN error.  Disconnecting from given peer. %s\n", err.Error())
			if s.randomNodesOK { // if user wants to connect to localhost, let him do so
				s.Connect("yes") // really any YupString here
			} else {
				s.con.Close()
				return
			}
		}
		s.RBytes += uint64(n)
		//		Log.Infof("Got %d byte %s message\n", n, xm.Command())
		switch m := xm.(type) {
		case *wire.MsgVersion:
			Log.Infof("Got version message.  Agent %s, version %d, at height %d\n",
				m.UserAgent, m.ProtocolVersion, m.LastBlock)
			s.remoteVersion = uint32(m.ProtocolVersion) // weird cast! bug?
		case *wire.MsgVerAck:
			Log.Infof("Got verack.  Whatever.\n")
		case *wire.MsgAddr:
			Log.Infof("got %d addresses.\n", len(m.AddrList))
		case *wire.MsgPing:
			// Log.Infof("Got a ping message.  We should pong back or they will kick us off.")
			go s.PongBack(m.Nonce)
		case *wire.MsgPong:
			Log.Infof("Got a pong response. OK.\n")
		case *wire.MsgBlock:
			s.IngestBlock(m)
		case *wire.MsgMerkleBlock:
			s.IngestMerkleBlock(m)
		case *wire.MsgHeaders: // concurrent because we keep asking for blocks
			go s.HeaderHandler(m)
		case *wire.MsgTx: // not concurrent! txs must be in order
			s.TxHandler(m)
		case *wire.MsgReject:
			Log.Infof("Rejected! cmd: %s code: %s tx: %s reason: %s",
				m.Cmd, m.Code.String(), m.Hash.String(), m.Reason)
		case *wire.MsgInv:
			s.InvHandler(m)
		case *wire.MsgNotFound:
			Log.Infof("Got not found response from remote:")
			for i, thing := range m.InvList {
				Log.Infof("\t%d) %s: %s", i, thing.Type, thing.Hash)
			}
		case *wire.MsgGetData:
			s.GetDataHandler(m)

		default:
			if m != nil {
				Log.Infof("Got unknown message type %s\n", m.Command())
			} else {
				Log.Errorf("Got nil message")
			}
		}
	}
}

// this one seems kindof pointless?  could get ridf of it and let
// functions call WriteMessageWithEncodingN themselves...
func (s *SPVCon) outgoingMessageHandler() {
	for {
		msg := <-s.outMsgQueue
		if msg == nil {
			Log.Errorf("ERROR: nil message to outgoingMessageHandler\n")
			continue
		}
		n, err := wire.WriteMessageWithEncodingN(s.con, msg, s.localVersion,
			wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)

		if err != nil {
			Log.Errorf("Write message error: %s", err.Error())
		}
		s.WBytes += uint64(n)
	}
}

// fPositiveHandler monitors false positives and when it gets enough of them,
func (s *SPVCon) fPositiveHandler() {
	var fpAccumulator int32
	for {
		fpAccumulator += <-s.fPositives // blocks here
		if fpAccumulator > 7 {
			filt, err := s.GimmeFilter()
			if err != nil {
				Log.Errorf("Filter creation error: %s\n", err.Error())
				Log.Errorf("uhoh, crashing filter handler")
				return
			}
			// send filter
			s.Refilter(filt)
			Log.Infof("sent filter %x\n", filt.MsgFilterLoad().Filter)

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

			Log.Infof("reset %d false positives\n", fpAccumulator)
			// reset accumulator
			fpAccumulator = 0
		}
	}
}

// REORG TODO: how to detect reorgs and send them up to wallet layer

// HeaderHandler ...
func (s *SPVCon) HeaderHandler(m *wire.MsgHeaders) {
	moar, err := s.IngestHeaders(m)
	if err != nil {
		Log.Errorf("Header error: %s\n", err.Error())
		return
	}
	// more to get? if so, ask for them and return
	if moar {
		err = s.AskForHeaders()
		if err != nil {
			Log.Errorf("AskForHeaders error: %s", err.Error())
		}
		return
	}
	// no moar, done w/ headers, send filter and get blocks
	if !s.HardMode { // don't send this in hardmode! that's the whole point
		filt, err2 := s.GimmeFilter()
		if err2 != nil {
			Log.Errorf("AskForBlocks error: %s", err2.Error())
			return
		}
		// send filter
		s.SendFilter(filt)
		Log.Infof("sent filter %x\n", filt.MsgFilterLoad().Filter)
	}

	err = s.AskForBlocks()
	if err != nil {
		Log.Errorf("AskForBlocks error: %s", err.Error())
		return
	}
}

// TxHandler takes in transaction messages that come in from either a request
// after an inv message or after a merkle block message.
func (s *SPVCon) TxHandler(tx *wire.MsgTx) {
	Log.Infof("received msgtx %s\n", tx.TxHash().String())
	// check if we have a height for this tx.
	s.OKMutex.Lock()
	height, ok := s.OKTxids[tx.TxHash()]
	s.OKMutex.Unlock()
	// if we don't have a height for this / it's not in the map, discard.
	// currently CRASHES when this happens because I want to see if it ever does.
	// it shouldn't if things are working properly.
	if !ok {
		Log.Errorf("Tx %s unknown, will not ingest\n", tx.TxHash().String())
		panic("unknown tx")
	}

	// check for double spends ...?
	//	allTxs, err := s.TS.GetAllTxs()
	//	if err != nil {
	//		Log.Errorf("Can't get txs from db: %s", err.Error())
	//		return
	//	}
	//	dubs, err := CheckDoubleSpends(m, allTxs)
	//	if err != nil {
	//		Log.Errorf("CheckDoubleSpends error: %s", err.Error())
	//		return
	//	}
	//	if len(dubs) > 0 {
	//		for i, dub := range dubs {
	//			Log.Infof("dub %d known tx %s and new tx %s are exclusive!!!\n",
	//				i, dub.String(), m.TxSha().String())
	//		}
	//	}

	// send txs up to wallit
	if s.MatchTx(tx) {
		s.TxUpToWallit <- lnutil.TxAndHeight{Tx: tx, Height: height}
	}
}

// GetDataHandler responds to requests for tx data, which happen after
// advertising our txs via an inv message
func (s *SPVCon) GetDataHandler(m *wire.MsgGetData) {
	Log.Infof("got GetData.  Contains:\n")
	var sent int32
	for i, thing := range m.InvList {
		Log.Infof("\t%d)%s : %s",
			i, thing.Type.String(), thing.Hash.String())

		// I think we do the same thing for witTx or tx...
		// I don't think they'll request non-witness anyway.
		if thing.Type == wire.InvTypeWitnessTx || thing.Type == wire.InvTypeTx {
			tx, ok := s.TxMap[thing.Hash]
			if !ok || tx == nil {
				Log.Infof("tx %s requested but we don't have it\n",
					thing.Hash.String())
				continue
			}
			s.outMsgQueue <- tx
			sent++
			continue
		}
		// didn't match, so it's not something we're responding to
		Log.Infof("We only respond to tx requests, ignoring")
	}
	Log.Infof("sent %d of %d requested items", sent, len(m.InvList))
}

// InvHandler ...
func (s *SPVCon) InvHandler(m *wire.MsgInv) {
	Log.Infof("got inv.  Contains:\n")
	for i, thing := range m.InvList {
		Log.Infof("\t%d)%s : %s",
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
				Log.Infof("asking for headers due to inv block\n")
				err := s.AskForHeaders()
				if err != nil {
					Log.Errorf("AskForHeaders error: %s", err.Error())
				}
			default:
				// drop it as if its component particles had high thermal energies
				Log.Infof("inv block but ignoring; not synced\n")
			}
		}
	}
}

// PongBack ...
func (s *SPVCon) PongBack(nonce uint64) {
	mpong := wire.NewMsgPong(nonce)

	s.outMsgQueue <- mpong
	return
}

// SendFilter ...
func (s *SPVCon) SendFilter(f *bloom.Filter) {
	s.outMsgQueue <- f.MsgFilterLoad()

	return
}
