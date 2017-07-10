package main

// Compute the predicted signature s*G
import (
	"fmt"
	"math/big"

	"github.com/adiabat/btcd/btcec"
	"github.com/adiabat/btcd/chaincfg/chainhash"
)

var (
	bigZero = new(big.Int).SetInt64(0)
)

// it's just R - h(R,m)A
func SGpredict(curve *btcec.KoblitzCurve,
	msg []byte, Pub, R *btcec.PublicKey) (*btcec.PublicKey, error) {

	// h = Hash(R,m)
	Rxb := R.X.Bytes()
	var hashInput []byte
	hashInput = append(hashInput, Rxb[:]...)
	hashInput = append(hashInput, msg...)
	h := chainhash.HashB(hashInput)

	// h * A
	Pub.X, Pub.Y = curve.ScalarMult(Pub.X, Pub.Y, h)

	// Negate in place
	Pub.Y.Neg(Pub.Y)

	Pub.Y.Mod(Pub.Y, curve.P)

	sG := new(btcec.PublicKey)

	// Pub has been negated; add it to R
	sG.X, sG.Y = curve.Add(R.X, R.Y, Pub.X, Pub.Y)

	return sG, nil
}

// RSign signs with the given k scalar.  Returns s as 32 bytes.
// This is variable time so don't share hardware with enemies.
// This re-calculates R from k, even though we already know R.
// Could be sped up by taking the stored R as an argument.
func RSign(curve *btcec.KoblitzCurve,
	msg []byte, priv [32]byte, k [32]byte) (*big.Int, error) {

	bigPriv := new(big.Int).SetBytes(priv[:])
	bigK := new(big.Int).SetBytes(k[:])

	if bigPriv.Cmp(bigZero) == 0 {
		return nil, fmt.Errorf("secret scalar is zero")
	}
	if bigPriv.Cmp(curve.N) >= 0 {
		return nil, fmt.Errorf("secret scalar is out of bounds")
	}
	if bigK.Cmp(bigZero) == 0 {
		return nil, fmt.Errorf("k scalar is zero")
	}
	if bigK.Cmp(curve.N) >= 0 {
		return nil, fmt.Errorf("k scalar is out of bounds")
	}

	// re-derive R = kG
	var Rx, Ry *big.Int
	Rx, Ry = curve.ScalarBaseMult(k[:])

	// Check if the field element that would be represented by Y is odd.
	// If it is, just keep k in the group order.
	if Ry.Bit(0) == 1 {
		bigK.Mod(bigK, curve.N)
		bigK.Sub(curve.N, bigK)
	}

	// e = Hash(r, m)
	Rxb := Rx.Bytes()
	var hashInput []byte
	hashInput = append(hashInput, Rxb[:]...)
	hashInput = append(hashInput, msg...)
	e := chainhash.HashB(hashInput)
	bigE := new(big.Int).SetBytes(e)

	// If the hash is bigger than N, fail.  Note that N is
	// FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141
	// So this happens about once every 2**128 signatures.
	if bigE.Cmp(curve.N) >= 0 {
		return nil, fmt.Errorf("hash of (R, m) too big")
	}

	// s = k - e*a
	bigS := new(big.Int)
	bigS.Mul(bigE, bigK)
	bigS.Sub(bigK, bigS)
	bigS.Mod(bigS, curve.N)

	// check if s is 0, and fail if it is.  Can't see how this would happen;
	// looks like it would happen about once every 2**256 signatures
	if bigS.Cmp(bigZero) == 0 {
		str := fmt.Errorf("sig s %v is zero", bigS)
		return nil, str
	}

	var empty, s [32]byte
	// Zero out private key and k in array and bigint form
	// who knows if this really helps...  can't hurt though.
	bigK.SetInt64(0)
	k = empty
	bigPriv.SetInt64(0)
	priv = empty

	copy(s[:], bigS.Bytes())

	return bigS, nil
}
