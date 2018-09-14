package portxo

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// KeyGen describes how to get to the key from the master / seed.
// it can be used with bip44 or other custom schemes (up to 5 levels deep)
// Depth must be 0 to 5 inclusive.  Child indexes of 0 are OK, so we can't just
// terminate at the first 0.
type KeyGen struct {
	Depth   uint8     `json:"depth"`   // how many levels of the path to use. 0 means privkey as-is
	Step    [5]uint32 `json:"steps"`   // bip 32 / 44 path numbers
	PrivKey [32]byte  `json:"privkey"` // private key
}

// Bytes returns the 53 byte serialized key derivation path.
// always works
func (k KeyGen) Bytes() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, k.Depth)
	binary.Write(&buf, binary.BigEndian, k.Step[0])
	binary.Write(&buf, binary.BigEndian, k.Step[1])
	binary.Write(&buf, binary.BigEndian, k.Step[2])
	binary.Write(&buf, binary.BigEndian, k.Step[3])
	binary.Write(&buf, binary.BigEndian, k.Step[4])
	buf.Write(k.PrivKey[:])
	return buf.Bytes()
}

// KeyGenFromBytes turns a 53 byte array into a key derivation path.  Always works
// (note a depth > 5 path is invalid, but this just deserializes & doesn't check)
func KeyGenFromBytes(b [53]byte) (k KeyGen) {
	buf := bytes.NewBuffer(b[:])
	binary.Read(buf, binary.BigEndian, &k.Depth)
	binary.Read(buf, binary.BigEndian, &k.Step[0])
	binary.Read(buf, binary.BigEndian, &k.Step[1])
	binary.Read(buf, binary.BigEndian, &k.Step[2])
	binary.Read(buf, binary.BigEndian, &k.Step[3])
	binary.Read(buf, binary.BigEndian, &k.Step[4])
	copy(k.PrivKey[:], buf.Next(32))
	return
}

// String turns a keygen into a string
func (k KeyGen) String() string {
	var s string
	//	s = fmt.Sprintf("\tkey derivation path: m")
	for i := uint8(0); i < k.Depth; i++ {
		if k.Step[i]&0x80000000 != 0 { // high bit means hardened
			s += fmt.Sprintf("/%d'", k.Step[i]&0x7fffffff)
		} else {
			s += fmt.Sprintf("/%d", k.Step[i])
		}
	}
	return s
}
