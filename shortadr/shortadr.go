package shortadr

import (
	"bytes"
	"encoding/binary"
	//"encoding/hex"
	"github.com/btcsuite/fastsha256"
	"github.com/mit-dci/lit/lnutil"
	"log"
)

type ShortReply struct {
	BestNonce uint64
	BestHash  [20]byte
}

type VanityReply struct {
	BestNonce uint64
	BestHash  string
}

/*
initial short address code.  You can spin up a bunch of worker goroutines
(1 per core?)  and give them different IDs.  They will try sequential nonces,
and report back when they find nonces that have more work than the minWork they're
given.  They'll keep reporting better and better nonces until either they loop
around the uint64 (which will take a long time, and should be avoided), or they're
given a kill command via the stop channel.
*/

// AdrWorker is a goroutine that looks for a short hash.
// pub is the pub key to work on.  ID & Nonce is the nonce to start at.
// BestNonce is a channel that returns the best nonce each time one is found
// stop is a channel that kills this worker
func ShortAdrWorker(
	pub [33]byte, id, nonce uint64, bestBits uint8,
	vanity chan ShortReply, stop chan bool) ([20]byte, uint64) {

	var empty [20]byte
	var hash [20]byte //define outside to save on assignement delays each time
	var bits uint8
	defer close(vanity)
	for {
		select { // select here so we don't block on an unrequested mblock
		case _ = <-stop: // got order to stop
			return empty, uint64(0)
		default:
			hash = DoOneTry(pub, id, nonce)
			bits = CheckWork(hash)
			if bits > bestBits {
				bestBits = bits
				res := new(ShortReply)
				res.BestNonce = nonce
				res.BestHash = hash
				log.Println("LOOK HERE", hash, nonce, bits)
				vanity <- *res
				//return empty, uint64(0)
			}
		}
		nonce++
	}
	return hash, nonce
}

func FunAdrWorker(
	pub [33]byte, id, nonce uint64, bestStr string,
	vanity chan VanityReply, stop chan bool) ([20]byte, uint64) {

	var empty [20]byte
	var hash [20]byte //define outside to save on assignement delays each time
	bestStrLen := len(bestStr)
	defer close(vanity)
	log.Println("Vanity mode: On")
	for {
		select { // select here so we don't block on an unrequested mblock
		case _ = <-stop: // got order to stop
			return empty, uint64(0)
		default:
			hash = DoOneTry(pub, id, nonce)
			if lnutil.LitFunAdrFromPubkey(hash)[3:3+bestStrLen] == bestStr {
				res := new(VanityReply)
				res.BestNonce = nonce
				res.BestHash = lnutil.LitFunAdrFromPubkey(hash)
				vanity <- *res
			}
		}
		nonce++
	}
	return hash, nonce
}

// doOneTry is a single hash attempt with a key and nonce
func DoOneTry(key [33]byte, id, nonce uint64) [20]byte {
	var buf bytes.Buffer
	buf.Write(key[:])
	//binary.Write(&buf, binary.BigEndian, id)
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
