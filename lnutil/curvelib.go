package lnutil

import (
	"math/big"

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
