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

// it's just sG = R - h(R,m)A
func SGpredict(Pub [33]byte, msg, R [32]byte) (*btcec.PublicKey, error) {

	// Hardcode curve
	curve := btcec.S256()

	A, err := btcec.ParsePubKey(Pub[:], curve)
	if err != nil {
		return nil, err
	}

	RPoint, err := btcec.ParsePubKey(append([]byte{0x02}, R[:]...), curve)
	if err != nil {
		return nil, err
	}

	// Ry is always even.  So always 0x02 starting byte.

	// e = Hash(R,m)
	var hashInput []byte
	hashInput = append(R[:], msg[:]...)
	e := chainhash.HashB(hashInput)

	bigE := new(big.Int).SetBytes(e)

	if bigE.Cmp(curve.N) >= 0 {
		return nil, fmt.Errorf("hash of (R, m) too big")
	}

	//	fmt.Printf("e: %x\n", e)

	// e * A
	A.X, A.Y = curve.ScalarMult(A.X, A.Y, e)

	//	fmt.Printf("1eA(x): %s\teA(y): %s\n", A.X.String(), A.Y.String())

	//	 Negate in place
	A.Y.Neg(A.Y)

	//	fmt.Printf("2eA(x): %s\teA(y): %s\n", A.X.String(), A.Y.String())

	//	fmt.Printf("3eA(x): %s\teA(y): %s\n", A.X.String(), A.Y.String())

	sG := new(btcec.PublicKey)

	// add to R
	sG.X, sG.Y = curve.Add(A.X, A.Y, RPoint.X, RPoint.Y)

	//	fmt.Printf("4eA(x): %s\teA(y): %s\n", sG.X.String(), sG.Y.String())

	return sG, nil
}

// R is 32 bytes and it's y-coordinate is always even.
// Derive R from k.
func KtoR(k [32]byte) [32]byte {

	// Hardcode curve
	curve := btcec.S256()

	Rx, _ := curve.ScalarBaseMult(k[:])

	// Ry is always even.

	var R [32]byte
	copy(R[:], Rx.Bytes())

	return R
}

// RSign signs with the given k scalar.  Returns s as 32 bytes.
// This is variable time so don't share hardware with enemies.
// This re-calculates R from k, even though we already know R.
// Could be sped up by taking the stored R as an argument.
func RSign(msg, priv, k [32]byte) ([32]byte, error) {

	var empty, s [32]byte

	// Hardcode curve
	curve := btcec.S256()

	bigPriv := new(big.Int).SetBytes(priv[:])
	priv = empty
	bigK := new(big.Int).SetBytes(k[:])

	if bigPriv.Cmp(bigZero) == 0 {
		return empty, fmt.Errorf("priv scalar is zero")
	}
	if bigPriv.Cmp(curve.N) >= 0 {
		return empty, fmt.Errorf("priv scalar is out of bounds")
	}
	if bigK.Cmp(bigZero) == 0 {
		return empty, fmt.Errorf("k scalar is zero")
	}
	if bigK.Cmp(curve.N) >= 0 {
		return empty, fmt.Errorf("k scalar is out of bounds")
	}

	// re-derive R = kG
	var Rx, Ry *big.Int
	Rx, Ry = curve.ScalarBaseMult(k[:])

	// Ry is always even.  Make it even if it's not.
	if Ry.Bit(0) == 1 {
		bigK.Mod(bigK, curve.N)
		bigK.Sub(curve.N, bigK)
	}

	// e = Hash(r, m)
	Rxb := Rx.Bytes()
	var hashInput []byte
	hashInput = append(Rxb[:], msg[:]...)
	e := chainhash.HashB(hashInput)
	bigE := new(big.Int).SetBytes(e)

	// If the hash is bigger than N, fail.  Note that N is
	// FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141
	// So this happens about once every 2**128 signatures.
	if bigE.Cmp(curve.N) >= 0 {
		return empty, fmt.Errorf("hash of (R, m) too big")
	}
	//	fmt.Printf("e: %x\n", e)
	// s = k + e*a
	bigS := new(big.Int)
	// e*a
	bigS.Mul(bigE, bigPriv)
	// k + (e*a)
	bigS.Add(bigK, bigS)
	bigS.Mod(bigS, curve.N)

	// check if s is 0, and fail if it is.  Can't see how this would happen;
	// looks like it would happen about once every 2**256 signatures
	if bigS.Cmp(bigZero) == 0 {
		str := fmt.Errorf("sig s %v is zero", bigS)
		return empty, str
	}

	// Zero out private key and k in array and bigint form
	// who knows if this really helps...  can't hurt though.
	bigK.SetInt64(0)
	k = empty
	bigPriv.SetInt64(0)

	copy(s[:], bigS.Bytes())

	return s, nil
}
