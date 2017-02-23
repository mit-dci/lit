package uspv

import (
	"bytes"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/bloom"
	"github.com/mit-dci/lit/lnutil"
)

var (
	WitMagicBytes = []byte{0x6a, 0x24, 0xaa, 0x21, 0xa9, 0xed}
)

// BlockOK checks for block self-consistency.
// If the block has no wintess txs, and no coinbase witness commitment,
// it only checks the tx merkle root.  If either a witness commitment or
// any witnesses are detected, it also checks that as well.
// Returns false if anything goes wrong, true if everything is fine.
func BlockOK(blk wire.MsgBlock) bool {
	var txids, wtxids []*chainhash.Hash // txids and wtxids
	// witMode true if any tx has a wintess OR coinbase has wit commit
	witMode := false

	for _, tx := range blk.Transactions { // make slice of (w)/txids
		txid := tx.TxHash()
		txids = append(txids, &txid)
		if !witMode && tx.HasWitness() {
			witMode = true
		}
	}
	if witMode {
		for _, wtx := range blk.Transactions {
			wtxid := wtx.WitnessHash()
			wtxids = append(wtxids, &wtxid)
		}
	}

	var commitBytes []byte
	// try to extract coinbase witness commitment (even if !witMode)
	cb := blk.Transactions[0]                 // get coinbase tx
	for i := len(cb.TxOut) - 1; i >= 0; i-- { // start at the last txout
		if bytes.HasPrefix(cb.TxOut[i].PkScript, WitMagicBytes) &&
			len(cb.TxOut[i].PkScript) > 37 {
			// 38 bytes or more, and starts with WitMagicBytes is a hit
			commitBytes = cb.TxOut[i].PkScript[6:38]
			witMode = true // it there is a wit commit it must be valid
		}
	}

	if witMode { // witmode, so check witness tree
		// first find ways witMode can be disqualified
		if len(commitBytes) != 32 {
			// witness in block but didn't find a wintess commitment; fail
			log.Printf("block %s has witness but no witcommit",
				blk.BlockHash().String())
			return false
		}
		if len(cb.TxIn) != 1 {
			log.Printf("block %s coinbase tx has %d txins (must be 1)",
				blk.BlockHash().String(), len(cb.TxIn))
			return false
		}
		if len(cb.TxIn[0].Witness) != 1 {
			log.Printf("block %s coinbase has %d witnesses (must be 1)",
				blk.BlockHash().String(), len(cb.TxIn[0].Witness))
			return false
		}

		if len(cb.TxIn[0].Witness[0]) != 32 {
			log.Printf("block %s coinbase has %d byte witness nonce (not 32)",
				blk.BlockHash().String(), len(cb.TxIn[0].Witness[0]))
			return false
		}
		// witness nonce is the cb's witness, subject to above constraints
		witNonce, err := chainhash.NewHash(cb.TxIn[0].Witness[0])
		if err != nil {
			log.Printf("Witness nonce error: %s", err.Error())
			return false // not sure why that'd happen but fail
		}

		var empty [32]byte
		wtxids[0].SetBytes(empty[:]) // coinbase wtxid is 0x00...00

		// witness root calculated from wtixds
		witRoot := calcRoot(wtxids)

		calcWitCommit := chainhash.DoubleHashH(
			append(witRoot.CloneBytes(), witNonce.CloneBytes()...))

		// witness root given in coinbase op_return
		givenWitCommit, err := chainhash.NewHash(commitBytes)
		if err != nil {
			log.Printf("Witness root error: %s", err.Error())
			return false // not sure why that'd happen but fail
		}
		// they should be the same.  If not, fail.
		if !calcWitCommit.IsEqual(givenWitCommit) {
			log.Printf("Block %s witRoot error: calc %s given %s",
				blk.BlockHash().String(),
				calcWitCommit.String(), givenWitCommit.String())
			return false
		}
	}

	// got through witMode check so that should be OK;
	// check regular txid merkleroot.  Which is, like, trivial.
	return blk.Header.MerkleRoot.IsEqual(calcRoot(txids))
}

// calcRoot calculates the merkle root of a slice of hashes.
func calcRoot(hashes []*chainhash.Hash) *chainhash.Hash {
	for len(hashes) < int(nextPowerOfTwo(uint32(len(hashes)))) {
		hashes = append(hashes, nil) // pad out hash slice to get the full base
	}
	for len(hashes) > 1 { // calculate merkle root. Terse, eh?
		hashes = append(hashes[2:], MakeMerkleParent(hashes[0], hashes[1]))
	}
	return hashes[0]
}

// RefilterLocal reconstructs the local in-memory bloom filter.  It does
// this by calling GimmeFilter() but doesn't broadcast the result.
func (s *SPVCon) Refilter(f *bloom.Filter) {
	if s.HardMode {
		s.localFilter.Reload(f.MsgFilterLoad())
	} else {
		s.SendFilter(f)
	}
}

// IngestBlock is like IngestMerkleBlock but aralphic
// different enough that it's better to have 2 separate functions
func (s *SPVCon) IngestBlock(m *wire.MsgBlock) {
	var err error

	// hand block over to the watchtower via the RawBlockSender chan
	// omit this if qln not connected
	/*
		if cap(w.SpvHook.RawBlockSender) != 0 {
			w.SpvHook.RawBlockSender <- m
		} else {
			fmt.Printf("Watchtower not initialized\n")
		}
	*/
	ok := BlockOK(*m) // check block self-consistency
	if !ok {
		fmt.Printf("block %s not OK!!11\n", m.BlockHash().String())
		return
	}

	var hah HashAndHeight
	select { // select here so we don't block on an unrequested mblock
	case hah = <-s.blockQueue: // pop height off mblock queue
		break
	default:
		log.Printf("Unrequested full block")
		return
	}

	newBlockHash := m.Header.BlockHash()
	if !hah.blockhash.IsEqual(&newBlockHash) {
		log.Printf("full block out of order error")
		return
	}

	fPositive := 0 // local filter false positives
	reFilter := 2  // after that many false positives, regenerate filter.
	// 2?  Making it up.  False positives have disk i/o cost, and regenning
	// the filter also has costs.  With a large local filter, false positives
	//	 should be rare.

	// iterate through all txs in the block, looking for matches.
	// use a local bloom filter to ignore txs that don't affect us
	txs := make([]*wire.MsgTx, 0, len(m.Transactions))
	for _, tx := range m.Transactions {
		utilTx := btcutil.NewTx(tx)
		// find txs that look like hits
		if s.localFilter.MatchTxAndUpdate(utilTx) {
			txs = append(txs, tx)
		}
	}
	// anything good?
	if len(txs) > 0 {
		// ingest all the txs
		for _, tx := range txs {
			s.TxUpToWallit <- lnutil.TxAndHeight{tx, hah.height}
		}
	}

	if fPositive > reFilter {
		fmt.Printf("%d filter false positives in this block\n", fPositive)
		filt, err := s.GimmeFilter()
		if err != nil {
			log.Printf("Refilter error: %s\n", err.Error())
			return
		}
		s.Refilter(filt)
	}
	// write to db that we've sync'd to the height indicated in the
	// merkle block.  This isn't QUITE true since we haven't actually gotten
	// the txs yet but if there are problems with the txs we should backtrack.
	s.CurrentHeightChan <- hah.height

	fmt.Printf("ingested full block %s height %d OK\n",
		m.Header.BlockHash().String(), hah.height)

	if hah.final { // check sync end
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
