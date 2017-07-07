package main

import "fmt"

func main() {
	// given an uint64, split into 2 bytes.
	// byte1: 5 bit shift, 2 bit MSBs
	// byte2: 7 bit LSBs

	for i := uint64(1000); i < 1100; i++ {
		ba, bb, err := split(i)
		if err != nil {
			panic(err)
		}
		//		fmt.Printf("ba:%x bb:%x\n", ba, bb)

		r := join(ba, bb)
		fmt.Printf("rejoin %d\n", r)

	}

	//fmt.Printf("")

	return
}

// split a uint64 into two bytes
// first byte has an exponent and most significant bits,
// the second byte has the least significant bits.
func split(x uint64) (uint8, uint8, error) {

	var ba, bb uint8
	//	 find most significant 1 bit in x
	highbit := highestbit(x)

	if highbit > 40 {
		return 0, 0, fmt.Errorf("given %d; cannot encode greater than 2**40", x)
	}

	var shift uint8
	// A bit ugly in that there are two "0" shifts.  0 means no shift, and
	// don't start with a 1.  1 means no shift, but put a 1 in the highest
	// bit.
	if highbit == 9 {
		shift = 1
	}
	if highbit > 9 {
		shift = highbit - 8
		x = x >> (shift - 1)
	}

	ba = shift << 2
	ba |= uint8(x>>7) & 0x03
	bb = uint8(x & 0x7f)

	return ba, bb, nil
}

// join two bytes back into a uint64.
func join(ba, bb uint8) uint64 {
	var shift, result uint64

	shift = (uint64(ba)) >> 2 // could also say & 7c

	result |= (uint64(ba) & 0x03) << 7
	result |= uint64(bb)

	if shift == 0 {
		return result
	}

	result |= 1 << 9

	if shift == 1 {
		return result
	}

	return result << (shift - 1)
}

// highestbit returns the position of the highest set bit
func highestbit(x uint64) uint8 {
	for i := uint64(63); i > 0; i-- {
		if x&(uint64(1)<<i) != 0 {
			return uint8(i)
		}
	}
	return 0
}
