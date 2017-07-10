package main

import (
	"crypto/hmac"
	"crypto/sha512"

	"github.com/mit-dci/lit/lnutil"
)

// deriveK derives a k scalar from a seed and an identifier string.
// Also returns the R point
// the string should be more structured but other functions can take care of that
// derivation is just sha512 hmac with key = seed private key, data = id string
func deriveK(seed [32]byte, id string) ([32]byte, [33]byte) {
	var k [32]byte
	hm := hmac.New(sha512.New, seed[:])
	copy(k[:], hm.Sum([]byte(id)))

	R := lnutil.PubFromHash(k)
	return k, R
}

//func SignWithR(priv)

//func
