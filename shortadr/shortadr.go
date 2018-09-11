package shortadr

import (
	"bytes"
	"encoding/binary"
	"github.com/btcsuite/fastsha256"
	"github.com/mit-dci/lit/lnutil"
)

type ShortReply struct {
	BestNonce uint64
	BestHash  [20]byte
}

/*
You can spin up a bunch of worker goroutines (1 per core? user specified?)
and give them different IDs.  They will try sequential nonces,
and report back when they find nonces that have more work than the minWork they're
given.  They'll keep reporting better and better nonces until either they loop
around the uint64 (which will take a long time, and should be avoided), or they're
given a kill command via the stop channel. Right now, they've been configured to
give one suitable nonce per channel run
*/

// AdrWorker is a goroutine that looks for a short hash.
// pub is the pub key to work on.  ID & Nonce is the nonce to start at.
// BestNonce is a channel that returns the best nonce each time one is found
// stop is a channel that kills this worker
func ShortAdrWorker(
	pub [33]byte, id, nonce uint64, bestBits uint8,
	vanity chan ShortReply, stop chan bool) ([20]byte, uint64) {

	var empty, hash [20]byte
	var bits uint8
	defer close(vanity)
	for {
		select { // select here so we don't block on an unrequested mblock
		case _ = <-stop: // got order to stop
			return empty, uint64(0)
		default:
			hash = HashOnce(pub, id, nonce)
			bits = CheckWork(hash)
			if bits > bestBits {
				bestBits = bits
				res := new(ShortReply)
				res.BestNonce = nonce
				res.BestHash = hash
				vanity <- *res
			}
		}
		nonce++
	}
	return hash, nonce
}

// HashOnce is a single hash attempt with a key and nonce
func HashOnce(key [33]byte, id, nonce uint64) [20]byte {
	var buf bytes.Buffer
	buf.Write(key[:])
	binary.Write(&buf, binary.BigEndian, nonce)

	shaoutput := fastsha256.Sum256(buf.Bytes())
	var hash [20]byte
	copy(hash[:], shaoutput[:20])
	return hash
}

// CheckWork returns the number of leading 0 bits
func CheckWork(hash [20]byte) (i uint8) {
	for ; (hash[i/8]>>(7-(i%8)))&1 == 0; i++ {
		// hash[i/8] for the byte we're looking at
		// >>(7-(i%8)) right shift for the bit that we want
		// &1 to convert to single bit
	}
	return i
}

func GetShortPKH(s [33]byte, nonce uint64) string {
	adrBytes := HashOnce(s, 0, nonce)
	var adr []byte
	for i := 0; i < len(adrBytes); i++ {
		if adrBytes[i] != 0 {
			adr = append(adr, adrBytes[i])
		}
	}
	return lnutil.LitShortAdrFromPubkey(adr)
}
