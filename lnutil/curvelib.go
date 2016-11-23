package lnutil

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// PrivKeyAddBytes adds bytes to a private key.
// NOTE that this modifies the key in place, overwriting it!!!!1
// If k is nil, does nothing and doesn't error (k stays nil)
func PrivKeyAddBytes(k *btcec.PrivateKey, b []byte) {
	if k == nil {
		return
	}
	// turn arg bytes into a bigint
	arg := new(big.Int).SetBytes(b)
	// add private key to arg
	k.D.Add(k.D, arg)
	// mod 2^256ish
	k.D.Mod(k.D, btcec.S256().N)
	// new key derived from this sum
	// D is already modified, need to update the pubkey x and y
	k.X, k.Y = btcec.S256().ScalarBaseMult(k.D.Bytes())
	return
}

// PubKeyAddBytes adds bytes to a public key.
// NOTE that this modifies the key in place, overwriting it!!!!1
func PubKeyAddBytes(k *btcec.PublicKey, b []byte) {
	// turn b into a point on the curve
	bx, by := btcec.S256().ScalarBaseMult(b)
	// add arg point to pubkey point
	k.X, k.Y = btcec.S256().Add(bx, by, k.X, k.Y)
	return
}

// PubKeyArrAddBytes adds a byte slice to a serialized point.
// You can't add scalars to a point, so you turn the bytes into a point,
// then add that point.
func PubKeyArrAddBytes(p *[33]byte, b []byte) error {
	pub, err := btcec.ParsePubKey(p[:], btcec.S256())
	if err != nil {
		return err
	}
	// turn b into a point on the curve
	bx, by := pub.ScalarBaseMult(b)
	// add arg point to pubkey point
	pub.X, pub.Y = btcec.S256().Add(bx, by, pub.X, pub.Y)
	copy(p[:], pub.SerializeCompressed())
	return nil
}

/* Key Aggregation

Note that this is not for signature aggregation in schnorr sigs; that's not here yet.
But we use the same construction.

If you want to put two pubkeys A, B together into composite pubkey C, you can't
just say C = A+B.  Because B might really be X-A, in which case C=X and the A
key is irrelevant.  C = A*h(A) + B*h(B) works but gets dangerous with lots of keys
due to generalized birthday attacks.  In the LN case that probably isn't relevant,
but we'll stick to the same constrction anyway.

First, concatenate all the keys together and hash that.
z = h(A, B...)
for generation of z, you need some ordering; everything else is commutative.
Here it does byte sorting.
Then add all the keys times the hash of z and themselves.
C = A*h(z, A) + B*(z, B) + ...
this works for lots of keys.  And is overkill for 2 but that's OK.
*/

// PubKeySlice are slices of serialized compressed pubkeys, which can be sorted.
type PubKeySlice []*[33]byte

// Make PubKeySlices sortable
func (p PubKeySlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p PubKeySlice) Len() int      { return len(p) }
func (p PubKeySlice) Less(i, j int) bool {
	return bytes.Compare(p[i][:], p[j][:]) == -1
}

// ComboCommit
func ComboCommit(pubKeys PubKeySlice) chainhash.Hash {
	// sort the pubkeys, smallest first
	sort.Sort(pubKeys)
	// feed em into the hash
	combo := make([]byte, len(pubKeys)*33)
	for i, k := range pubKeys {
		copy(combo[i*33:(i+1*33)], k[:])
	}
	return chainhash.HashH(combo)

}

// ###########################
// HAKD/Elkrem point functions

// AddPointArrs takes two 33 byte serialized points, adds them, and
// returns the sum as a 33 byte array.
// Silently returns a zero array if there's an input error
func AddPubs(a, b [33]byte) [33]byte {
	var c [33]byte
	apoint, err := btcec.ParsePubKey(a[:], btcec.S256())
	if err != nil {
		return c
	}
	bpoint, err := btcec.ParsePubKey(b[:], btcec.S256())
	if err != nil {
		return c
	}

	apoint.X, apoint.Y = btcec.S256().Add(apoint.X, apoint.Y, bpoint.X, bpoint.Y)
	copy(c[:], apoint.SerializeCompressed())

	return c
}

// HashToPub turns a 32 byte hash into a 33 byte serialized pubkey
func PubFromHash(h chainhash.Hash) (p [33]byte) {
	_, pub := btcec.PrivKeyFromBytes(btcec.S256(), h[:])
	copy(p[:], pub.SerializeCompressed())
	return
}
