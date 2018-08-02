package main

import (
	"crypto/rand" // slows down nonce generation a lot, could use math/rand maybe?
	"crypto/sha256"
	//"encoding/hex"
	//"fmt"
	"log"

	//"github.com/mit-dci/lit/bech32"
	//"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/btcec"
)

type Hash [32]byte


const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
)

func genNonceSecure(length int) []byte {

	result := make([]byte, length)
	bufferSize := int(float64(length) * 1.3)
	for i, j, randomBytes := 0, 0, []byte{}; i < length; j++ {
		if j%bufferSize == 0 {
			randomBytes = getRandomness(bufferSize)
		}
		if idx := int(randomBytes[j%length] & letterIdxMask); idx < len(letterBytes) {
			result[i] = letterBytes[idx]
			i++
		}
	}
	return result
}

// getRandomness returns the requested number of bytes using crypto/rand
func getRandomness(length int) []byte {
	var randomBytes = make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatal("Unable to generate random bytes")
	}
	return randomBytes
}

func grind(pubKey *btcec.PublicKey, nonce []byte) []byte {
	data := make([]byte, 73) //33 + 40 bytes
	copy(data[34:73], nonce)
	copy(data[:33], pubKey.SerializeCompressed())
	shasum := sha256.Sum256(data)
	return shasum[:32]
}

func GenNewAddress() (*btcec.PublicKey, []byte, error) {
	privKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, err
	}

	pubKey := privKey.PubKey()

	for i := 0 ; i < 600000 ; i ++ {
		//if grind(pubKey, genNonceSecure(10)) < PoWTarget {
		//	return pubKey, nil
		//}
		nonce := genNonceSecure(10)
		a := grind(pubKey, nonce)
		if a[0] == 0 && a[1] == 0 {
			log.Println("NONCE FOUND")
			log.Println(a)
			return pubKey, nonce, nil
		}
	}
	return pubKey, nil, nil
}

type FoundAddr struct {
	privkey    btcec.PrivateKey
	pubkeyhash []byte
	id         string
}

func main() {
	_, _, _ = GenNewAddress()
}
