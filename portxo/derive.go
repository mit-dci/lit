package portxo

import (
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/hdkeychain"
)

// DerivePrivateKey returns the private key for a utxo based on a master key
func (kg *KeyGen) DerivePrivateKey(
	m *hdkeychain.ExtendedKey) (*btcec.PrivateKey, error) {

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

	// if porTxo's private key has something in it, add that to the derived key
	if kg.PrivKey != empty { // actually could work without this line :)
		privKeyAddArrayInPlace(derivedPrivKey, kg.PrivKey)
	}

	// done, return derived sum
	return derivedPrivKey, nil
}

// privKeyAddArrayInPlace adds a 32 byte array to an existing private key
// the private key is modified in place
func privKeyAddArrayInPlace(k *btcec.PrivateKey, b [32]byte) {
	// give up if nil privkey
	if k == nil {
		return
	}
	// turn arg byte array into a bigint
	arr := new(big.Int).SetBytes(b[:])
	// add private key to arr
	k.D.Add(k.D, arr)
	// mod 2^256ish
	k.D.Mod(k.D, btcec.S256().N)
	// new key derived from this sum
	// D is already modified, need to update the pubkey x and y
	// this is the slow part and not actually needed for hardened derivation.
	// could probably optimize and skip this EC part
	k.X, k.Y = btcec.S256().ScalarBaseMult(k.D.Bytes())
	return
}
