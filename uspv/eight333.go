package uspv

import (
	"fmt"
	"log"
	"os"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/bloom"
	"github.com/mit-dci/lit/lnutil"
)

const (
	// keyFileName and headerFileName are not referred in this file? -- takaya
	keyFileName    = "testseed.hex"
	headerFileName = "headers.bin"

	// version hardcoded for now, probably ok...?
	// 70012 is for segnet... make this a init var?
	VERSION = 70012
)

// GimmeFilter ... or I'm gonna fade away
func (s *SPVCon) GimmeFilter() (*bloom.Filter, error) {

	filterElements := uint32(len(s.TrackingAdrs) + (len(s.TrackingOPs)))

	f := bloom.NewFilter(filterElements, 0, 0.000001, wire.BloomUpdateAll)

	// note there could be false positives since we're just looking
	// for the 20 byte PKH without the opcodes.
	for a160, _ := range s.TrackingAdrs { // add 20-byte pubkeyhash
		//		fmt.Printf("adding address hash %x\n", a160)
		f.Add(a160[:])
	}
	//	for _, u := range allUtxos {
	//		f.AddOutPoint(&u.Op)
	//	}

	// actually... we should monitor addresses, not txids, right?
	// or no...?
	for wop, _ := range s.TrackingOPs {
		// try just outpoints, not the txids as well
		f.AddOutPoint(&wop)
	}
	// still some problem with filter?  When they broadcast a close which doesn't
	// send any to us, sometimes we don't see it and think the channel is still open.
	// so not monitoring the channel outpoint properly?  here or in ingest()

	log.Printf("made %d element filter\n", filterElements)
	return f, nil
}

// MatchTx queries whether a tx mathches registered addresses and outpoints.
func (s *SPVCon) MatchTx(tx *wire.MsgTx) bool {
	gain := false
	txid := tx.TxHash()
	// start with optimism.  We may gain money.  Iterate through all output scripts.
	for i, out := range tx.TxOut {
		// create outpoint of what we're looking at
		op := wire.NewOutPoint(&txid, uint32(i))

		// 20 byte pubkey hash of this txout (if any)
		var adr20 [20]byte
		copy(adr20[:], lnutil.KeyHashFromPkScript(out.PkScript))
		// when we gain utxo, set as gain so we can return a match, but
		// also go through all gained utxos and register to track them

		//		fmt.Printf("got output key %x ", adr20)
		if s.TrackingAdrs[adr20] {
			gain = true
			s.TrackingOPs[*op] = true
		} else {
			//			fmt.Printf(" no match\n")
		}

		// this outpoint may confirm an outpoint we're watching.  Check that here.
		if s.TrackingOPs[*op] {
			// not quite "gain", more like confirm, but same idea.
			gain = true
		}

	}

	// No need to check for loss if we have a gain
	if gain {
		return true
	}

	// next pessimism.  Iterate through inputs, matching tracked outpoints
	for _, in := range tx.TxIn {
		if s.TrackingOPs[in.PreviousOutPoint] {
			return true
		}
	}

	return false
}

// OKTxid assigns a height to a txid.  This means that
// the txid exists at that height, with whatever assurance (for height 0
// it's no assurance at all)
func (s *SPVCon) OKTxid(txid *chainhash.Hash, height int32) error {
	if txid == nil {
		return fmt.Errorf("tried to add nil txid")
	}
	log.Printf("added %s to OKTxids at height %d\n", txid.String(), height)
	s.OKMutex.Lock()
	s.OKTxids[*txid] = height
	s.OKMutex.Unlock()
	return nil
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
	// actually we should do this AFTER sending all the txs...
	s.CurrentHeightChan <- hah.height

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
		log.Printf("got %d headers. Range:\n%s - %s\n",
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

	log.Printf("get headers message has %d header hashes, first one is %s\n",
		len(ghdr.BlockLocatorHashes), ghdr.BlockLocatorHashes[0].String())

	s.outMsgQueue <- ghdr
	return nil
}

// AskForMerkBlocks requests blocks from current to last
// right now this asks for 1 block per getData message.
// Maybe it's faster to ask for many in a each message?
func (s *SPVCon) AskForBlocks() error {
	var hdr wire.BlockHeader

	s.headerMutex.Lock() // lock just to check filesize
	stat, err := os.Stat(s.headerFile.Name())
	s.headerMutex.Unlock() // checked, unlock
	endPos := stat.Size()

	// move back 1 header length to read
	headerTip := int32(endPos/80) + (s.headerStartHeight - 1)

	log.Printf("blockTip to %d headerTip %d\n", s.syncHeight, headerTip)
	if s.syncHeight > headerTip {
		return fmt.Errorf("error- db longer than headers! shouldn't happen.")
	}
	if s.syncHeight == headerTip {
		// nothing to ask for; set wait state and return
		log.Printf("no blocks to request, entering wait state\n")
		log.Printf("%d bytes received\n", s.RBytes)
		s.inWaitState <- true

		// check if we can grab outputs
		// Do this on wallit level instead
		//		err = s.GrabAll()
		//		if err != nil {
		//			return err
		//		}
		// also advertise any unconfirmed txs here
		//		s.Rebroadcast()
		// ask for mempool each time...?  put something in to only ask the
		// first time we sync...?
		//		if !s.Ironman {
		//			s.AskForMempool()
		//		}
		return nil
	}

	log.Printf("will request blocks %d to %d\n", s.syncHeight+1, headerTip)
	reqHeight := s.syncHeight

	// loop through all heights where we want merkleblocks.
	for reqHeight < headerTip {
		reqHeight++ // we're requesting the next header

		// load header from file
		s.headerMutex.Lock() // seek to header we need
		_, err = s.headerFile.Seek(
			int64((reqHeight-s.headerStartHeight)*80), os.SEEK_SET)
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

		hah := NewRootAndHeight(hdr.BlockHash(), reqHeight)
		if reqHeight == headerTip { // if this is the last block, indicate finality
			hah.final = true
		}
		// waits here most of the time for the queue to empty out
		s.blockQueue <- hah // push height and mroot of requested block on queue
		s.outMsgQueue <- gdataMsg
	}
	return nil
}
