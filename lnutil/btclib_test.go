package lnutil

import (
	"bytes"
	"testing"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/wire"
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
		inA  wire.OutPoint
		inB  wire.OutPoint
		want bool
	}{
		//
		// normal situation
		//
		// hash of inA is not the same as hash of inB
		// index of inA is not the same as index of inB
		{
			wire.OutPoint{Hash: hash1, Index: u2}, // inA
			wire.OutPoint{Hash: hash2, Index: u1}, // inB
			false,
		},

		// hash of inA is not the same as hash of inB
		// index of inA is the same as index of inB
		{
			wire.OutPoint{Hash: hash1, Index: u1}, // inA
			wire.OutPoint{Hash: hash2, Index: u1}, // inB
			false,
		},

		// hash of inA is the same as hash of inB
		// index of inA is not the same as index of inB
		{
			wire.OutPoint{Hash: hash1, Index: u1}, // inA
			wire.OutPoint{Hash: hash1, Index: u2}, // inB
			false,
		},

		// hash of inA is the same as hash of inB
		// index of inA is the same as index of inB
		{
			wire.OutPoint{Hash: hash1, Index: u1}, // inA
			wire.OutPoint{Hash: hash1, Index: u1}, // inB
			true,
		},

		//
		// anomaly situation
		//
		// index of OutPoint is uint32 and the index can not be nil
		// so no test for nil index
		// inA and inB can not be nil so test just initialized one

		// hash of inA is just initialized
		{
			wire.OutPoint{Hash: [32]byte{}, Index: u1}, // inA
			wire.OutPoint{Hash: hash1, Index: u1},      // inB
			false,
		},

		// hash of inB is just initialized
		{
			wire.OutPoint{Hash: hash1, Index: u1},      // inA
			wire.OutPoint{Hash: [32]byte{}, Index: u1}, // inB
			false,
		},

		// hash of inA is just initialized
		// hash of inB is just initialized
		{
			wire.OutPoint{Hash: [32]byte{}, Index: u1}, // inA
			wire.OutPoint{Hash: [32]byte{}, Index: u1}, // inB
			true,
		},
	}

	// compare with want
	for i, test := range tests {
		result := OutPointsEqual(test.inA, test.inB)
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
	op := wire.OutPoint{Hash: hash32, Index: u}

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
// this depends on a test of NewOutPoint() in adiabat/btcd/wire/msgtx.go
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
	// test for a normal situation(blackbox test)
	// input: inB2, []bytes slice, []byte{0x00, 0x01}
	// want: wantScriptB, []bytes slice, script bytes

	// a bytes slice to be an input for P2WSHify()
	var inB2 = []byte{0x00, 0x01}

	// script to be compared
	wantScriptB := []byte{0x00, 0x20, 0xb4, 0x13, 0xf4, 0x7d, 0x13, 0xee,
		0x2f, 0xe6, 0xc8, 0x45, 0xb2, 0xee, 0x14, 0x1a,
		0xf8, 0x1d, 0xe8, 0x58, 0xdf, 0x4e, 0xc5, 0x49,
		0xa5, 0x8b, 0x79, 0x70, 0xbb, 0x96, 0x64, 0x5b,
		0xc8, 0xd2}

	if !bytes.Equal(P2WSHify(inB2), wantScriptB) {
		t.Fatalf("it needs to be equal")
	}

	// TODO: one more test case
}

// DirectWPKHScript
// test some public keys
// this depends on a test of Hash160() in adiabat/btcutil
func TestDirectWPKHScript(t *testing.T) {
	// test for a normal situation(blackbox test)
	// input: inB33, [33]bytes array, public key
	// want: wantScriptB, []bytes slice, script bytes

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
	wantScriptB := []byte{0x00, 0x14, 0x4c, 0x20, 0xbb, 0x22, 0xf2, 0x88,
		0xee, 0x1f, 0xae, 0xc2, 0x66, 0xdf, 0x21, 0x52,
		0x0e, 0x4c, 0x13, 0xc7, 0x25, 0x6c}

	if !bytes.Equal(DirectWPKHScript(inB33), wantScriptB) {
		t.Fatalf("it needs to be equal")
	}

	// TODO: one more test case
}

// DirectWPKHScriptFromPKH
// test some public key hashes
func TestDirectWPKHScriptFromPKH(t *testing.T) {
	// test for a normal situation(blackbox test)
	// input: inB20, [20]bytes array, public key hash
	// want: wantScriptB, []bytes slice, script bytes

	// a bytes array to be an input for DirectWPKHScriptFromPKH()
	var inB20 = [20]byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01}

	// Script to be compared
	wantScriptB := []byte{0x00, 0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00}

	if !bytes.Equal(DirectWPKHScriptFromPKH(inB20), wantScriptB) {
		t.Fatalf("it needs to be equal")
	}

	// TODO: one more test case
}
