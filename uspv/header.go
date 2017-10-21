/* this is blockchain technology.  Well, except without the blocks.
Really it's header chain technology.
The blocks themselves don't really make a chain.  Just the headers do.
*/

package uspv

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"

	"github.com/adiabat/btcd/blockchain"

	"github.com/adiabat/btcd/wire"
	"github.com/mit-dci/lit/coinparam"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func moreWork(a, b []*wire.BlockHeader) bool {
	if a[0].MerkleRoot != b[0].MerkleRoot {
		log.Printf("Chains don't start with the same header. Quitting.!")
		return false
	}
	var flag int = 0
	var pos int = 0 //can safely assume this thanks to the first check
	for i := min(len(a), len(b)); i >= 0; i-- {
		if a[i].MerkleRoot == b[i].MerkleRoot {
			flag = 1
			pos = i
			break
		}
	}
	if flag == 1 {
		a1 := a[pos+1:]
		b1 := b[pos+1:]
		var work_a *big.Int
		var work_b *big.Int
		for i := 0; i < len(a1); i++ {
			work_a.Add(blockchain.CalcWork(a1[i].Bits), work_a)
		}
		for i := 0; i < len(b1); i++ {
			work_b.Add(blockchain.CalcWork(b1[i].Bits), work_b)
		}
		if work_a.Cmp(work_b) < 0 {
			// traditional integer comparison doesn't work here, so use Cmp instead.
			return true
		}
	}
	return false
}

// checkProofOfWork verifies the header hashes into something
// lower than specified by the 4-byte bits field.
func checkProofOfWork(header wire.BlockHeader, p *coinparam.Params) bool {

	target := blockchain.CompactToBig(header.Bits)

	// The target must more than 0.  Why can you even encode negative...
	if target.Sign() <= 0 {
		log.Printf("block target %064x is neagtive(??)\n", target.Bytes())
		return false
	}
	// The target must be less than the maximum allowed (difficulty 1)
	if target.Cmp(p.PowLimit) > 0 {
		log.Printf("block target %064x is "+
			"higher than max of %064x", target, p.PowLimit.Bytes())
		return false
	}

	// The header hash must be less than the claimed target in the header.
	var buf bytes.Buffer
	_ = wire.WriteBlockHeader(&buf, 0, &header)

	blockHash := p.PoWFunction(buf.Bytes())

	hashNum := new(big.Int)

	hashNum = blockchain.HashToBig(&blockHash)
	if hashNum.Cmp(target) > 0 {
		log.Printf("block hash %064x is higher than "+
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
		return nil, err
	}

	hdr := new(wire.BlockHeader)
	err = hdr.Deserialize(s.headerFile)
	if err != nil {
		return nil, err
	}

	return hdr, nil
}

// GetHeaderAtHeight gives back a header at the specified height
func (s *SPVCon) GetHeaderTipHeight() int32 {
	s.headerMutex.Lock() // start header file ops
	defer s.headerMutex.Unlock()
	info, err := s.headerFile.Stat()
	if err != nil {
		log.Printf("Header file error: %s", err.Error())
		return 0
	}
	headerFileSize := info.Size()
	if headerFileSize == 0 || headerFileSize%80 != 0 { // header file broken
		// try to fix it!
		s.headerFile.Truncate(headerFileSize - (headerFileSize % 80))
		log.Printf("ERROR: Header file not a multiple of 80 bytes. Truncating")
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
			return -1, err
		}

		//	for blkhash.IsEqual(&target) {
		err = cur.Deserialize(r)
		if err != nil {
			return -1, err
		}
		curhash := cur.BlockHash()

		//		fmt.Printf("try %d %s\n", tries, curhash.String())

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
		// check if there's a valid proof of work.  That whole "Bitcoin" thing.
		if !checkProofOfWork(*hdr, p) {
			return 0, fmt.Errorf("header %d in message has bad proof of work", i)
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

	// seek to start of last header
	pos, err := r.Seek(-80, os.SEEK_END)
	if err != nil {
		return 0, err
	}
	fmt.Printf("pos: %d\n", pos)
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
	fmt.Printf("made %d header slice\n", len(oldHeaders))
	// load a bunch of headers from disk into ram
	for i, _ := range oldHeaders {
		// read from file at current offset
		oldHeaders[i] = new(wire.BlockHeader)
		err = oldHeaders[i].Deserialize(r)
		if err != nil {
			log.Printf("CheckHeaderChain ran out of file at oldheader %d\n", i)
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

		log.Printf("Header %s attaches at height %d\n",
			inHeaders[0].BlockHash().String(), attachHeight)

		// TODO check for more work here instead of length.  This is wrong...
		// if we've been given insufficient headers, don't reorg, but
		// ask for more headers.

		if attachHeight+int32(len(inHeaders)) < height {
			return -1, fmt.Errorf(
				"reorg message up to height %d, but have up to %d",
				attachHeight+int32(len(inHeaders)), height-1)
		}

		log.Printf("reorg from height %d to %d",
			height-1, attachHeight+int32(len(inHeaders)))

		// reorg is go, snip to attach height
		reorgDepth := height - attachHeight
		oldHeaders = oldHeaders[:numheaders-reorgDepth]
	}

	prevHeaders := oldHeaders

	// check difficulty adjustments in the new headers
	// since we call this many times, append each time
	for i, hdr := range inHeaders {
		// build slice of "previous" headers
		prevHeaders = append(prevHeaders, inHeaders[i])
		rightBits, err := p.DiffCalcFunction(prevHeaders, height+int32(i), p)
		if err != nil {
			return 0, fmt.Errorf("Error calculating Block %d %s difficuly. %s",
				int(height)+i, hdr.BlockHash().String(), err.Error())
		}

		if hdr.Bits != rightBits {
			return 0, fmt.Errorf("Block %d %s incorrect difficuly.  Read %x, expect %x",
				int(height)+i, hdr.BlockHash().String(), hdr.Bits, rightBits)
		}
	}

	return attachHeight, nil
}

func CheckHeader(r io.ReadSeeker, height, startheight int32, p *coinparam.Params) bool {
	// startHeight is the height the file starts at
	// header start must be 0 mod 2106
	var err error
	var cur, prev wire.BlockHeader
	// don't try to verfy the genesis block.  That way madness lies.
	if height == 0 {
		return true
	}

	offsetHeight := height - startheight
	// initial load of headers

	// load previous and current.

	// seek to n-1 header
	_, err = r.Seek(int64(80*(offsetHeight-1)), os.SEEK_SET)
	if err != nil {
		log.Printf(err.Error())
		return false
	}
	// read in n-1
	err = prev.Deserialize(r)
	if err != nil {
		log.Printf(err.Error())
		return false
	}

	// seek to curHeight header and read in
	_, err = r.Seek(int64(80*(offsetHeight)), os.SEEK_SET)
	if err != nil {
		log.Printf(err.Error())
		return false
	}
	err = cur.Deserialize(r)
	if err != nil {
		log.Printf(err.Error())
		return false
	}

	// get hash of n-1 header
	prevHash := prev.BlockHash()
	// check if headers link together.  That whole 'blockchain' thing.
	if prevHash.IsEqual(&cur.PrevBlock) == false {
		log.Printf("Headers %d and %d don't link.\n",
			height-1, height)
		log.Printf("%s - %s",
			prev.BlockHash().String(), cur.BlockHash().String())
		return false
	}

	// Check that the difficulty bits are correct
	if offsetHeight > 0 && height >= p.AssumeDiffBefore {
		//		rightBits, err := p.DiffCalcFunction(r, height, startheight, p)
		rightBits, err := p.DiffCalcFunction(nil, height, p)
		if err != nil {
			log.Printf("Error calculating Block %d %s difficuly. %s\n",
				height, cur.BlockHash().String(), err.Error())
			return false
		}

		if cur.Bits != rightBits {
			log.Printf("Block %d %s incorrect difficuly.  Read %x, expect %x\n",
				height, cur.BlockHash().String(), cur.Bits, rightBits)
			return false
		}
	}

	// check if there's a valid proof of work.  That whole "Bitcoin" thing.
	if !checkProofOfWork(cur, p) {
		log.Printf("Block %d Bad proof of work.\n", height)
		return false
	}

	// Check for checkpoints
	for _, checkpoint := range p.Checkpoints {
		if checkpoint.Height == height {
			if *checkpoint.Hash != cur.BlockHash() {
				log.Printf("Block %d is not a valid checkpoint", height)
				return false
			}
			break
		}
	}

	// Not entirely sure why I need to do this, but otherwise the tip block
	// can go missing
	_, err = r.Seek(int64(80*(offsetHeight)), os.SEEK_SET)
	if err != nil {
		log.Printf(err.Error())
		return false
	}
	err = cur.Deserialize(r)
	if err != nil {
		log.Printf(err.Error())
		return false
	}

	return true // it must have worked if there's no errors and got to the end.
}

/* checkrange verifies a range of headers.  it checks their proof of work,
difficulty adjustments, and that they all link in to each other properly.
This is the only blockchain technology in the whole code base.
Returns false if anything bad happens.  Returns true if the range checks
out with no errors. */
func CheckRange(r io.ReadSeeker, first, last, startHeight int32, p *coinparam.Params) bool {
	for i := first; i <= last; i++ {
		if !CheckHeader(r, i, startHeight, p) {
			return false
		}
	}
	return true // all good.
}
