package uspv

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/bloom"
)

const (
	// keyFileName and headerFileName are not referred in this file? -- takaya
	keyFileName    = "testseed.hex"
	headerFileName = "headers.bin"

	// version hardcoded for now, probably ok...?
	// 70012 is for segnet... make this a init var?
	VERSION = 70012
)

type SPVCon struct {
	con net.Conn // the (probably tcp) connection to the node

	// Enhanced SPV modes for users who have outgrown easy mode SPV
	// but have not yet graduated to full nodes.
	HardMode bool // hard mode doesn't use filters.
	Ironman  bool // ironman only gets blocks, never requests txs.

	headerMutex       sync.Mutex
	headerFile        *os.File // file for SPV headers
	headerStartHeight int32    // first header on disk is nth header in chain

	OKTxids map[chainhash.Hash]int32 // known good txids and their heights
	OKMutex sync.Mutex

	localFilter *bloom.Filter // local bloom filter for hard mode

	//[doesn't work without fancy mutexes, nevermind, just use header file]
	// localHeight   int32  // block height we're on
	remoteHeight  int32  // block height they're on
	localVersion  uint32 // version we report
	remoteVersion uint32 // version remote node

	// what's the point of the input queue? remove? leave for now...
	inMsgQueue  chan wire.Message // Messages coming in from remote node
	outMsgQueue chan wire.Message // Messages going out to remote node

	WBytes uint64 // total bytes written
	RBytes uint64 // total bytes read

	Param *chaincfg.Params // network parameters (testnet3, segnet, etc)
	TS    *TxStore         // transaction store to write to

	// RawBlockSender is a channel to send full blocks up to the qln / watchtower
	// only kicks in when requested from upper layer
	RawBlockSender chan *wire.MsgBlock

	// mBlockQueue is for keeping track of what height we've requested.
	blockQueue chan HashAndHeight
	// fPositives is a channel to keep track of bloom filter false positives.
	fPositives chan int32

	// waitState is a channel that is empty while in the header and block
	// sync modes, but when in the idle state has a "true" in it.
	inWaitState chan bool
}

// AskForTx requests a tx we heard about from an inv message.
// It's one at a time but should be fast enough.
// I don't like this function because SPV shouldn't even ask...
func (s *SPVCon) AskForTx(txid chainhash.Hash) {
	gdata := wire.NewMsgGetData()
	inv := wire.NewInvVect(wire.InvTypeTx, &txid)
	// no longer get wit txs if in hardmode... don't need to, right?
	//	if s.HardMode {
	//		inv.Type = wire.InvTypeWitnessTx
	//	}
	gdata.AddInvVect(inv)
	log.Printf("asking for tx %s\n", txid.String())
	s.outMsgQueue <- gdata
}

// HashAndHeight is needed instead of just height in case a fullnode
// responds abnormally (?) by sending out of order merkleblocks.
// we cache a merkleroot:height pair in the queue so we don't have to
// look them up from the disk.
// Also used when inv messages indicate blocks so we can add the header
// and parse the txs in one request instead of requesting headers first.
type HashAndHeight struct {
	blockhash chainhash.Hash
	height    int32
	final     bool // indicates this is the last merkleblock requested
}

// NewRootAndHeight saves like 2 lines.
func NewRootAndHeight(b chainhash.Hash, h int32) (hah HashAndHeight) {
	hah.blockhash = b
	hah.height = h
	return
}

func (s *SPVCon) RemoveHeaders(r int32) error {
	endPos, err := s.headerFile.Seek(0, os.SEEK_END)
	if err != nil {
		return err
	}
	err = s.headerFile.Truncate(endPos - int64(r*80))
	if err != nil {
		return fmt.Errorf("couldn't truncate header file")
	}
	return nil
}

func (s *SPVCon) IngestMerkleBlock(m *wire.MsgMerkleBlock) {

	txids, err := checkMBlock(m) // check self-consistency
	if err != nil {
		log.Printf("Merkle block error: %s\n", err.Error())
		return
	}
	var hah HashAndHeight
	select { // select here so we don't block on an unrequested mblock
	case hah = <-s.blockQueue: // pop height off mblock queue
		break
	default:
		log.Printf("Unrequested merkle block")
		return
	}

	// this verifies order, and also that the returned header fits
	// into our SPV header file
	newMerkBlockSha := m.Header.BlockHash()
	if !hah.blockhash.IsEqual(&newMerkBlockSha) {
		log.Printf("merkle block out of order got %s expect %s",
			m.Header.BlockHash().String(), hah.blockhash.String())
		log.Printf("has %d hashes %d txs flags: %x",
			len(m.Hashes), m.Transactions, m.Flags)
		return
	}

	for _, txid := range txids {
		err := s.OKTxid(txid, hah.height)
		if err != nil {
			log.Printf("Txid store error: %s\n", err.Error())
			return
		}
	}
	// write to db that we've sync'd to the height indicated in the
	// merkle block.  This isn't QUITE true since we haven't actually gotten
	// the txs yet but if there are problems with the txs we should backtrack.
	err = s.TS.SetDBSyncHeight(hah.height)
	if err != nil {
		log.Printf("Merkle block error: %s\n", err.Error())
		return
	}
	if hah.final {
		// don't set waitstate; instead, ask for headers again!
		// this way the only thing that triggers waitstate is asking for headers,
		// getting 0, calling AskForMerkBlocks(), and seeing you don't need any.
		// that way you are pretty sure you're synced up.
		err = s.AskForHeaders()
		if err != nil {
			log.Printf("Merkle block error: %s\n", err.Error())
			return
		}
	}
	return
}

// IngestHeaders takes in a bunch of headers and appends them to the
// local header file, checking that they fit.  If there's no headers,
// it assumes we're done and returns false.  If it worked it assumes there's
// more to request and returns true.
func (s *SPVCon) IngestHeaders(m *wire.MsgHeaders) (bool, error) {
	gotNum := int64(len(m.Headers))
	if gotNum > 0 {
		fmt.Printf("got %d headers. Range:\n%s - %s\n",
			gotNum, m.Headers[0].BlockHash().String(),
			m.Headers[len(m.Headers)-1].BlockHash().String())
	} else {
		log.Printf("got 0 headers, we're probably synced up")
		return false, nil
	}

	s.headerMutex.Lock()
	defer s.headerMutex.Unlock()

	var err error
	// seek to last header
	_, err = s.headerFile.Seek(-80, os.SEEK_END)
	if err != nil {
		return false, err
	}
	var last wire.BlockHeader
	err = last.Deserialize(s.headerFile)
	if err != nil {
		return false, err
	}
	prevHash := last.BlockHash()

	endPos, err := s.headerFile.Seek(0, os.SEEK_END)
	if err != nil {
		return false, err
	}
	tip := int32(endPos/80) + (s.headerStartHeight - 1) // move back 1 header length to read

	// check first header returned to make sure it fits on the end
	// of our header file
	if !m.Headers[0].PrevBlock.IsEqual(&prevHash) {
		// delete 100 headers if this happens!  Dumb reorg.
		log.Printf("reorg? header msg doesn't fit. points to %s, expect %s",
			m.Headers[0].PrevBlock.String(), prevHash.String())
		if endPos < 8160 {
			// jeez I give up, back to genesis
			s.headerFile.Truncate(160)
		} else {
			err = s.headerFile.Truncate(endPos - 8000)
			if err != nil {
				return false, fmt.Errorf("couldn't truncate header file")
			}
		}
		return true, fmt.Errorf("Truncated header file to try again")
	}

	for _, resphdr := range m.Headers {
		// write to end of file
		err = resphdr.Serialize(s.headerFile)
		if err != nil {
			return false, err
		}
		// advance chain tip
		tip++
		// check last header
		worked := CheckHeader(s.headerFile, tip, s.headerStartHeight, s.Param)
		if !worked {
			if endPos < 8160 {
				// jeez I give up, back to genesis
				s.headerFile.Truncate(160)
			} else {
				err = s.headerFile.Truncate(endPos - 8000)
				if err != nil {
					return false, fmt.Errorf("couldn't truncate header file")
				}
			}
			// probably should disconnect from spv node at this point,
			// since they're giving us invalid headers.
			return true, fmt.Errorf(
				"Header %d - %s doesn't fit, dropping 100 headers.",
				tip, resphdr.BlockHash().String())
		}
	}
	log.Printf("Headers to height %d OK.", tip)
	return true, nil
}

func (s *SPVCon) AskForHeaders() error {
	var hdr wire.BlockHeader
	ghdr := wire.NewMsgGetHeaders()
	ghdr.ProtocolVersion = s.localVersion

	s.headerMutex.Lock() // start header file ops
	info, err := s.headerFile.Stat()
	if err != nil {
		return err
	}
	headerFileSize := info.Size()
	if headerFileSize == 0 || headerFileSize%80 != 0 { // header file broken
		// try to fix it!
		s.headerFile.Truncate(headerFileSize - (headerFileSize % 80))
		log.Printf("ERROR: Header file not a multiple of 80 bytes. Truncating")
	}

	// seek to 80 bytes from end of file
	ns, err := s.headerFile.Seek(-80, os.SEEK_END)
	if err != nil {
		log.Printf("can't seek\n")
		return err
	}

	log.Printf("sought to offset %d (should be near the end\n", ns)

	// get header from last 80 bytes of file
	err = hdr.Deserialize(s.headerFile)
	if err != nil {
		log.Printf("can't Deserialize")
		return err
	}
	s.headerMutex.Unlock() // done with header file

	cHash := hdr.BlockHash()
	err = ghdr.AddBlockLocatorHash(&cHash)
	if err != nil {
		return err
	}

	fmt.Printf("get headers message has %d header hashes, first one is %s\n",
		len(ghdr.BlockLocatorHashes), ghdr.BlockLocatorHashes[0].String())

	s.outMsgQueue <- ghdr
	return nil
}

// Ask for their mempool.  2 line func. 1 of which is "return"
func (s *SPVCon) AskForMempool() {
	s.outMsgQueue <- wire.NewMsgMemPool()
	return
}

// AskForOneBlock is for testing only, so you can ask for a specific block height
// and see what goes wrong
/* not used
func (s *SPVCon) AskForOneBlock(h int32) error {
	var hdr wire.BlockHeader
	var err error

	dbTip := int32(h)
	s.headerMutex.Lock() // seek to header we need
	_, err = s.headerFile.Seek(int64((dbTip)*80), os.SEEK_SET)
	if err != nil {
		return err
	}
	err = hdr.Deserialize(s.headerFile) // read header, done w/ file for now
	s.headerMutex.Unlock()              // unlock after reading 1 header
	if err != nil {
		log.Printf("header deserialize error!\n")
		return err
	}

	bHash := hdr.BlockSha()
	// create inventory we're asking for
	iv1 := wire.NewInvVect(wire.InvTypeWitnessBlock, &bHash)
	gdataMsg := wire.NewMsgGetData()
	// add inventory
	err = gdataMsg.AddInvVect(iv1)
	if err != nil {
		return err
	}
	hah := NewRootAndHeight(bHash, h)
	s.outMsgQueue <- gdataMsg
	s.blockQueue <- hah // push height and mroot of requested block on queue
	return nil
}
*/

// AskForMerkBlocks requests blocks from current to last
// right now this asks for 1 block per getData message.
// Maybe it's faster to ask for many in a each message?
func (s *SPVCon) AskForBlocks(dbTip int32) error {
	var hdr wire.BlockHeader

	s.headerMutex.Lock() // lock just to check filesize
	stat, err := os.Stat(s.headerFile.Name())
	s.headerMutex.Unlock() // checked, unlock
	endPos := stat.Size()

	// move back 1 header length to read
	headerTip := int32(endPos/80) + (s.headerStartHeight - 1)

	fmt.Printf("dbTip %d headerTip %d\n", dbTip, headerTip)
	if dbTip > headerTip {
		return fmt.Errorf("error- db longer than headers! shouldn't happen.")
	}
	if dbTip == headerTip {
		// nothing to ask for; set wait state and return
		fmt.Printf("no blocks to request, entering wait state\n")
		fmt.Printf("%d bytes received\n", s.RBytes)
		s.inWaitState <- true
		// check if we can grab outputs
		err = s.GrabAll()
		if err != nil {
			return err
		}
		// also advertise any unconfirmed txs here
		s.Rebroadcast()
		// ask for mempool each time...?  put something in to only ask the
		// first time we sync...?
		//		if !s.Ironman {
		//			s.AskForMempool()
		//		}
		return nil
	}

	fmt.Printf("will request blocks %d to %d\n", dbTip+1, headerTip)

	// loop through all heights where we want merkleblocks.
	for dbTip < headerTip {
		dbTip++ // we're requesting the next header

		// load header from file
		s.headerMutex.Lock() // seek to header we need
		_, err = s.headerFile.Seek(int64((dbTip-s.headerStartHeight)*80), os.SEEK_SET)
		if err != nil {
			return err
		}
		err = hdr.Deserialize(s.headerFile) // read header, done w/ file for now
		s.headerMutex.Unlock()              // unlock after reading 1 header
		if err != nil {
			log.Printf("header deserialize error!\n")
			return err
		}

		bHash := hdr.BlockHash()
		// create inventory we're asking for
		iv1 := new(wire.InvVect)
		// if hardmode, ask for legit blocks, none of this ralphy stuff
		// I don't think you can have a queue for SPV.  You miss stuff.
		if s.HardMode {
			iv1 = wire.NewInvVect(wire.InvTypeWitnessBlock, &bHash)
		} else { // ah well
			iv1 = wire.NewInvVect(wire.InvTypeFilteredBlock, &bHash)
		}
		gdataMsg := wire.NewMsgGetData()
		// add inventory
		err = gdataMsg.AddInvVect(iv1)
		if err != nil {
			return err
		}

		hah := NewRootAndHeight(hdr.BlockHash(), dbTip)
		if dbTip == headerTip { // if this is the last block, indicate finality
			hah.final = true
		}
		// waits here most of the time for the queue to empty out
		s.blockQueue <- hah // push height and mroot of requested block on queue
		s.outMsgQueue <- gdataMsg
	}
	return nil
}
