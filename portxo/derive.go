package portxo

import (
	"fmt"
	"math/big"

	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/crypto/koblitz"
)

// DerivePrivateKey returns the private key for a utxo based on a master key
func (kg *KeyGen) DerivePrivateKey(
	m *hdkeychain.ExtendedKey) (*koblitz.PrivateKey, error) {

	var err error
	var empty [32]byte

	if m == nil {
		return nil, fmt.Errorf("nil master key")
	}
	// there might be a private key here, but it can't be "derived"
	if kg.Depth == 0 {
		return nil, fmt.Errorf("no key derivation path")
	}

	currentKey := m
	// this doesn't check for depth > 5, which is kindof invalid...
	for i, step := range kg.Step {
		if uint8(i) == kg.Depth {
			break
		}
		currentKey, err = currentKey.Child(step)
		if err != nil {
			return nil, err
		}
	}

	// get private key from the final derived child key
	derivedPrivKey, err := currentKey.ECPrivKey()
	if err != nil {
		return nil, err
	}

	// if porTxo's private key has something in it, combine that with derived key
	// using the delinearization scheme
	if kg.PrivKey != empty {
		PrivKeyAddBytes(derivedPrivKey, kg.PrivKey[:])
	}

	// done, return derived sum
	return derivedPrivKey, nil
}

// PrivKeyAddBytes adds bytes to a private key.
// NOTE that this modifies the key in place, overwriting it!!!!1
// If k is nil, does nothing and doesn't error (k stays nil)
func PrivKeyAddBytes(k *koblitz.PrivateKey, b []byte) {
	if k == nil {
		return
	}
	// turn arg bytes into a bigint
	arg := new(big.Int).SetBytes(b)
	// add private key to arg
	k.D.Add(k.D, arg)
	// mod 2^256ish
	k.D.Mod(k.D, koblitz.S256().N)
	// new key derived from this sum
	// D is already modified, need to update the pubkey x and y
	k.X, k.Y = koblitz.S256().ScalarBaseMult(k.D.Bytes())
	return
}
