package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"math/big"

	"github.com/adiabat/btcd/btcec"
)

// deriveK derives a k scalar from a seed and an identifier string.
// Also returns the R point
// the string should be more structured but other functions can take care of that
// derivation is just sha512 hmac with key = seed private key, data = id string
func deriveK(seed [32]byte, id string) ([32]byte, [33]byte) {
	// Hardcode curve
	curve := btcec.S256()

	var k [32]byte
	var R [33]byte

	Rpub := new(btcec.PublicKey)
	Rpub.Curve = curve

	hm := hmac.New(sha512.New, seed[:])
	hm.Write([]byte(id))
	copy(k[:], hm.Sum(nil))

	bigK := new(big.Int).SetBytes(k[:])
	// derive R = kG
	Rpub.X, Rpub.Y = curve.ScalarBaseMult(k[:])

	// If R's y-coordinate is odd, modify k.  But not R!
	// TODO figure out why this is a thing
	if Rpub.Y.Bit(0) == 1 {
		bigK.Mod(bigK, curve.N)
		bigK.Sub(curve.N, bigK)
	}

	copy(R[:], Rpub.SerializeCompressed())
	//	R := lnutil.PubFromHash(k)
	return k, R
}

//func SignWithR(priv)

//func
