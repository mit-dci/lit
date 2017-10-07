package uspv

import (
	"encoding/json"
	"github.com/adiabat/btcd/wire"
	"github.com/adiabat/btcutil/bloom"
	"github.com/mit-dci/lit/lnutil"
	"log"
	"os"
	"time"
)

func (s *SPVCon) incomingMessageHandler() {
	for {
		n, xm, _, err := wire.ReadMessageWithEncodingN(s.con, s.localVersion,
			wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
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
		case *wire.MsgGetAddr: // what case is this?
			log.Printf("Got getaddr. Do nothing since we aren't a full node.")
			// read this info and store somewhere.
		case *wire.MsgAddr:
			log.Printf("got %d addresses.\n", len(m.AddrList))
			s.AddrListHandler(m)
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

		n, err := wire.WriteMessageWithEncodingN(s.con, msg, s.localVersion,
			wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)

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
			filt, err := s.GimmeFilter()
			if err != nil {
				log.Printf("Filter creation error: %s\n", err.Error())
				log.Printf("uhoh, crashing filter handler")
				return
			}
			// send filter
			s.Refilter(filt)
			log.Printf("sent filter %x\n", filt.MsgFilterLoad().Filter)

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

			log.Printf("reset %d false positives\n", fpAccumulator)
			// reset accumulator
			fpAccumulator = 0
		}
	}
}

// REORG TODO: how to detect reorgs and send them up to wallet layer

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
		filt, err := s.GimmeFilter()
		if err != nil {
			log.Printf("AskForBlocks error: %s", err.Error())
			return
		}
		// send filter
		s.SendFilter(filt)
		log.Printf("sent filter %x\n", filt.MsgFilterLoad().Filter)
	}

	err = s.AskForBlocks()
	if err != nil {
		log.Printf("AskForBlocks error: %s", err.Error())
		return
	}
}

// TxHandler takes in transaction messages that come in from either a request
// after an inv message or after a merkle block message.
func (s *SPVCon) TxHandler(tx *wire.MsgTx) {
	log.Printf("received msgtx %s\n", tx.TxHash().String())
	// check if we have a height for this tx.
	s.OKMutex.Lock()
	height, ok := s.OKTxids[tx.TxHash()]
	s.OKMutex.Unlock()
	// if we don't have a height for this / it's not in the map, discard.
	// currently CRASHES when this happens because I want to see if it ever does.
	// it shouldn't if things are working properly.
	if !ok {
		log.Printf("Tx %s unknown, will not ingest\n", tx.TxHash().String())
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
	//			log.Printf("dub %d known tx %s and new tx %s are exclusive!!!\n",
	//				i, dub.String(), m.TxSha().String())
	//		}
	//	}

	// send txs up to wallit
	if s.MatchTx(tx) {
		s.TxUpToWallit <- lnutil.TxAndHeight{tx, height}
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

		// I think we do the same thing for witTx or tx...
		// I don't think they'll request non-witness anyway.
		if thing.Type == wire.InvTypeWitnessTx || thing.Type == wire.InvTypeTx {
			tx, ok := s.TxMap[thing.Hash]
			if !ok || tx == nil {
				log.Printf("tx %s requested by we don't have it\n",
					thing.Hash.String())
			}
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
				log.Printf("asking for headers due to inv block\n")
				err := s.AskForHeaders()
				if err != nil {
					log.Printf("AskForHeaders error: %s", err.Error())
				}
			default:
				// drop it as if its component particles had high thermal energies
				log.Printf("inv block but ignoring; not synced\n")
			}
		}
	}
}

func (s *SPVCon) AddrListHandler(m *wire.MsgAddr) {
	if _, errNotExists := os.Stat("./peers/peers.json"); os.IsNotExist(errNotExists) {
		os.Mkdir("./peers", 0700)
		crfile, errcreate := os.Create("./peers/peers.json")
		crfile.Close()
		if errcreate != nil {
			log.Println("File creation error. Exiting")
			return
		}
	}
	readvalues := getNodes()
	log.Println("Its coming in here")
	// log.Println(readvalues)

	flag := 0
	var ips [100]string
	var ports [100]int
	var ve [150]wire.NetAddress
	for i, vals := range readvalues {
		// do what we want with the IPs, which is to try connecting to them.
		ve[i] = vals
		ips[i] = vals.IP.String()
		ports[i] = int(vals.Port)
	}
	// og.Println(ve) // ve has all the addressses read from the file. We just have to create a new file if needed.
	log.Printf("Start IP List")
	var a wire.NetAddress // limit to 125 like bitcoind?
	// refer https://github.com/btcsuite/btcd/blob/master/wire/netaddress.go for *wire.NetAddr struct
	for i, addresses := range m.AddrList {
		log.Printf("IP: %s", addresses.IP)
		now := time.Now()
		if addresses.Timestamp.After(now.Add(time.Minute * 10)) {
			addresses.Timestamp = now.Add(-1 * time.Hour * 24 * 5)
		}
		log.Printf("Timestamp: %s", addresses.Timestamp)
		a.IP = addresses.IP
		a.Port = addresses.Port
		for _, addr := range ips { // for a list of all IPs in the file
			if addr == "<nil>" { // if its nil, stop execution
				flag = 0
				break
			}
			if addr == a.IP.String() { // is the address is already present in the file, ignore
				for _, port := range ports { // if we are connecting to the same host with a different port, we should still save it
					if port == int(a.Port) {
						flag = 1
						break
					}
				}
				if flag == 1 {
					break
				}
			}
		}
		// log.Println("The elusive flag")
		// log.Println(flag)
		if flag != 1 {
			//log.Println("This should get executed in case we can't load the ips from the file")
			a.Timestamp = addresses.Timestamp
			a.Services = addresses.Services

			// attach a to ve
			ve[99+i] = a
			for j, values := range ve {

				dest, err := json.Marshal(values)
				if err != nil {
					log.Println("Converting to a JSON object failed")
				}
				if j == 0 {
					errdel := os.Remove("./peers/peers.json")
					if errdel != nil {
						log.Println("File deletion error. Exiting")
						break
					}
					crfile, errcreate := os.Create("./peers/peers.json")
					if errcreate != nil {
						log.Println("File creation error. Exiting")
						break
					}
					crfile.Close()
				}
				file, err2 := os.OpenFile("./peers/peers.json", os.O_APPEND|os.O_WRONLY, 0644)
				if err2 != nil {
					log.Println(err2)
				}
				// defer file.Close()

				if j == 0 {
					_, rnd := file.WriteString("[")
					if rnd != nil {
						log.Println("Saving starting character failed. Quitting")
						break
					}
				}

				if values.IP.String() != "<nil>" {
					_, err1 := file.WriteString(string(dest))

					if err1 != nil {
						log.Println("Appending to a file failed")
					}

					if j < 99+i {
						_, err3 := file.WriteString(",\n")
						if err3 != nil {
							log.Println("Failed to write Newline")
						}
					}
					if j == 99+i {
						_, rnd1 := file.WriteString("\n]")
						if rnd1 != nil {
							log.Println("Saving ending characters failed. Quitting")
							break
						}
					}
				}
			}
			// log.Println(string(json.Marshal(a[0].IP)))
		} else {
			break
		}
	}
	log.Printf("end ip list")
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
