/* this is blockchain technology.  Well, except without the blocks.
Really it's header chain technology.
The blocks themselves don't really make a chain.  Just the headers do.
*/

package uspv

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/mit-dci/lit/logging"

	"github.com/mit-dci/lit/btcutil/blockchain"

	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/wire"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func moreWork(a, b []*wire.BlockHeader, p *coinparam.Params) bool {
	isMoreWork := false
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	// if (&a[0].MerkleRoot).IsEqual(&b[0].MerkleRoot) { this always returns false,
	// so we use the String() method to convert them, thing stole half an hour..
	pos := 0 //can safely assume this thanks to the first check
	for i := min(len(a), len(b)) - 1; i >= 1; i-- {
		hash := a[i-1].BlockHash()
		if a[i].PrevBlock.IsEqual(&hash) && b[i].PrevBlock.IsEqual(&hash) {
			isMoreWork = true
			pos = i
			break
		}
	}
	if !isMoreWork {
		return isMoreWork
	} else {
		var a1, b1 []*wire.BlockHeader
		a1 = a[pos:]
		b1 = b[pos:]
		workA := big.NewInt(0) // since raw declarations don't work, lets set it to 0
		workB := big.NewInt(0) // since raw declarations don't work, lets set it to 0
		for i := 0; i < len(a1); i++ {
			workA.Add(blockchain.CalcWork(a1[0].Bits), workA)
		}
		for i := 0; i < len(b1); i++ {
			//logging.Info(i)
			workB.Add(blockchain.CalcWork(b1[i].Bits), workB)
		}
		logging.Info("Work done by alt chains A and B are: ")
		logging.Info(workA, workB)
		// due to cmp's stquirks in big, we can't return directly
		if workA.Cmp(workB) > 0 { // if chain A does more work than B return true
			return isMoreWork // true
		}
		return !isMoreWork // false
	}
}

// checkProofOfWork verifies the header hashes into something
// lower than specified by the 4-byte bits field.
func checkProofOfWork(header wire.BlockHeader, p *coinparam.Params, height int32) bool {

	target := blockchain.CompactToBig(header.Bits)

	// The target must more than 0.  Why can you even encode negative...
	if target.Sign() <= 0 {
		logging.Errorf("block target %064x is neagtive(??)\n", target.Bytes())
		return false
	}
	// The target must be less than the maximum allowed (difficulty 1)
	if target.Cmp(p.PowLimit) > 0 {
		logging.Errorf("block target %064x is "+
			"higher than max of %064x", target, p.PowLimit.Bytes())
		return false
	}

	// The header hash must be less than the claimed target in the header.
	var buf bytes.Buffer
	_ = wire.WriteBlockHeader(&buf, 0, &header)

	blockHash := p.PoWFunction(buf.Bytes(), height)

	hashNum := new(big.Int)

	hashNum = blockchain.HashToBig(&blockHash)
	if hashNum.Cmp(target) > 0 {
		logging.Errorf("block hash %064x is higher than "+
			"required target of %064x", hashNum, target)
		return false
	}
	return true
}

// GetHeaderAtHeight gives back a header at the specified height
func (s *SPVCon) GetHeaderAtHeight(h int32) (*wire.BlockHeader, error) {
	s.headerMutex.Lock()
	defer s.headerMutex.Unlock()

	// height is reduced by startHeight
	h = h - s.Param.StartHeight

	// seek to that header
	_, err := s.headerFile.Seek(int64(80*h), os.SEEK_SET)
	if err != nil {
		logging.Error(err)
		return nil, err
	}

	hdr := new(wire.BlockHeader)
	err = hdr.Deserialize(s.headerFile)
	if err != nil {
		logging.Error(err)
		return nil, err
	}
	return hdr, nil
}

// GetHeaderTipHeight gives back a header at the specified height.
func (s *SPVCon) GetHeaderTipHeight() int32 {
	s.headerMutex.Lock() // start header file ops
	defer s.headerMutex.Unlock()
	info, err := s.headerFile.Stat()
	if err != nil {
		logging.Errorf("Header file error: %s", err.Error())
		return 0
	}
	headerFileSize := info.Size()
	if headerFileSize == 0 || headerFileSize%80 != 0 { // header file broken
		// try to fix it!
		s.headerFile.Truncate(headerFileSize - (headerFileSize % 80))
		logging.Errorf("ERROR: Header file not a multiple of 80 bytes. Truncating")
	}
	// subtract 1 as we want the start of the tip offset, not the end
	return int32(headerFileSize/80) + s.Param.StartHeight - 1
}

// FindHeader will try to find where the header you give it is.
// it runs backwards to find it and gives up after 1000 headers
func FindHeader(r io.ReadSeeker, hdr wire.BlockHeader) (int32, error) {

	var cur wire.BlockHeader

	for tries := 1; tries < 2200; tries++ {
		offset, err := r.Seek(int64(-80*tries), os.SEEK_END)
		if err != nil {
			logging.Error(err)
			return -1, err
		}

		//	for blkhash.IsEqual(&target) {
		err = cur.Deserialize(r)
		if err != nil {
			logging.Error(err)
			return -1, err
		}
		curhash := cur.BlockHash()

		//		logging.Infof("try %d %s\n", tries, curhash.String())

		if hdr.PrevBlock.IsEqual(&curhash) {
			return int32(offset / 80), nil
		}
	}

	return 0, nil
}

// CheckHeaderChain takes in the headers message and sees if they all validate.
// This function also needs read access to the previous headers.
// Does not deal with re-orgs; assumes new headers link to tip
// returns true if *all* headers are cool, false if there is any problem
// Note we don't know what the height is, just the relative height.
// returnin nil means it worked
// returns an int32 usually 0, but if there's a reorg, shows what height to
// reorg back to before adding on the headers
func CheckHeaderChain(
	r io.ReadSeeker, inHeaders []*wire.BlockHeader,
	p *coinparam.Params) (int32, error) {

	// make sure we actually got new headers
	if len(inHeaders) < 1 {
		return 0, fmt.Errorf(
			"CheckHeaderChain: headers message doesn't have any headers.")
	}

	// first, look through all the incoming headers to make sure
	// they're at least self-consistent.  Do this before even
	// checking that they link to anything; it's all in-ram and quick
	for i, hdr := range inHeaders {
		// check they link to each other
		// That whole 'blockchain' thing.
		if i > 1 {
			hash := inHeaders[i-1].BlockHash()
			if !hdr.PrevBlock.IsEqual(&hash) {
				return 0, fmt.Errorf(
					"headers %d and %d in header message don't link", i, i-1)
			}
		}

		// check that header version is non-negative (fork detect)
		if hdr.Version < 0 {
			return 0, fmt.Errorf(
				"header %d in message has negative version (hard fork?)", i)
		}
	}
	// incoming header message is internally consistent, now check that it
	// links with what we have on disk

	epochLength := int32(p.TargetTimespan / p.TargetTimePerBlock)

	if p.MinHeaders > 0 {
		epochLength = p.MinHeaders
	}

	// seek to start of last header
	pos, err := r.Seek(-80, os.SEEK_END)
	if err != nil {
		logging.Error(err)
		return 0, err
	}
	logging.Infof("header file position: %d\n", pos)
	if pos%80 != 0 {
		return 0, fmt.Errorf(
			"CheckHeaderChain: Header file not a multiple of 80 bytes.")
	}

	// we know incoming height; it's startheight + all the headers on disk + 1
	height := int32(pos/80) + p.StartHeight + 1

	// see if we don't have enough & load em all.
	var numheaders int32 // number of headers to read

	// load only last epoch if there are a lot on disk
	if pos > int64(80*(epochLength+1)) {
		_, err = r.Seek(int64(-80*(epochLength+1)), os.SEEK_END)
		numheaders = epochLength + 1
	} else { // otherwise load everything, start at byte 0
		_, err = r.Seek(0, os.SEEK_SET)
		numheaders = height - p.StartHeight
	}
	if err != nil { // seems like it will always be ok here..?
		return 0, err
	}

	// weird off-by-1 stuff here; makes numheaders, incluing the 0th
	oldHeaders := make([]*wire.BlockHeader, numheaders)
	logging.Infof("made %d header slice\n", len(oldHeaders))
	// load a bunch of headers from disk into ram
	for i := range oldHeaders {
		// read from file at current offset
		oldHeaders[i] = new(wire.BlockHeader)
		err = oldHeaders[i].Deserialize(r)
		if err != nil {
			logging.Errorf("CheckHeaderChain ran out of file at oldheader %d\n", i)
			return 0, err
		}
	}

	tiphash := oldHeaders[len(oldHeaders)-1].BlockHash()

	var attachHeight int32
	// make sure the first header in the message points to our on-disk tip
	if !inHeaders[0].PrevBlock.IsEqual(&tiphash) {

		// find where it points to

		attachHeight, err = FindHeader(r, *inHeaders[0])
		if err != nil {
			return 0, fmt.Errorf(
				"CheckHeaderChain: header message doesn't attach to tip or anywhere.")
		}

		// adjust attachHeight by adding the startheight
		attachHeight += p.StartHeight

		logging.Infof("Header %s attaches at height %d\n",
			inHeaders[0].BlockHash().String(), attachHeight)

		// if we've been given insufficient headers, don't reorg, but
		// ask for more headers.

		// Check between two chains with attachHeight+int32(len(inHeaders)) and height
		// lengths, then Compare the work associated with them.
		// 1. check whether they really are from the same chain i.e. start from the same Header
		// 2. Find the most recent common Block
		// 3. Calculate work on both chains from that block
		// 4. Return true or false based on which one is better.
		// the two arrays are chain+inHeaders+attachHeight and the height chain itself
		// 2,3,4 -? fn

		if moreWork(inHeaders, oldHeaders, p) {
			// pretty sure this won't work without testing
			// if attachHeight+int32(len(inHeaders)) < height {
			return -1, fmt.Errorf(
				"reorg message up to height %d, but have up to %d",
				attachHeight+int32(len(inHeaders)), height-1)
		}

		logging.Infof("reorg from height %d to %d",
			height-1, attachHeight+int32(len(inHeaders)))

		// reorg is go, snip to attach height
		reorgDepth := height - attachHeight
		if reorgDepth > numheaders {
			return -1, fmt.Errorf("Reorg depth greater than number of headers")
		}
		oldHeaders = oldHeaders[:numheaders-reorgDepth]
	}

	prevHeaders := oldHeaders

	// check difficulty adjustments in the new headers
	// since we call this many times, append each time
	for i, hdr := range inHeaders {
		if height+int32(i) > p.AssumeDiffBefore {
			// check if there's a valid proof of work.  That whole "Bitcoin" thing.
			if !checkProofOfWork(*hdr, p, height+int32(i)) {
				logging.Error("header in message has bad proof of work")
				return 0, fmt.Errorf("header %d in message has bad proof of work", i)
			}
			// build slice of "previous" headers
			prevHeaders = append(prevHeaders, inHeaders[i])
			rightBits, err := p.DiffCalcFunction(prevHeaders, height+int32(i), p)
			if err != nil {
				logging.Error(err)
				return 0, fmt.Errorf("Error calculating Block %d %s difficulty. %s",
					int(height)+i, hdr.BlockHash().String(), err.Error())
			}

			// vertcoin diff adjustment not yet implemented
			// TODO - get rid of coin specific workaround
			if hdr.Bits != rightBits && (p.Name != "vtctest" && p.Name != "vtc") && !p.TestCoin {
				return 0, fmt.Errorf("Block %d %s incorrect difficulty.  Read %x, expect %x",
					int(height)+i, hdr.BlockHash().String(), hdr.Bits, rightBits)
			}
		}
	}

	return attachHeight, nil
}
