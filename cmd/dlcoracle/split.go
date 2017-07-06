package main

import "fmt"

func main() {
	// given an uint64, split into 2 bytes.
	// byte1: 5 bit shift, 2 bit MSBs
	// byte2: 7 bit LSBs

	ba, bb, err := split(1024)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ba:%x bb:%x\n", ba, bb)

	r := join(ba, bb)
	fmt.Printf("rejoin %d\n", r)

	//fmt.Printf("")

	return
}

func split(x uint64) (uint8, uint8, error) {

	var ba, bb uint8
	//	 find most significant 1 bit in x
	highbit := highestbit(x)

	if highbit > 40 {
		return 0, 0, fmt.Errorf("given %d; cannot encode greater than 2**40", x)
	}

	var shift uint8

	if highbit < 9 {
		shift = 0
	} else {
		shift = highbit - 9
	}

	x = x >> shift

	ba = shift << 2
	ba |= uint8(x >> 7)

	bb = uint8(x & 0x7f)

	return ba, bb, nil
}

func join(ba, bb uint8) uint64 {
	var shift, result uint64

	shift = (uint64(ba) & 0x7c) >> 2
	fmt.Printf("shift: %d\n", shift)
	if shift > 0 {
		result |= 1 << 8
	}

	result |= (uint64(ba) & 0x03) << 7
	result |= uint64(bb)

	result = result << shift

	return result
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
