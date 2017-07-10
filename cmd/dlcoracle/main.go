package main

import (
	"fmt"

	"github.com/adiabat/btcd/chaincfg/chainhash"
	"github.com/mit-dci/lit/lnutil"
)

func main() {
	// given an uint64, split into 2 bytes.
	// byte1: 5 bit shift, 2 bit MSBs
	// byte2: 7 bit LSBs

	/*
		for i := uint64(1000); i < 1100; i++ {
			ba, bb, err := split(i)
			if err != nil {
				panic(err)
			}
			//		fmt.Printf("ba:%x bb:%x\n", ba, bb)

			r := join(ba, bb)
			fmt.Printf("rejoin %d\n", r)

		}
	*/

	privRoot := chainhash.HashH([]byte("my private key"))

	ka, Ra := deriveK(privRoot, "asset 1 a")

	kb, Rb := deriveK(privRoot, "asset 1 a")

	fmt.Printf("a: %x %x\n", ka, Ra)
	fmt.Printf("b: %x %x\n", kb, Rb)

	msg := chainhash.HashH([]byte("hi"))

	sa, err := RSign(msg, privRoot, ka)
	if err != nil {
		panic(err)
	}
	sb, err := RSign(msg, privRoot, kb)
	if err != nil {
		panic(err)
	}

	pubRootArr := lnutil.PubFromHash(privRoot)

	saGpub, err := SGpredict(msg, pubRootArr, Ra)
	if err != nil {
		panic(err)
	}

	sbGpub, err := SGpredict(msg, pubRootArr, Rb)
	if err != nil {
		panic(err)
	}

	saG := lnutil.PubFromHash(sa)
	sbG := lnutil.PubFromHash(sb)

	fmt.Printf("saG:\npredict\t%x\ncorrect\t%x\n", saGpub.SerializeCompressed(), saG)
	fmt.Printf("sbG:\npredict\t%x\ncorrect\t%x\n", sbGpub.SerializeCompressed(), sbG)

	return
}
