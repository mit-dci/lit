package shortadr

import (
	"crypto/rand" // slows down nonce generation a lot, could use math/rand maybe?
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/bech32"
	"encoding/hex"
	mathrand "math/rand"
	"log"
)

const (
	letterBytes   = "$_!;abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
)

func GenNonceSecure(length int) []byte {
	length = mathrand.Intn(20)
	// math/rand is much faster than crypto/rand, so I guess its fine to sacrifice
	// some randomness for much speed
	result := make([]byte, length)
	bufferSize := int(float64(length) * 1.3)
	for i, j, randomBytes := 0, 0, []byte{}; i < length; j++ {
		if j%bufferSize == 0 {
			randomBytes = GetRandomness(bufferSize)
		}
		if idx := int(randomBytes[j%length] & letterIdxMask); idx < len(letterBytes) {
			result[i] = letterBytes[idx]
			i++
		}
	}
	return result
}

// GetRandomness returns the requested number of bytes using crypto/rand
func GetRandomness(length int) []byte {
	var randomBytes = make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatal("Unable to generate random bytes")
	}
	return randomBytes
}

func Grind(byteString [33]byte, nonce []byte) []byte {
	data := make([]byte, 64) // 44 + 20 bytes
	copy(data[45:64], nonce)
	copy(data[:44], byteString[:])
	shasum := fastsha256.Sum256(data)
	return shasum[:32]
}

func CheckProofOfWork(in []byte) int {
	pow := 0
	for _, byte := range in {
		if byte != 0 {
			break
		}
		pow++
	}
	return pow
}

func GenVanityAdr(byteString [33]byte, vanityStr string) string {
	// generate your own vanity address based on your PoW bytes
	nonce := GenNonceSecure(20)
	a := Grind(byteString, nonce)
	vanityLen := len(vanityStr)
	vanityStr = vanityStr[:vanityLen]
	if vanityLen > 20 {
		vanityLen = 20
		vanityStr = vanityStr[:20]
	}
	vanity := bech32.Encode("ln", a[vanityLen:])
	for vanity[0:vanityLen] != vanityStr {
		//if grind(byteString, genNonceSecure(10)) < PoWTarget {
		//	return byteString, nil
		//}
		nonce = GenNonceSecure(20)
		a = Grind(byteString, nonce)
		if CheckProofOfWork(a) == vanityLen {
			log.Println("A:", hex.EncodeToString(a), "NONCE FOUND:", hex.EncodeToString(nonce))
			//temp := make([]byte, 30+20)
			//copy(temp[0:30], a)
			//copy(temp[31:50], nonce)
			vanity = bech32.Encode("ln", a[vanityLen:]) // 18 bytes passed, 33 - 15PoW bytes
			vanity = 	vanity[:3] + vanity[4:]
			log.Println("VANITY", vanity)
		}
	}
	return vanity
}
