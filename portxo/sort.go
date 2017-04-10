package portxo

import (
	"bytes"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// txoSliceByBip69 is a sortable txo slice - same algo as txsort / BIP69
type TxoSliceByBip69 []*PorTxo

// Sort utxos just like txins -- Len, Less, Swap
func (s TxoSliceByBip69) Len() int      { return len(s) }
func (s TxoSliceByBip69) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// outpoint sort; First input hash (reversed / rpc-style), then index.
func (s TxoSliceByBip69) Less(i, j int) bool {
	// Input hashes are the same, so compare the index.
	ihash := s[i].Op.Hash
	jhash := s[j].Op.Hash
	if ihash == jhash {
		return s[i].Op.Index < s[j].Op.Index
	}
	// At this point, the hashes are not equal, so reverse them to
	// big-endian and return the result of the comparison.
	const hashSize = chainhash.HashSize
	for b := 0; b < hashSize/2; b++ {
		ihash[b], ihash[hashSize-1-b] = ihash[hashSize-1-b], ihash[b]
		jhash[b], jhash[hashSize-1-b] = jhash[hashSize-1-b], jhash[b]
	}
	return bytes.Compare(ihash[:], jhash[:]) == -1
}

// txoSliceByAmt is a sortable txo slice.  Sorts by value, and puts unconfirmed last.
// Also has sum functions for calculating balances
type TxoSliceByAmt []*PorTxo

func (s TxoSliceByAmt) Len() int      { return len(s) }
func (s TxoSliceByAmt) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// height 0 means you are lesser
func (s TxoSliceByAmt) Less(i, j int) bool {
	if s[i].Height == 0 && s[j].Height > 0 {
		return true
	}
	if s[j].Height == 0 && s[i].Height > 0 {
		return false
	}
	return s[i].Value < s[j].Value
}

func (s TxoSliceByAmt) Sum() int64 {
	var total int64
	for _, txo := range s {
		total += txo.Value
	}
	return total
}

func (s TxoSliceByAmt) SumWitness(currentHeight int32) int64 {
	var total int64
	for _, txo := range s {
		// check that it's witness,
		// then make sure it's confirmed, and any timeouts have passed
		if txo.Mode&FlagTxoWitness != 0 &&
			txo.Height > 0 && txo.Height+int32(txo.Seq) <= currentHeight {
			total += txo.Value
		}
	}
	return total
}

// KeyGenSortableSlice is a sortable slice of keygens. Shorter and lower numbers first.
type KeyGenSortableSlice []*KeyGen

func (s KeyGenSortableSlice) Len() int      { return len(s) }
func (s KeyGenSortableSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s KeyGenSortableSlice) Less(i, j int) bool {
	// first - shorter path is less
	if s[i].Depth < s[j].Depth {
		return true
	}
	if s[i].Depth > s[j].Depth {
		return false
	}

	// paths are the 	same, iterate and compare
	for x := uint8(0); x < s[i].Depth; x++ {
		if s[i].Step[x] < s[j].Step[x] {
			return true
		}
	}

	// if we got here, they're exactly the same
	return false
}
