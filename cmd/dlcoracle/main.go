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

	return
}
