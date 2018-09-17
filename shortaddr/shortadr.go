package shortadr

import (
	"bytes"
	"encoding/binary"

	"github.com/mit-dci/lit/crypto/fastsha256"
)

/*
initial short address code.  You can spin up a bunch of worker goroutines
(1 per core?)  and give them different IDs.  They will try sequential nonces,
and report back when they find nonces that have more work than the minWork they're
given.  They'll keep reporting better and better nonces until either they loop
around the uint64 (which will take a long time, and should be avoided), or they're
given a kill command via the stop channel.
*/

// AdrWorker is a gorouting that looks for a short hash.
// pub is the pub key to work on.  ID & Nonce is the nonce to start at.
// bestNonce is a channel that returns the best nonce each time one is found
// stop is a channel that kills this worker
func AdrWorker(
	pub [33]byte, id, nonce uint64, bestBits uint8,
	bestNonce chan uint64, stop chan bool) {

	for {
		select { // select here so we don't block on an unrequested mblock
		case _ = <-stop: // got order to stop
			return
		default:
			hash := DoOneTry(pub, id, nonce)
			bits := CheckWork(hash)
			if bits > bestBits {
				bestBits = bits
				bestNonce <- nonce
			}
		}
		nonce++
	}
}

// doOneTry is a single hash attempt with a key and nonce
func DoOneTry(key [33]byte, id, nonce uint64) (hash [20]byte) {
	var buf bytes.Buffer
	buf.Write(key[:])
	binary.Write(&buf, binary.BigEndian, id)
	binary.Write(&buf, binary.BigEndian, nonce)

	shaoutput := fastsha256.Sum256(buf.Bytes())
	copy(hash[:], shaoutput[:20])
	return
}

// CheckWork returns the number of leading 0 bits
// make it not crash with all zero hash arguments
func CheckWork(hash [20]byte) (i uint8) {
	for ; i < 160 && (hash[i/8]>>(7-(i%8)))&1 == 0; i++ {
	}
	return
}
