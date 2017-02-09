package lnutil

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

// OutPointsEqual
// test 2 patterns x 2 patterns, a.Hash compared with b.Hash and a.Index compared with b.Index
func TestOutPointsEqual(t *testing.T) {
	var hash1 chainhash.Hash = [32]byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01}
	var hash2 chainhash.Hash = [32]byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x02}

	var u1 uint32 = 1
	var u2 uint32 = 2

	tests := []struct {
		in_a wire.OutPoint
		in_b wire.OutPoint
		want bool
	}{
		//
		// normal situation
		//
		// hash of in_a is not the same as hash of in_b
		// index of in_a is not the same as index of in_b
		{
			wire.OutPoint{hash1, u2}, // in_a
			wire.OutPoint{hash2, u1}, // in_b
			false,
		},

		// hash of in_a is not the same as hash of in_b
		// index of in_a is the same as index of in_b
		{
			wire.OutPoint{hash1, u1}, // in_a
			wire.OutPoint{hash2, u1}, // in_b
			false,
		},

		// hash of in_a is the same as hash of in_b
		// index of in_a is not the same as index of in_b
		{
			wire.OutPoint{hash1, u1}, // in_a
			wire.OutPoint{hash1, u2}, // in_b
			false,
		},

		// hash of in_a is the same as hash of in_b
		// index of in_a is the same as index of in_b
		{
			wire.OutPoint{hash1, u1}, // in_a
			wire.OutPoint{hash1, u1}, // in_b
			true,
		},

		//
		// anomaly situation
		//
		// index of OutPoint is uint32 and the index can not be nil
		// so no test for nil index

		// hash of in_a is just initialized
		{
			wire.OutPoint{[32]byte{}, u1}, // in_a
			wire.OutPoint{hash1, u1},      // in_b
			false,
		},

		// hash of in_b is just initialized
		{
			wire.OutPoint{hash1, u1},      // in_a
			wire.OutPoint{[32]byte{}, u1}, // in_b
			false,
		},

		// hash of in_a is just initialized
		// hash of in_b is just initialized
		{
			wire.OutPoint{[32]byte{}, u1}, // in_a
			wire.OutPoint{[32]byte{}, u1}, // in_b
			true,
		},
	}

	// compare with want
	for i, test := range tests {
		result := OutPointsEqual(test.in_a, test.in_b)
		if test.want != result {
			t.Fatalf("test failed at %d th test", i+1)
			continue
		}
	}
}

// OutPointToBytes
// test some byte arrays
func TestOutPointToBytes(t *testing.T) {
	// test for a normal situation
	// input: [32]bytes array, uint32
	// output: [36]bytes array
	//
	// no problem if input equals to output

	// OutPoint to be an input for OutPointToBytes
	var hash32 chainhash.Hash = [32]byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01}
	var u uint32 = 1
	op := wire.OutPoint{hash32, u}

	// a bytes array to be compared
	b36 := [36]byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x01}

	if OutPointToBytes(op) != b36 {
		t.Fatalf("it needs to be equal")
	}

	// TODO: one more test case
}

// OutPointFromBytes
// test some bytes arrays of 36 byte long
// this depends on a test of NewOutPoint() in btcsuite/btcd/wire/msgtx.go
// and OutPointsEqual() in this file
func TestOutPointFromBytes(t *testing.T) {
	// test for a normal situation
	// input: [36]bytes array
	// output: Outpoint created by wire.NewOutPoint()
	//
	// no problem if input equals to output

	// a bytes array to be an input for OutPointFromBytes
	b36 := [36]byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x01}

	// OutPoint to be compared
	var hash32 chainhash.Hash = [32]byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01}
	var u uint32 = 1
	op := wire.NewOutPoint(&hash32, u)

	// compare an output of OutPointFromBytes() and op
	if !OutPointsEqual(*OutPointFromBytes(b36), *op) {
		t.Fatalf("it needs to be equal")
	}

	// TODO: one more test case
}

// P2WSHify
// test some simple script bytes
func TestP2WSHify(t *testing.T) {
	// test for a normal situation
	// input: []bytes slice, []byte{0x00, 0x01}
	// output: []bytes slice, script bytes
	//
	// no problem if input equals to output

	// a bytes slice to be an input for P2WSHify()
	var inB2 = []byte{0x00, 0x01}

	// script to be compared
	// sha256.Sum256() needs a bytes slice
	sha256InB2 := sha256.Sum256(inB2) // use a different built-in function from fastsha26.Sum256()

	bScript := []byte{txscript.OP_0}
	bScript = append(bScript, txscript.OP_DATA_32) // data for sha256
	for _, b := range sha256InB2 {
		bScript = append(bScript, byte(b))
	}

	// compare an output of P2WSHify() and bScript
	if !bytes.Equal(P2WSHify(inB2), bScript) {
		t.Fatalf("it needs to be equal")
	}

	// TODO: one more test case
}

// DirectWPKHScript
// test some public keys
// this depends on a test of Hash160() in btcsuite/btcutil
func TestDirectWPKHScript(t *testing.T) {
	// test for a normal situation
	// input: [33]bytes array, public key
	// output: []bytes slice, script bytes
	//
	// no problem if input equals to output

	// a bytes array to be an input for P2WSHify()
	var inB33 = [33]byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x01}

	// script to be compared
	bScript := []byte{txscript.OP_0}
	bScript = append(bScript, txscript.OP_DATA_20) // data for hash160

	hash160InB33 := btcutil.Hash160(inB33[:])
	for _, b := range hash160InB33 {
		bScript = append(bScript, byte(b))
	}

	// compare an output of DirectWPKHScript() and bScript
	if !bytes.Equal(DirectWPKHScript(inB33), bScript) {
		t.Fatalf("it needs to be equal")
	}

	// TODO: one more test case
}

// DirectWPKHScriptFromPKH
// test some public key hashes
func TestDirectWPKHScriptFromPKH(t *testing.T) {
	// test for a normal situation
	// input: [20]bytes array, public key hash
	// output: []bytes slice, script bytes
	//
	// no problem if input equals to output

	// a bytes array to be an input for DirectWPKHScriptFromPKH()
	var inB20 = [20]byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01}

	// Script to be compared
	bScript := []byte{txscript.OP_0}
	bScript = append(bScript, txscript.OP_DATA_20) // public key hash
	for _, b := range inB20 {
		bScript = append(bScript, byte(b))
	}

	// compare an output of DirectWPKHScriptFromPKH() and bScript
	if !bytes.Equal(DirectWPKHScriptFromPKH(inB20), bScript) {
		t.Fatalf("it needs to be equal")
	}

	// TODO: one more test case
}
