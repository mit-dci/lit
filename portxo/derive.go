package portxo

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/mit-dci/lit/lnutil"
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

	// if porTxo's private key has something in it, combine that with derived key
	// using the delinearization scheme
	if kg.PrivKey != empty {
		derivedPrivKey = lnutil.PrivKeyCombineBytes(derivedPrivKey, kg.PrivKey[:])
	}

	// done, return derived sum
	return derivedPrivKey, nil
}
