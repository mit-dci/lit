package uspv

import (
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"li.lan/tx/lit/lnutil"
	"li.lan/tx/lit/portxo"
)

/*
Key derivation for a TxStore has 3 levels: use case, peer index, and keyindex.
Regular wallet addresses are use 0, peer 0, and then a linear index.
The identity key is use 11, peer 0, index 0.
Channel multisig keys are use 2, peer and index per peer and channel.
Channel refund keys are use 3, peer and index per peer / channel.
*/

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

// multiplies a pubkey point by a scalar
func PubKeyMultBytes(k *btcec.PublicKey, n uint32) {
	b := lnutil.U32tB(n)
	k.X, k.Y = btcec.S256().ScalarMult(k.X, k.Y, b)
}

// multiply the private key by a coefficient
func PrivKeyMult(k *btcec.PrivateKey, n uint32) {
	bigN := new(big.Int).SetUint64(uint64(n))
	k.D.Mul(k.D, bigN)
	k.D.Mod(k.D, btcec.S256().N)
	k.X, k.Y = btcec.S256().ScalarBaseMult(k.D.Bytes())
}

// IDPrivAdd returns a channel pubkey from the sum of two scalars
func IDPrivAdd(idPriv, ds *btcec.PrivateKey) *btcec.PrivateKey {
	cPriv := new(btcec.PrivateKey)
	cPriv.Curve = btcec.S256()

	cPriv.D.Add(idPriv.D, ds.D)
	cPriv.D.Mod(cPriv.D, btcec.S256().N)

	cPriv.X, cPriv.Y = btcec.S256().ScalarBaseMult(cPriv.D.Bytes())
	return cPriv
}

func IdToPub(idArr [32]byte) (*btcec.PublicKey, error) {
	// IDs always start with 02
	return btcec.ParsePubKey(append([]byte{0x02}, idArr[:]...), btcec.S256())
}

// =====================================================================
// OK only use these now

// PathPrivkey returns a private key by descending the given path
// Returns nil if there's an error.
func (t *TxStore) PathPrivkey(kg portxo.KeyGen) *btcec.PrivateKey {
	// in uspv, we require path depth of 5
	if kg.Depth != 5 {
		return nil
	}
	priv, err := kg.DerivePrivateKey(t.rootPrivKey)
	if err != nil {
		fmt.Printf("PathPrivkey err %s", err.Error())
		return nil
	}
	return priv
}

// PathPubkey returns a public key by descending the given path.
// Returns nil if there's an error.
func (t *TxStore) PathPubkey(kg portxo.KeyGen) *btcec.PublicKey {
	priv := t.PathPrivkey(kg)
	if priv == nil {
		return nil
	}
	return t.PathPrivkey(kg).PubKey()
}

// PathPubHash160 returns a 20 byte pubkey hash for the given path
// It'll always return 20 bytes, or a nil if there's an error.
func (t *TxStore) PathPubHash160(kg portxo.KeyGen) []byte {
	pub := t.PathPubkey(kg)
	if pub == nil {
		return nil
	}
	return btcutil.Hash160(pub.SerializeCompressed())
}

// ------------- end of 2 main key deriv functions

// get a private key from the regular wallet
func (t *TxStore) GetWalletPrivkey(idx uint32) *btcec.PrivateKey {
	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = 0 | 1<<31
	kg.Step[2] = 0 | 1<<31
	kg.Step[3] = 0 | 1<<31
	kg.Step[4] = idx | 1<<31
	return t.PathPrivkey(kg)
}

// GetWalletKeygen returns the keygen for a standard wallet address
func GetWalletKeygen(idx uint32) portxo.KeyGen {
	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = 0 | 1<<31
	kg.Step[2] = 0 | 1<<31
	kg.Step[3] = 0 | 1<<31
	kg.Step[4] = idx | 1<<31
	return kg
}

// get a public key from the regular wallet
func (t *TxStore) GetWalletAddress(idx uint32) *btcutil.AddressWitnessPubKeyHash {
	if t == nil {
		fmt.Printf("GetAddress %d nil txstore\n", idx)
		return nil
	}
	priv := t.GetWalletPrivkey(idx)
	if priv == nil {
		fmt.Printf("GetAddress %d made nil pub\n", idx)
		return nil
	}
	adr, err := btcutil.NewAddressWitnessPubKeyHash(
		btcutil.Hash160(priv.PubKey().SerializeCompressed()), t.Param)
	if err != nil {
		fmt.Printf("GetAddress %d made nil pub\n", idx)
		return nil
	}
	return adr
}

// GetUsePrive generates a private key for the given use case & keypath
func (t *TxStore) GetUsePriv(kg portxo.KeyGen, use uint32) *btcec.PrivateKey {
	kg.Step[2] = use
	return t.PathPrivkey(kg)
}

// GetUsePub generates a pubkey for the given use case & keypath
func (t *TxStore) GetUsePub(kg portxo.KeyGen, use uint32) [33]byte {
	var b [33]byte
	pub := t.GetUsePriv(kg, use).PubKey()
	if pub != nil {
		copy(b[:], pub.SerializeCompressed())
	}
	return b
}

// IdKey returns the identity private key
func (t *TxStore) IdKey() *btcec.PrivateKey {
	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = 0 | 1<<31
	kg.Step[2] = 9 | 1<<31
	kg.Step[3] = 0 | 1<<31
	kg.Step[4] = 0 | 1<<31
	return t.PathPrivkey(kg)
}
