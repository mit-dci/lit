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

	privRoot := chainhash.HashH([]byte("my private keyzzzz"))

	ka, Ra := deriveK(privRoot, "asset 1 a01zzz")

	kb, Rb := deriveK(privRoot, "asset 1 b021")

	fmt.Printf("key: %x\n", privRoot[:])
	//	fmt.Printf("b: %x %x\n", kb, Rb)

	msg := chainhash.HashH([]byte("hi0s"))

	sa, err := RSign(msg, privRoot, ka)
	if err != nil {
		panic(err)
	}
	sb, err := RSign(msg, privRoot, kb)
	if err != nil {
		panic(err)
	}

	fmt.Printf("---\n")
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

	oneTry(privRoot, ka, msg)

	return
}

func oneTry(a, k, m [32]byte) {
	pubRootArr := lnutil.PubFromHash(a)
	rArr := lnutil.PubFromHash(k)

	s, err := RSign(m, a, k)
	if err != nil {
		panic(err)
	}

	sGarr := lnutil.PubFromHash(s)

	sGpred, err := SGpredict(m, pubRootArr, rArr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("p: %x\nc: %x\n", sGarr[:], sGpred.SerializeCompressed())

}
