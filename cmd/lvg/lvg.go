package main

import (
	"encoding/hex"
	"fmt"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/btcec"
)

func GenNewAddress() (*btcec.PrivateKey, []byte, error) {
	privk, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, err
	}

	// Run through the hashing ops.
	hashed := btcutil.Hash160(privk.PubKey().SerializeCompressed())

	// Shorten the address (?)
	firstNonzero := 0
	for i, b := range hashed {
		if b == 0 {
			firstNonzero = i + 1
		} else {
			break
		}
	}
	shortAddr := hashed[firstNonzero:]

	return privk, shortAddr, nil

}

func PrintLoop(ret chan FoundAddr) {
	for true {
		privkey, pubkeyhash, err := GenNewAddress()
		if err != nil {
			panic(err)
		}
		if len(pubkeyhash) <= 18 {
			ret <- FoundAddr{
				privkey:    *privkey,
				pubkeyhash: pubkeyhash,
				id:         bech32.Encode("ln", pubkeyhash),
			}
		}
	}
}

type FoundAddr struct {
	privkey    btcec.PrivateKey
	pubkeyhash []byte
	id         string
}

func main() {
	ret := make(chan FoundAddr)
	go PrintLoop(ret)
	go PrintLoop(ret)
	go PrintLoop(ret)
	go PrintLoop(ret)
	go PrintLoop(ret)
	go PrintLoop(ret)
	go PrintLoop(ret)
	go PrintLoop(ret)

	var next FoundAddr
	for true {
		next = <-ret
		hexstr := hex.EncodeToString(next.pubkeyhash)
		fmt.Printf("%d bytes: %s | hex: %s\n", len(next.pubkeyhash), next.id, hexstr)
	}
}
