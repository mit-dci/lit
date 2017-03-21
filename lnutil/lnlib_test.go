package lnutil

import (
	"bytes"
	"testing"
)

var (
	// pubkey from bitcoin blockchain tx
	// adfa661e05b2221a1425190265f2ab397ff5e00380b33c35b57142575defff14
	pubKeyCmpd0 = [33]byte{
		0x03, 0x70, 0xed, 0xd2, 0xa2, 0x47, 0x90, 0x76,
		0x6d, 0xdc, 0xbc, 0x25, 0x98, 0x00, 0xd6, 0x7e,
		0x83, 0x6d, 0x5b, 0xed, 0xd0, 0xa2, 0x74, 0x3c,
		0x5f, 0x8c, 0x67, 0xd2, 0x6c, 0x9d, 0x90, 0xf6,
		0x35}
	// pubkey from bitcoin blockchain tx
	// 23df526af42b7546987ebe7c2dd712faa146f78c7ae2162b3515fd861c7b045f
	pubKeyCmpd1 = [33]byte{
		0x03, 0xc1, 0xa3, 0xb5, 0xdc, 0x62, 0x60, 0xff,
		0xdd, 0xd0, 0x6c, 0x6e, 0x52, 0x8c, 0x34, 0x40,
		0x84, 0xe6, 0x96, 0xb2, 0x11, 0x14, 0x3f, 0xac,
		0x15, 0xf6, 0x1f, 0x85, 0xe2, 0x45, 0x89, 0x29,
		0x5a}
)

// CommitScript
func TestScriptInCommitScript(t *testing.T) {
	// test for a normal situation(blackbox test)
	// input: inRKey, revocable key
	// input: inTKey, timeout key
	// input: inDelay
	// want: wantB, byte slice
	//
	// blackbox test
	inRKey := [33]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x01}
	inTKey := [33]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x02}
	var inDelay uint16 = 2

	wantB := []byte{0x63, 0x21}
	wantB = append(wantB, inRKey[:]...)
	wantB = append(wantB, 0x67)
	wantB = append(wantB, 0x52) // it is related to inDelay
	wantB = append(wantB, []byte{0xb2, 0x75, 0x21}...)
	wantB = append(wantB, inTKey[:]...)
	wantB = append(wantB, []byte{0x68, 0xac}...)

	if !bytes.Equal(CommitScript(inRKey, inTKey, inDelay), wantB) {
		t.Fatalf("it needs to be equal")
	}
}

// FundTxScript
func TestFundTxScript(t *testing.T) {
	// test for a normal situation
	// input1: pubKeyCmpd0, 33 bytes array
	// input2: pubKeyCmpd1, 33 bytes array
	//
	// test for swapped value
	// this tests three patterns, normal-order, inverse-order and the same
	_, swappedTrue, _ := FundTxScript(pubKeyCmpd0, pubKeyCmpd1)
	if swappedTrue != true {
		t.Fatalf("wrong swapped value")
	}

	_, swappedFalse, _ := FundTxScript(pubKeyCmpd1, pubKeyCmpd0)
	if swappedFalse != false {
		t.Fatalf("wrong swapped value")
	}

	_, swappedSame, _ := FundTxScript(pubKeyCmpd0, pubKeyCmpd0)
	if swappedSame != false {
		t.Fatalf("wrong swapped value")
	}

	// test for a normal situation(blackbox test)
	// input1: pubKeyCmpd0, 33 bytes array
	// input2: pubKeyCmpd1, 33 bytes array
	// want: wantB, byte slice
	script, _, _ := FundTxScript(pubKeyCmpd0, pubKeyCmpd1)

	wantB := []byte{0x52, 0x21}
	wantB = append(wantB, pubKeyCmpd1[:]...)
	wantB = append(wantB, 0x21)
	wantB = append(wantB, pubKeyCmpd0[:]...)
	wantB = append(wantB, []byte{0x52, 0xae}...)

	if !bytes.Equal(script, wantB) {
		t.Fatalf("it needs to be equal")
	}
}

// FundTxOut
func TestFundTxOut(t *testing.T) {
	// test for a normal situation(blackbox test)
	// input1: pubKeyCmpd0, 33 bytes array
	// input2: pubKeyCmpd1, 33 bytes array
	// input3: inI, int64
	// want1: wantI, int64
	// want2: wantB, byte slice
	var inI int64 = 2
	txOut, _ := FundTxOut(pubKeyCmpd0, pubKeyCmpd1, inI)

	var wantI int64 = 2
	wantB := []byte{0x00, 0x20, 0x98, 0x5e, 0x92, 0x74, 0x25, 0x91,
		0x16, 0xf0, 0x26, 0x7a, 0x69, 0x96, 0x84, 0x26,
		0x7a, 0x6e, 0x26, 0xb8, 0xc0, 0x9b, 0x73, 0xd8,
		0x6f, 0x37, 0x60, 0x45, 0xec, 0x6f, 0xe2, 0xd6,
		0x03, 0xda}

	if txOut.Value != wantI {
		t.Fatalf("it needs to be equal")
	}
	if !bytes.Equal(txOut.PkScript, wantB) {
		t.Fatalf("it needs to be equal")
	}

	// test for an anomaly situation
	// input1: pubKeyCmpd0, 33 bytes array
	// input2: pubKeyCmpd1, 33 bytes array
	// input3: inIMinus, int64
	// want: txOut is nil, err is not nil
	//
	// test for amt in case that amt is minus
	var inIMinus int64 = -2
	txOutNil, errNotNil := FundTxOut(pubKeyCmpd0, pubKeyCmpd1, inIMinus)

	if txOutNil != nil {
		t.Fatalf("it needs to be nil")
	}
	if errNotNil == nil {
		t.Fatalf("it needs not to be nil")
	}

}
