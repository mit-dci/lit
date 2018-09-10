package uspv

import (
	"fmt"
	"log"
	"os"

	"github.com/mit-dci/lit/btcutil/bloom"
	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/wire"
)

const (
	// keyFileName and headerFileName are not referred in this file? -- takaya
	keyFileName    = "testseed.hex"
	headerFileName = "headers.bin"

	// VERSION hardcoded for now, probably ok...?
	// 70012 is for segnet... make this an init var?
	VERSION = 70012
)

// GimmeFilter ... or I'm gonna fade away
func (s *SPVCon) GimmeFilter() (*bloom.Filter, error) {

	s.TrackingAdrsMtx.Lock()
	defer s.TrackingAdrsMtx.Unlock()
	s.TrackingOPsMtx.Lock()
	defer s.TrackingOPsMtx.Unlock()

	filterElements := uint32(len(s.TrackingAdrs) + (len(s.TrackingOPs)))

	f := bloom.NewFilter(filterElements, 0, 0.000001, wire.BloomUpdateAll)

	// note there could be false positives since we're just looking
	// for the 20 byte PKH without the opcodes.
	for a160 := range s.TrackingAdrs { // add 20-byte pubkeyhash
		//		log.Printf("adding address hash %x\n", a160)
		f.Add(a160[:])
	}
	//	for _, u := range allUtxos {
	//		f.AddOutPoint(&u.Op)
	//	}

	// actually... we should monitor addresses, not txids, right?
	// or no...?
	for wop := range s.TrackingOPs {
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

	// get lock for adrs / outpoints
	s.TrackingAdrsMtx.Lock()
	defer s.TrackingAdrsMtx.Unlock()
	s.TrackingOPsMtx.Lock()
	defer s.TrackingOPsMtx.Unlock()

	// start with optimism.  We may gain money.  Iterate through all output scripts.
	for i, out := range tx.TxOut {
		// create outpoint of what we're looking at
		op := wire.NewOutPoint(&txid, uint32(i))

		// 20 byte pubkey hash of this txout (if any)
		var adr20 [20]byte
		copy(adr20[:], lnutil.KeyHashFromPkScript(out.PkScript))
		// when we gain utxo, set as gain so we can return a match, but
		// also go through all gained utxos and register to track them

		//		log.Printf("got output key %x ", adr20)
		if s.TrackingAdrs[adr20] {
			gain = true
			s.TrackingOPs[*op] = true
		} else {
			//			log.Printf(" no match\n")
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

// IngestMerkleBlock ...
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
		err = s.OKTxid(txid, hah.height)
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

// IngestHeaders takes in a bunch of headers, checks them,
// and if they're OK, appends them to the local header file.
// If there are no headers, it assumes we're done and returns false.
// Otherwise it assumes there's more to request and returns true.
func (s *SPVCon) IngestHeaders(m *wire.MsgHeaders) (bool, error) {

	// headerChainLength is how many headers we give to the
	// verification function.  In bitcoin you never need more than 2016 previous
	// headers to figure out the validity of the next; some alcoins need more
	// though, like 4K or so.
	//	headerChainLength := 4096

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
	// even though we will be doing a bunch without writing, should be
	// OK performance-wise to keep it locked for this function duration,
	// because verification is pretty quick.
	defer s.headerMutex.Unlock()

	reorgHeight, err := CheckHeaderChain(s.headerFile, m.Headers, s.Param)
	if err != nil {
		// insufficient depth reorg means we're still trying to sync up?
		// really, the re-org hasn't been proven; if the remote node
		// provides us with a new block we'll ask again.
		if reorgHeight == -1 {
			log.Printf("Header error: %s\n", err.Error())
			return false, nil
		}
		// some other error
		return false, err
	}

	// truncate header file if reorg happens
	if reorgHeight != 0 {
		fileHeight := reorgHeight - s.Param.StartHeight
		err = s.headerFile.Truncate(int64(fileHeight) * 80)
		if err != nil {
			return false, err
		}

		// also we need to tell the upstream modules that a reorg happened
		s.CurrentHeightChan <- reorgHeight
		s.syncHeight = reorgHeight
	}

	// a header message is all or nothing; if we think there's something
	// wrong with it, we don't take any of their headers
	for _, resphdr := range m.Headers {
		// write to end of file
		err = resphdr.Serialize(s.headerFile)
		if err != nil {
			return false, err
		}
	}
	log.Printf("Added %d headers OK.", len(m.Headers))
	return true, nil
}

// AskForHeaders ...
func (s *SPVCon) AskForHeaders() error {
	ghdr := wire.NewMsgGetHeaders()
	ghdr.ProtocolVersion = s.localVersion

	tipheight := s.GetHeaderTipHeight()
	log.Printf("got header tip height %d\n", tipheight)
	// get tip header, as well as a few older ones (inefficient...?)
	// yes, inefficient; really we should use "getheaders" and skip some of this

	tipheader, err := s.GetHeaderAtHeight(tipheight)
	if err != nil {
		log.Printf("AskForHeaders GetHeaderAtHeight error\n")
		return err
	}

	tHash := tipheader.BlockHash()
	err = ghdr.AddBlockLocatorHash(&tHash)
	if err != nil {
		return err
	}

	backnum := int32(1)

	// add more blockhashes in there if we're high enough
	for tipheight > s.Param.StartHeight+backnum {
		backhdr, err := s.GetHeaderAtHeight(tipheight - backnum)
		if err != nil {
			return err
		}
		backhash := backhdr.BlockHash()

		err = ghdr.AddBlockLocatorHash(&backhash)
		if err != nil {
			return err
		}

		// send the most recent 10 blockhashes, then get sparse
		if backnum > 10 {
			backnum <<= 2
		} else {
			backnum++
		}
	}

	log.Printf("get headers message has %d header hashes, first one is %s\n",
		len(ghdr.BlockLocatorHashes), ghdr.BlockLocatorHashes[0].String())

	s.outMsgQueue <- ghdr
	return nil
}

// AskForBlocks requests blocks from current to last
// right now this asks for 1 block per getData message.
// Maybe it's faster to ask for many in each message?
func (s *SPVCon) AskForBlocks() error {
	var hdr wire.BlockHeader

	s.headerMutex.Lock() // lock just to check filesize
	stat, err := os.Stat(s.headerFile.Name())
	if err != nil {
		return err
	}
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
		var iv1 *wire.InvVect
		// if hardmode, ask for legit blocks, none of this ralphy stuff
		// I don't think you can have a queue for SPV.  You miss stuff.
		// also ask if someone wants rawblocks, like the watchtower
		if s.HardMode || s.RawBlockActive {
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
