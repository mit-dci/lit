package lnutil

import (
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

var (
	// invalid pubkey which is not on the curve, Secp256k1
	invalidPubKeyCmpd = [33]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00}

	// pubkey from bitcoin blockchain tx
	// adfa661e05b2221a1425190265f2ab397ff5e00380b33c35b57142575defff14
	//	pubKeyCmpd0 = [33]byte{
	//		0x03, 0x70, 0xed, 0xd2, 0xa2, 0x47, 0x90, 0x76,
	//		0x6d, 0xdc, 0xbc, 0x25, 0x98, 0x00, 0xd6, 0x7e,
	//		0x83, 0x6d, 0x5b, 0xed, 0xd0, 0xa2, 0x74, 0x3c,
	//		0x5f, 0x8c, 0x67, 0xd2, 0x6c, 0x9d, 0x90, 0xf6,
	//		0x35}
	// pubkey from bitcoin blockchain tx
	// 23df526af42b7546987ebe7c2dd712faa146f78c7ae2162b3515fd861c7b045f
	//	pubKeyCmpd1 = [33]byte{
	//		0x03, 0xc1, 0xa3, 0xb5, 0xdc, 0x62, 0x60, 0xff,
	//		0xdd, 0xd0, 0x6c, 0x6e, 0x52, 0x8c, 0x34, 0x40,
	//		0x84, 0xe6, 0x96, 0xb2, 0x11, 0x14, 0x3f, 0xac,
	//		0x15, 0xf6, 0x1f, 0x85, 0xe2, 0x45, 0x89, 0x29,
	//		0x5a}

	// very simple privkey to test
	privKeyB0 = []byte{0x00, 0x01}
	privKeyB1 = []byte{0x00, 0x03}
)

// AddPubsEZ
func TestAddPubsEZ(t *testing.T) {
	// test a normal situation
	// input a: pubKeyCmpd0, 33 bytes array
	// input b: pubKeyCmpd1, 33 bytes array
	// want: wantB33, 33 bytes array
	//
	// inputs a and b are [33]byte and they can not be nil so no test for nil
	var wantB33 = [33]byte{
		0x02, 0x95, 0x82, 0xad, 0x0e, 0xa8, 0xc0, 0xd3,
		0x20, 0x79, 0x96, 0xee, 0x7f, 0x5a, 0x53, 0x28,
		0x71, 0x25, 0xfb, 0x88, 0x4c, 0x14, 0xff, 0xbe,
		0x15, 0x10, 0x13, 0xa7, 0x48, 0x9f, 0x75, 0xd4,
		0x8e}
	if AddPubsEZ(pubKeyCmpd0, pubKeyCmpd1) != wantB33 {
		t.Fatalf("it needs to be equal")
	}

	// test an anomaly situation
	// input a: invalidPubKeyCmpd, 33 bytes array
	// input b: pubKeyCmpd0, 33 bytes array
	// want: invalidPubKeyCmpd, 33 bytes array
	//
	// if HAKD base is invalid pubkey, it just returns invalid HAKD base
	if AddPubsEZ(invalidPubKeyCmpd, pubKeyCmpd0) != invalidPubKeyCmpd {
		t.Fatalf("it needs that output equals to invalid pubkey")
	}

	// test an anomaly situation
	// input a: pubKeyCmpd0, 33 bytes array
	// input b: invalidPubKeyCmpd, 33 bytes array
	// want: not invalidPubKeyCmpd, 33 bytes array
	//
	// if elekpoint is invalid pubkey, it does not return invalid HAKD base
	if AddPubsEZ(pubKeyCmpd0, invalidPubKeyCmpd) == invalidPubKeyCmpd {
		t.Fatalf("it needs that output does not equal to invalid pubkey")
	}
}

// CombinePubs
func TestCombinePubs(t *testing.T) {
	// test a normal situation
	// input a: pubKeyCmpd0, 33 bytes array
	// input b: pubKeyCmpd1, 33 bytes array
	// want: wantB33, 33 bytes array
	//
	// inputs a and b are [33]byte and they can not be nil so no test for nil
	var wantB33 = [33]byte{
		0x03, 0xee, 0xd5, 0x4a, 0x37, 0x98, 0xe8, 0xa8,
		0x2b, 0x35, 0xd4, 0x38, 0x3e, 0x24, 0x68, 0xef,
		0x74, 0x3b, 0xb6, 0x3e, 0x20, 0xb8, 0x81, 0x29,
		0x90, 0xf4, 0x71, 0xf5, 0xd7, 0x35, 0xfb, 0x61,
		0xf4}
	if CombinePubs(pubKeyCmpd0, pubKeyCmpd1) != wantB33 {
		t.Fatalf("it needs to be equal")
	}

	// test an anomaly situation
	// if one of inputs is an invalid pubkey, it returns [33]byte{0x00, 0x00, ..., 0x00}
	//
	// input a: invalidPubKeyCmpd, 33 bytes array
	// input b: pubKeyCmpd0, 33 bytes array
	// want: 33 bytes array, [33]byte{0x00, 0x00, ..., 0x00}
	if CombinePubs(invalidPubKeyCmpd, pubKeyCmpd0) != [33]byte{} {
		t.Fatalf("it needs that output equals to 33 bytes array which has only 0x00")
	}
	// input a: pubKeyCmpd0, 33 bytes array
	// input b: invalidPubKeyCmpd, 33 bytes array
	// want: 33 bytes array, [33]byte{0x00, 0x00, ..., 0x00}
	if CombinePubs(pubKeyCmpd0, invalidPubKeyCmpd) != [33]byte{} {
		t.Fatalf("it needs that output equals to 33 bytes array which has only 0x00")
	}
	// input a: invalidPubKeyCmpd, 33 bytes array
	// input b: invalidPubKeyCmpd, 33 bytes array
	// want: 33 bytes array, [33]byte{0x00, 0x00, ..., 0x00}
	if CombinePubs(invalidPubKeyCmpd, invalidPubKeyCmpd) != [33]byte{} {
		t.Fatalf("it needs that output equals to 33 bytes array which has only 0x00")
	}
}

// Swap(CombinablePubKeySlice method)
// this is obvious but test in case
func TestSwap(t *testing.T) {
	// test for a normal situation
	// input: CombinablePubKeySlice contains
	//   PubKey: pubKeyCmpd0
	//   PubKey: pubKeyCmpd1
	// want: first elm and second elm is swapped
	inPubKey0, _ := btcec.ParsePubKey(pubKeyCmpd0[:], btcec.S256())
	inPubKey1, _ := btcec.ParsePubKey(pubKeyCmpd1[:], btcec.S256())
	var c CombinablePubKeySlice = []*btcec.PublicKey{inPubKey0, inPubKey1}

	c.Swap(0, 1)
	if !c[0].IsEqual(inPubKey1) {
		t.Fatalf("it needs to be equal")
	}
	if !c[1].IsEqual(inPubKey0) {
		t.Fatalf("it needs to be equal")
	}

	// test for an anomaly situation
	// input: CombinablePubKeySlice contains
	//   PubKey: ivalidPubKeyCmpd
	//   PubKey: pubKeyCmpd1
	// want: first elm and second elm is swapped even if elms has nil
	inPubKeyNil, _ := btcec.ParsePubKey(invalidPubKeyCmpd[:], btcec.S256())
	var c0Nil CombinablePubKeySlice = []*btcec.PublicKey{inPubKeyNil, inPubKey1}

	c0Nil.Swap(0, 1)
	if !c0Nil[0].IsEqual(inPubKey1) {
		t.Fatalf("it needs to be equal")
	}
	if c0Nil[1] != inPubKeyNil { // can not use IsEqual because inPubKeyNil is nil
		t.Fatalf("it needs to be equal")
	}
}

// Len(CombinablePubKeySlice method)
// this is obvious but test in case
func TestLen(t *testing.T) {
	// test a normal situation
	// input: CombinablePubKeySlice contains
	//   PubKey: pubKeyCmpd0
	//   PubKey: pubKeyCmpd1
	// want: 2
	inPubKey0, _ := btcec.ParsePubKey(pubKeyCmpd0[:], btcec.S256())
	inPubKey1, _ := btcec.ParsePubKey(pubKeyCmpd1[:], btcec.S256())
	var c CombinablePubKeySlice = []*btcec.PublicKey{inPubKey0, inPubKey1}

	if c.Len() != 2 {
		t.Fatalf("it needs to be equal")
	}

	// test for an anomaly situation
	// input: CombinablePubKeySlice contains
	//   PubKey: ivalidPubKeyCmpd
	//   PubKey: pubKeyCmpd1
	// want: 2 even if elms has nil
	inPubKeyNil, _ := btcec.ParsePubKey(invalidPubKeyCmpd[:], btcec.S256())
	var c0Nil CombinablePubKeySlice = []*btcec.PublicKey{inPubKeyNil, inPubKey1}

	if c0Nil.Len() != 2 {
		t.Fatalf("it needs to be equal")
	}
}

// Less(CombinablePubKeySlice method)
// this is obvious but test in case
func TestLess(t *testing.T) {
	// test a normal situation
	// input: CombinablePubKeySlice contains
	//   PubKey: pubKeyCmpd0
	//   PubKey: pubKeyCmpd1
	inPubKey0, _ := btcec.ParsePubKey(pubKeyCmpd0[:], btcec.S256())
	inPubKey1, _ := btcec.ParsePubKey(pubKeyCmpd1[:], btcec.S256())
	var c CombinablePubKeySlice = []*btcec.PublicKey{inPubKey0, inPubKey1}

	if c.Less(0, 1) != true {
		t.Fatalf("it needs to be equal")
	}
	if c.Less(1, 0) != false {
		t.Fatalf("it needs to be equal")
	}
	if c.Less(0, 0) != false {
		t.Fatalf("it needs to be equal")
	}
	if c.Less(1, 1) != false {
		t.Fatalf("it needs to be equal")
	}

	// test an anomaly situation
	// TODO: fix it
	// panic if one of compared pubkeys is nil

	//inPubKeyNil, _ := btcec.ParsePubKey(invalidPubKeyCmpd[:], btcec.S256())
	//var c0Nil CombinablePubKeySlice = []*btcec.PublicKey{inPubKeyNil, inPubKey1}
	/*
	   if c0Nil.Less(0, 1) != true {
	           t.Fatalf("it needs to be equal")
	   }
	   if c0Nil.Less(1, 0) != false {
	           t.Fatalf("it needs to be equal")
	   }
	   if c0Nil.Less(0, 0) != false {
	           t.Fatalf("it needs to be equal")
	   }
	   if c0Nil.Less(1, 1) != false {
	           t.Fatalf("it needs to be equal")
	   }
	*/
}

// ComboCommit(CombinablePubKeySlice method)
func TestComboCommit(t *testing.T) {
	// test a normal situation
	// input: CombinablePubKeySlice contains
	//   PubKey: pubKeyCmpd0
	//   PubKey: pubKeyCmpd1
	// want: wantH, hcainhash.Hash
	inPubKey0, _ := btcec.ParsePubKey(pubKeyCmpd0[:], btcec.S256())
	inPubKey1, _ := btcec.ParsePubKey(pubKeyCmpd1[:], btcec.S256())
	var c CombinablePubKeySlice = []*btcec.PublicKey{inPubKey0, inPubKey1}

	var wantH chainhash.Hash = [32]byte{
		0x4d, 0x3e, 0x71, 0x71, 0xa1, 0x98, 0x2f, 0x96,
		0x23, 0x5f, 0x7b, 0x6b, 0x39, 0x95, 0x71, 0x5d,
		0x29, 0x43, 0x35, 0x25, 0x16, 0x27, 0x4e, 0x0d,
		0xf0, 0xae, 0xb5, 0x43, 0x60, 0x4a, 0x4d, 0x08}

	if c.ComboCommit() != wantH {
		t.Fatalf("it needs to be equal")
	}

	// test an anomaly situation
	// TODO: fix it
	// panic if one of pubkeys is nil

	//inPubKeyNil, _ := btcec.ParsePubKey(invalidPubKeyCmpd[:], btcec.S256())
	//want := ...
	//var c0Nil CombinablePubKeySlice = []*btcec.PublicKey{inPubKeyNil, inPubKey1}
	//if c0Nil.ComboCommit() != want {
	//	t.Fatalf("it needs to be equal")
	//}
}

// CombinePrivKeyAndSubtract
func TestCombinePrivKeyAndSubtract(t *testing.T) {
	// test a normal situation
	// input1: inPrivKey0, btcec.PrivateKey
	// input2: inPrivKey1, btcec.PrivateKey
	// want: wantB, 32 bytes array
	inPrivKey0, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKeyB0)
	inPrivKeyB1 := privKeyB1
	wantB := [32]byte{
		0xfc, 0x92, 0xf7, 0x65, 0x42, 0xd7, 0x96, 0xb8,
		0x21, 0xba, 0x8d, 0xf6, 0xcd, 0x2a, 0x9e, 0x0a,
		0xe9, 0x72, 0xb7, 0x0f, 0xaf, 0x10, 0x5c, 0xb1,
		0x49, 0xfd, 0x41, 0x9a, 0x94, 0x84, 0x11, 0xf4}
	if CombinePrivKeyAndSubtract(inPrivKey0, inPrivKeyB1) != wantB {
		t.Fatalf("it needs to be equal")
	}
}

// CombinePrivKeyWithBytes
func TestCombinePrivKeyWithBytes(t *testing.T) {
	// test a normal situation
	// input1: inPrivKey0, btcec.PrivateKey
	// input2: inPrivKeyB1, byte slice
	// want: wantPrivKey, btcec.PrivateKey
	inPrivKey0, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKeyB0)
	inPrivKeyB1 := privKeyB1
	wantPrivKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), []byte{
		0xfc, 0x92, 0xf7, 0x65, 0x42, 0xd7, 0x96, 0xb8,
		0x21, 0xba, 0x8d, 0xf6, 0xcd, 0x2a, 0x9e, 0x0a,
		0xe9, 0x72, 0xb7, 0x0f, 0xaf, 0x10, 0x5c, 0xb1,
		0x49, 0xfd, 0x41, 0x9a, 0x94, 0x84, 0x11, 0xf5})
	if CombinePrivKeyWithBytes(inPrivKey0, inPrivKeyB1).D.Cmp(wantPrivKey.D) != 0 {
		t.Fatalf("it needs to be equal")
	}
}

// CombinePrivateKeys
// this can have some parameters but the parameters are not as slice or array.
// it should be already tested as golang itself if too many parameters
func TestCombinePrivateKeys(t *testing.T) {
	// test a normal situation
	// the number of input: 2
	// input1: inPrivKey0, btcec.PrivateKey
	// input2: inPrivKey1, btcec.PrivateKey
	// want: wantPrivKey, btcec.PrivateKey
	inPrivKey0, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKeyB0)
	inPrivKey1, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKeyB1)
	wantPrivKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), []byte{
		0xfc, 0x92, 0xf7, 0x65, 0x42, 0xd7, 0x96, 0xb8,
		0x21, 0xba, 0x8d, 0xf6, 0xcd, 0x2a, 0x9e, 0x0a,
		0xe9, 0x72, 0xb7, 0x0f, 0xaf, 0x10, 0x5c, 0xb1,
		0x49, 0xfd, 0x41, 0x9a, 0x94, 0x84, 0x11, 0xf5})
	if CombinePrivateKeys(inPrivKey0, inPrivKey1).D.Cmp(wantPrivKey.D) != 0 {
		t.Fatalf("it needs to be equal")
	}

	// test a normal situation
	// the number of input: 1
	// input1: inPrivKey0, btcec.PrivateKey
	// want: inPrivKey0, btcec.PrivateKey,
	// it should return input itself if the number of parameters is one
	if CombinePrivateKeys(inPrivKey0) != inPrivKey0 {
		t.Fatalf("it needs to be equal")
	}

	// test an anomaly situation
	// the number of input: 1
	// input1: btcec.PrivateKey but nil
	// want: nil
	var inNil *btcec.PrivateKey
	if CombinePrivateKeys(inNil) != nil {
		t.Fatalf("it needs to be nil")
	}
}

// ElkScalar
func TestElkScalar(t *testing.T) {
	// test a normal situation
	// input: inHash, chainhash.Hash
	// want: wantHash, chainhash.Hash
	var inHash chainhash.Hash = [32]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	var wantHash chainhash.Hash = [32]byte{
		0x49, 0xb8, 0x31, 0x3d, 0xe7, 0xec, 0x03, 0xb1,
		0x21, 0xdc, 0x26, 0xf4, 0x9a, 0x51, 0x6d, 0xb7,
		0x46, 0xde, 0x9b, 0xd5, 0x6e, 0x46, 0x87, 0x4a,
		0xcf, 0xb1, 0xd2, 0xd0, 0xba, 0x72, 0x37, 0xe9}
	if ElkScalar(&inHash) != wantHash {
		t.Fatalf("it needs to be equal")
	}
}

// ElkPointFromHash
func TestElkPointFromHash(t *testing.T) {
	// test a normal situation
	// input: inHash, chainhash.Hash
	// want: wantB33, 33 bytes array
	var inHash chainhash.Hash = [32]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	var wantB33 = [33]byte{
		0x02, 0x8d, 0x73, 0x16, 0x29, 0xe3, 0x3a, 0xad,
		0x54, 0xa4, 0xc1, 0x9b, 0xd7, 0xe3, 0x1a, 0xba,
		0x65, 0xe2, 0xde, 0xba, 0x7b, 0xfc, 0x4b, 0xa2,
		0xc9, 0xb9, 0xb1, 0x71, 0xff, 0xae, 0x42, 0x71,
		0xfd}
	if ElkPointFromHash(&inHash) != wantB33 {
		t.Fatalf("it needs to be equal")
	}
}

// PubFromHash
func TestPubFromHash(t *testing.T) {
	// test a normal situation
	// input: inHash, chainhash.Hash
	// want: wantB33, 33 bytes array
	var inHash chainhash.Hash = [32]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	var wantB33 = [33]byte{
		0x02, 0x79, 0xbe, 0x66, 0x7e, 0xf9, 0xdc, 0xbb,
		0xac, 0x55, 0xa0, 0x62, 0x95, 0xce, 0x87, 0x0b,
		0x07, 0x02, 0x9b, 0xfc, 0xdb, 0x2d, 0xce, 0x28,
		0xd9, 0x59, 0xf2, 0x81, 0x5b, 0x16, 0xf8, 0x17,
		0x98}
	if PubFromHash(inHash) != wantB33 {
		t.Fatalf("it needs to be equal")
	}
}
