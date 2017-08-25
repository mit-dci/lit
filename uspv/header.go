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

/* checkProofOfWork verifies the header hashes into something
lower than specified by the 4-byte bits field. */
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

	targethash := hdr.BlockHash()

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

		if targethash.IsEqual(&curhash) {
			return int32(offset / 80), nil
		}
	}

	return 0, nil
}

// CheckHeaderChain takes in the headers message and sees if they all validate.
// This function also needs read access to the previous headers.
// Does not deal with re-orgs
// returns true if *all* headers are cool, false if there is any problem
func CheckHeaderChain(
	r io.ReadSeeker, m wire.MsgHeaders, p *coinparam.Params) (bool, error) {

	var prevTip wire.BlockHeader

	// seek to last header
	pos, err := r.Seek(80, os.SEEK_END)
	if err != nil {
		return false, err
	}
	if pos%80 != 0 {
		return false, fmt.Errorf(
			"CheckHeaderChain: Header file not a multiple of 80 bytes.")
	}
	// get the height of this tip from the file length
	height := int32(pos/80) + p.StartHeight
	// read in last header
	err = prevTip.Deserialize(r)
	if err != nil {
		return false, err
	}

	tiphash := prevTip.BlockHash()

	if len(m.Headers) < 1 {
		return false, fmt.Errorf(
			"CheckHeaderChain: headers message doesn't have any headers.")
	}

	// make sure the first header in the message points to our on-disk tip
	if !m.Headers[0].PrevBlock.IsEqual(&tiphash) {
		return false, fmt.Errorf(
			"CheckHeaderChain: header message doesn't attach to tip.")
	}

	// run through all the new headers, checking what we can
	for i, hdr := range m.Headers {

		// check they link to each other
		// That whole 'blockchain' thing.
		if i > 1 {
			hash := hdr.BlockHash()
			if !hdr.PrevBlock.IsEqual(&hash) {
				return false, fmt.Errorf(
					"headers %d and %d in header message don't link", i, i-1)
			}
		}
	}

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
		rightBits, err := p.DiffCalcFunction(r, height, startheight, p)
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
