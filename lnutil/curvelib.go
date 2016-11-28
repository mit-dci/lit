package lnutil

import (
	"bytes"
	"fmt"
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

// PubKeyMultiplyByHash multiplies a pubkey by a hash.
// returns nothing, modifies in place.  Probably the slowest curve operation.
func MultiplyPointByHash(k *btcec.PublicKey, h chainhash.Hash) {
	k.X, k.Y = btcec.S256().ScalarMult(k.X, k.Y, h[:])
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

// PubKeySlice are slices of pubkeys, which can be combined (and sorted)
type CombinablePubKeySlice []*btcec.PublicKey

// Make PubKeySlices sortable
func (p CombinablePubKeySlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p CombinablePubKeySlice) Len() int      { return len(p) }
func (p CombinablePubKeySlice) Less(i, j int) bool {
	return bytes.Compare(p[i].SerializeCompressed(), p[j].SerializeCompressed()) == -1
}

// CombinableSliceFromArrSlice turns an array of 33 byte pubkey arrays into a
// CombinablePubKeySlice which can then be combined.
// Varadic.  First time I've used that, seems appropriate here.
func PubsFromArrs(arrSlice ...[33]byte) (CombinablePubKeySlice, error) {
	if len(arrSlice) < 2 {
		return nil, fmt.Errorf("Need 2 or more pubkeys to combine")
	}
	var p CombinablePubKeySlice
	for _, arr := range arrSlice {
		nextPub, err := btcec.ParsePubKey(arr[:], btcec.S256())
		if err != nil {
			return nil, err
		}
		p = append(p, nextPub)
	}
	return p, nil
}

// ComboCommit generates the "combination commitment" which contributes to the
// hash-coefficient for every key being combined.
func ComboCommit(pubKeys CombinablePubKeySlice) chainhash.Hash {
	// sort the pubkeys, smallest first
	sort.Sort(pubKeys)
	// feed em into the hash
	combo := make([]byte, len(pubKeys)*33)
	for i, k := range pubKeys {
		copy(combo[i*33:(i+1*33)], k.SerializeCompressed())
	}
	return chainhash.HashH(combo)
}

// Combine combines pubkeys into one.
// Never errors; just returns empty pubkeys instead (which will trigger other errors
// because those aren't valid pubkeys)
// Careful to not modify the slice in place.
func (p CombinablePubKeySlice) Combine() *btcec.PublicKey {
	// first, give up if the argument set is empty
	if p == nil || len(p) == 0 {
		return nil
	}

	// make the combo commit.  call it z.
	z := ComboCommit(p)

	// for each pubkey, multiply it by sha256d(z, A)
	// where A is the 33 byte serialized pubkey

	for _, k := range p {
		h := chainhash.HashH(append(z[:], k.SerializeCompressed()...))
		MultiplyPointByHash(k, h)
	}

	// final sum key is called q.
	q := new(btcec.PublicKey)

	// use index i to optimize a bit
	for i, k := range p {
		if i == 0 { // if this is the first key, set instead of adding
			q.X = k.X
			q.Y = k.Y
		} else {
			q.X, q.Y = btcec.S256().Add(q.X, q.Y, k.X, k.Y)
		}
	}
	return q
}

// CombinePrivateKeys takes a set of private keys and combines them in the same way
// as done for public keys.  This only works if you know *all* of the private keys.
// If you don't, we'll do something with returning a scalar coefficient...
// I don't know how that's going to work.  Schnorr stuff isn't decided yet.
func CombinePrivateKeys(keys []*btcec.PrivateKey) *btcec.PrivateKey {

	if keys == nil || len(keys) == 0 {
		return nil
	}
	if len(keys) == 1 {
		return keys[0]
	}
	// bunch of keys
	var pubs CombinablePubKeySlice
	for _, k := range keys {
		pubs = append(pubs, k.PubKey())
	}
	z := ComboCommit(pubs)

	sum := new(big.Int)

	for _, k := range keys {
		h := chainhash.HashH(append(z[:], k.PubKey().SerializeCompressed()...))
		// turn coefficient hash h into a bigint
		hashInt := new(big.Int).SetBytes(h[:])
		// multiply the hash by the private scalar for this particular key
		hashInt.Mul(hashInt, k.D)
		// reduce mod curve N
		hashInt.Mod(hashInt, btcec.S256().N)
		// add this scalar to the aggregate and reduce mod N again
		sum.Add(sum, k.D)
		sum.Mod(sum, btcec.S256().N)
	}

	// kindof ugly that it's converting the bigint to bytes and back but whatever
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), sum.Bytes())

	return priv
}

// ###########################
// HAKD/Elkrem point functions

// CombinePubs takes two 33 byte serialized points, and combines them with
// the deliniearized combination process.  Returns empty array if there's an error.
func CombinePubs(a, b [33]byte) [33]byte {
	var c [33]byte
	apoint, err := btcec.ParsePubKey(a[:], btcec.S256())
	if err != nil {
		return c
	}
	bpoint, err := btcec.ParsePubKey(b[:], btcec.S256())
	if err != nil {
		return c
	}
	cSlice := CombinablePubKeySlice{apoint, bpoint}
	cPub := cSlice.Combine()
	copy(c[:], cPub.SerializeCompressed())
	return c
}

// HashToPub turns a 32 byte hash into a 33 byte serialized pubkey
func PubFromHash(h chainhash.Hash) (p [33]byte) {
	_, pub := btcec.PrivKeyFromBytes(btcec.S256(), h[:])
	copy(p[:], pub.SerializeCompressed())
	return
}
