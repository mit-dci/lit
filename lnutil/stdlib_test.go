package lnutil

import (
	"bytes"
	"testing"
)

// I32tB
//
// Test three patterns, zero, max and min for int32
// These are obvious but test in case
func TestI32tB(t *testing.T) {
	// if i is nil, it occurs an error
	// when execution because a type of i is not int32

	// test for a normal situation
	// input: zero(int32)
	// output: 4 bytes array, []byte{0x00, 0x00, 0x00, 0x00}
	//
	// no problem if input equals to output
	var zeroI32 int32 = 0
	zeroB32 := []byte{0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(I32tB(zeroI32), zeroB32) {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: max(int32)
	// output: 4 bytes array, []byte{0x7f, 0xff, 0xff, 0xff}
	//
	// no problem if input equals to output
	var plusMaxI32 int32 = 2147483647
	plusMaxB32 := []byte{0x7f, 0xff, 0xff, 0xff}
	if !bytes.Equal(I32tB(plusMaxI32), plusMaxB32) {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: min(int32)
	// output: 4 bytes array, []byte{0x80, 0x00, 0x00, 0x00}
	//
	// no problem if input equals to output
	var minusMaxI32 int32 = -2147483648
	minusMaxB32 := []byte{0x80, 0x00, 0x00, 0x00}
	if !bytes.Equal(I32tB(minusMaxI32), minusMaxB32) {
		t.Fatalf("it needs that input equals to output")
	}
}

// U32tB
//
// Test three patterns, zero and max for uint32
// These are obvious but test in case
func TestU32tB(t *testing.T) {
	// if i is nil, it occurs an error
	// when execution because a type of i is not uint32

	// test for a normal situation
	// input: zero(uint32)
	// output: 4 bytes array, []byte{0x00, 0x00, 0x00, 0x00}
	//
	// no problem if input equals to output
	var zeroU32 uint32 = 0
	zeroB32 := []byte{0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(U32tB(zeroU32), zeroB32) {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: max(uint32)
	// output: 4 bytes array, []byte{0xff, 0xff, 0xff, 0xff}
	//
	// no problem if input equals to output
	var maxU32 uint32 = 4294967295
	maxB32 := []byte{0xff, 0xff, 0xff, 0xff}
	if !bytes.Equal(U32tB(maxU32), maxB32) {
		t.Fatalf("it needs that input equals to output")
	}
}

// BtU32
//
// BtU32 has an input of a byte array for arbitrary length(the length is a int type)
// If the length of the input is not 4, it returns fixed value.
// This function tests the length
func TestLenBInBtU32(t *testing.T) {
	// if b is nil, it occurs an error
	// when execution because a type of b is not []byte

	// test for a normal situation
	// input: 4 bytes array, []byte{0x00, 0x00, 0x00, 0x00}
	// output: 0xffffffff
	//
	// no problem if input unequals to output
	zeroB32 := []byte{0x00, 0x00, 0x00, 0x00}
	if BtU32(zeroB32) == 0xffffffff {
		t.Fatalf("it needs that input unequals to output")
	}

	// test for an anomaly situation
	// input: 5 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00}
	// output: 0xffffffff
	//
	// no problem if input equals to output
	zeroB40 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00}
	if BtU32(zeroB40) != 0xffffffff {
		t.Fatalf("it needs that input equals to output")
	}

	// test for an anomaly situation
	// input: 3 bytes array, []byte{0x00, 0x00, 0x00}
	// output: 0xffffffff
	//
	// no problem if input equals to output
	zeroB24 := []byte{0x00, 0x00, 0x00}
	if BtU32(zeroB24) != 0xffffffff {
		t.Fatalf("it needs that input equals to output")
	}

	// TODO: test a large size bytes arary
	//zeroB18446744073709551615 := []byte{0x00 * 18446744073709551615}
	//if BtU32(zeroB18446744073709551615) != 0xffffffff {
	//	t.Fatalf("it needs that input equals to output")
	//}

	// test for an anomaly situation
	// input: empty bytes array
	// output: 0xffffffff
	//
	// no problem if input equals to output
	emptyB0 := []byte{}
	if BtU32(emptyB0) != 0xffffffff {
		t.Fatalf("it needs that input equals to output")
	}
}

// BtU32
//
// Test three patterns, zero and max for uint32
// These are obvious but test in case
func TestBtU32(t *testing.T) {
	// if b is nil, it occurs an error
	// when execution because a type of b is not []byte

	// test for a normal situation
	// input: 4 bytes array, []byte{0x00, 0x00, 0x00, 0x00}
	// output: zero(uint32)
	//
	// no problem if input equals to output
	var zeroU32 uint32 = 0
	zeroB32 := []byte{0x00, 0x00, 0x00, 0x00}
	if BtU32(zeroB32) != zeroU32 {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: 4 bytes array, []byte{0xff, 0xff, 0xff, 0xff}
	// output: max(uint32)
	//
	// no problem if input equals to output
	var maxU32 uint32 = 4294967295
	maxB32 := []byte{0xff, 0xff, 0xff, 0xff}
	if BtU32(maxB32) != maxU32 {
		t.Fatalf("it needs that input equals to output")
	}
}

// BtI32
//
// BtI32 has an input of a byte array for arbitrary length(the length is a int type)
// If the length of the input is not 4, it returns fixed value.
// This function tests the length
func TestLenBInBtI32(t *testing.T) {
	// if b is nil, it occurs an error
	// when execution because a type of b is not []byte

	// test for a normal situation
	// input: 4 bytes array, []byte{0x00, 0x00, 0x00, 0x00}
	// output: 0x7fffffff
	//
	// no problem if input unequals to output
	zeroB32 := []byte{0x00, 0x00, 0x00, 0x00}
	if BtI32(zeroB32) == 0x7fffffff {
		t.Fatalf("it needs that input unequals to output")
	}

	// test for a normal situation
	// input: 5 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00}
	// output: 0x7fffffff
	//
	// no problem if input equals to output
	zeroB40 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00}
	if BtI32(zeroB40) != 0x7fffffff {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: 3 bytes array, []byte{0x00, 0x00, 0x00}
	// output: 0x7fffffff
	//
	// no problem if input equals to output
	zeroB24 := []byte{0x00, 0x00, 0x00}
	if BtI32(zeroB24) != 0x7fffffff {
		t.Fatalf("it needs that input equals to output")
	}

	// TODO: test a large size bytes arary
	//zeroB18446744073709551615 := []byte{0x00 * 18446744073709551615}
	//if BtI32(zeroB18446744073709551615) != 0x7fffffff {
	//	t.Fatalf("it needs that input equals to output")
	//}

	// test for an anomaly situation
	// input: empty bytes array
	// output: 0x7fffffff
	//
	// no problem if input equals to output
	emptyB0 := []byte{}
	if BtI32(emptyB0) != 0x7fffffff {
		t.Fatalf("it needs that input equals to output")
	}
}

// BtI32
//
// Test three patterns, zero, max and min for int32
// These are obvious but test in case
func TestBtI32(t *testing.T) {
	// if b is nil, it occurs an error
	// when execution because a type of b is not []byte

	// test for a normal situation
	// input: 4 bytes array, []byte{0x00, 0x00, 0x00, 0x00}
	// output: zero(int32)
	//
	// no problem if input equals to output
	var zeroI32 int32 = 0
	zeroB32 := []byte{0x00, 0x00, 0x00, 0x00}
	if BtI32(zeroB32) != zeroI32 {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: 4 bytes array, []byte{0x7f, 0xff, 0xff, 0xff}
	// output: max(int32)
	//
	// no problem if input equals to output
	var plusMaxI32 int32 = 2147483647
	plusMaxB32 := []byte{0x7f, 0xff, 0xff, 0xff}
	if BtI32(plusMaxB32) != plusMaxI32 {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: 4 bytes array, []byte{0x80, 0x00, 0x00, 0x00}
	// output: min(int32)
	//
	// no problem if input equals to output
	var minusMaxI32 int32 = -2147483648
	minusMaxB32 := []byte{0x80, 0x00, 0x00, 0x00}
	if BtI32(minusMaxB32) != minusMaxI32 {
		t.Fatalf("it needs that input equals to output")
	}
}

// I64tB
//
// Test three patterns, zero, max and min for int64
// These are obvious but test in case
func TestI64tB(t *testing.T) {
	// if i is nil, it occurs an error
	// when execution because a type of i is not int64

	// test for a normal situation
	// input: zero(int64)
	// output: 8 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	//
	// no problem if input equals to output
	var zeroI64 int64 = 0
	zeroB64 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(I64tB(zeroI64), zeroB64) {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: max(int64)
	// output: 8 bytes array, []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	//
	// no problem if input equals to output
	var plusMaxI64 int64 = 9223372036854775807
	plusMaxB64 := []byte{0x7f, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff}
	if !bytes.Equal(I64tB(plusMaxI64), plusMaxB64) {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: min(int64)
	// output: 8 bytes array, []byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	//
	// no problem if input equals to output
	var minusMaxI64 int64 = -9223372036854775808
	minusMaxB64 := []byte{0x80, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(I64tB(minusMaxI64), minusMaxB64) {
		t.Fatalf("it needs that input equals to output")
	}
}

// U64tB
//
// Test three patterns, zero, max and min for uint64
// These are obvious but test in case
func TestU64tB(t *testing.T) {
	// if i is nil, it occurs an error
	// when execution because a type of i is not uint64

	// test for a normal situation
	// input: zero(uint64)
	// output: 8 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	//
	// no problem if input equals to output
	var zeroU64 uint64 = 0
	zeroB64 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(U64tB(zeroU64), zeroB64) {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: max(uint64)
	// output: 8 bytes array, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	//
	// no problem if input equals to output
	var plusMaxU64 uint64 = 18446744073709551615
	plusMaxB64 := []byte{0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff}
	if !bytes.Equal(U64tB(plusMaxU64), plusMaxB64) {
		t.Fatalf("it needs that input equals to output")
	}
}

// BtI64
//
// BtI64 has an input of a byte array for arbitrary length(the length is a int type)
// If the length of the input is not 8, it returns fixed value.
// This function tests the length
func TestLenBInBtI64(t *testing.T) {
	// if b is nil, it occurs an error
	// when execution because a type of b is not []byte

	// test for a normal situation
	// input: 8 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// output: 0x7fffffffffffffff
	//
	// no problem if input unequals to output
	zeroB64 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}
	if BtI64(zeroB64) == 0x7fffffffffffffff {
		t.Fatalf("it needs that input unequals to output")
	}

	// test for an anomaly situation
	// input: 9 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// output: 0x7fffffffffffffff
	//
	// no problem if input unequals to output
	zeroB72 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00}
	if BtI64(zeroB72) != 0x7fffffffffffffff {
		t.Fatalf("it needs that input equals to output")
	}

	// test for an anomaly situation
	// input: 7 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// output: 0x7fffffffffffffff
	//
	// no problem if input unequals to output
	zeroB56 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00}
	if BtI64(zeroB56) != 0x7fffffffffffffff {
		t.Fatalf("it needs that input equals to output")
	}

	// TODO: test a large size bytes array
	//zeroB18446744073709551615 := []byte{0x00 * 18446744073709551615}
	//if BtI64(zeroB18446744073709551615) != 0x7fffffffffffffff {
	//	t.Fatalf("it needs that input equals to output")
	//}

	// test for an anomaly situation
	// input: empty bytes array
	// output: 0x7fffffffffffffff
	//
	// no problem if input equals to output
	emptyB0 := []byte{}
	if BtI64(emptyB0) != 0x7fffffffffffffff {
		t.Fatalf("it needs that input equals to output")
	}
}

// BtI64
//
// Test three patterns, zero and max for int64
// These are obvious but test in case
func TestBtI64(t *testing.T) {
	// if b is nil, it occurs an error
	// when execution because a type of b is not []byte

	// test for a normal situation
	// input: 8 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// output: zero(int64)
	//
	// no problem if input equals to output
	var zeroI64 int64 = 0
	zeroB64 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}
	if BtI64(zeroB64) != zeroI64 {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: 8 bytes array, []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	// output: max(int64)
	//
	// no problem if input equals to output
	var plusMaxI64 int64 = 9223372036854775807
	plusMaxB64 := []byte{0x7f, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff}
	if BtI64(plusMaxB64) != plusMaxI64 {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: 8 bytes array, []byte{0x80, 0x00, 0x00, 0x00}
	// output: min(int64)
	//
	// no problem if input equals to output
	var minusMaxI64 int64 = -9223372036854775808
	minusMaxB64 := []byte{0x80, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}
	if BtI64(minusMaxB64) != minusMaxI64 {
		t.Fatalf("it needs that input equals to output")
	}
}

// BtU64
//
// BtU64 has an input of a byte array for arbitrary length(the length is a int type)
// If the length of the input is not 8, it returns fixed value.
// This function tests the length
func TestLenBInBtU64(t *testing.T) {
	// if b is nil, it occurs an error
	// when execution because a type of b is not []byte

	// test for a normal situation
	// input: 8 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// output: 0xffffffffffffffff
	//
	// no problem if input unequals to output
	zeroB64 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}
	if BtU64(zeroB64) == 0xffffffffffffffff {
		t.Fatalf("it needs that input unequals to output")
	}

	// test for an anomaly situation
	// input: 9 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// output: 0xffffffffffffffff
	//
	// no problem if input equals to output
	zeroB72 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00}
	if BtU64(zeroB72) != 0xffffffffffffffff {
		t.Fatalf("it needs that input equals to output")
	}

	// test for an anomaly situation
	// input: 7 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// output: 0xffffffffffffffff
	//
	// no problem if input equals to output
	zeroB56 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00}
	if BtU64(zeroB56) != 0xffffffffffffffff {
		t.Fatalf("it needs that input equals to output")
	}

	// TODO: test a large size bytes arary
	//zeroB18446744073709551615 := []byte{0x00 * 18446744073709551615}
	//if BtU64(zeroB18446744073709551615) != 0xffffffffffffffff {
	//	t.Fatalf("it needs that input equals to output")
	//}

	// test for an anomaly situation
	// input: empty bytes array
	// output: 0xffffffffffffffff
	//
	// no problem if input equals to output
	emptyB0 := []byte{}
	if BtU64(emptyB0) != 0xffffffffffffffff {
		t.Fatalf("it needs that input equals to output")
	}
}

// BtU64
//
// Test three patterns, zero and max for uint64
// These are obvious but test in case
func TestBtU64(t *testing.T) {
	// if b is nil, it occurs an error
	// when execution because a type of b is not []byte

	// test for a normal situation
	// input: 8 bytes array, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// output: zero(uint32)
	//
	// no problem if input equals to output
	var zeroU64 uint64 = 0
	zeroB64 := []byte{0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}
	if BtU64(zeroB64) != zeroU64 {
		t.Fatalf("it needs that input equals to output")
	}

	// test for a normal situation
	// input: 8 bytes array, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	// output: max(uint32)
	//
	// no problem if input equals to output
	var maxU64 uint64 = 18446744073709551615
	maxB64 := []byte{0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff}
	if BtU64(maxB64) != maxU64 {
		t.Fatalf("it needs that input equals to output")
	}
}
