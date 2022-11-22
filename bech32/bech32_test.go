package bech32

import (
	"bytes"
	"math/rand"
	"testing"
)

type validTestAddress struct {
	address string
	data    []byte
}

var (
	validChecksum = []string{
		"A12UEL5L",
		"an83characterlonghumanreadablepartthatcontainsthenumber1andtheexcludedcharactersbio1tt5tgs",
		"abcdef1qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw",
		"11qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqc8247j",
		"split1checkupstagehandshakeupstreamerranterredcaperred2y9e3w",
	}

	validSegwitAddresses = []validTestAddress{
		{
			"BC1QW508D6QEJXTDG4Y5R3ZARVARY0C5XW7KV8F3T4",
			[]byte{0x00, 0x14, 0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54,
				0x94, 0x1c, 0x45, 0xd1, 0xb3, 0xa3, 0x23, 0xf1, 0x43, 0x3b, 0xd6},
		},
		{
			"tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sl5k7",
			[]byte{0x00, 0x20, 0x18, 0x63, 0x14, 0x3c, 0x14, 0xc5, 0x16, 0x68, 0x04,
				0xbd, 0x19, 0x20, 0x33, 0x56, 0xda, 0x13, 0x6c, 0x98, 0x56, 0x78,
				0xcd, 0x4d, 0x27, 0xa1, 0xb8, 0xc6, 0x32, 0x96, 0x04, 0x90, 0x32,
				0x62},
		},
		{
			"bc1pw508d6qejxtdg4y5r3zarvary0c5xw7kw508d6qejxtdg4y5r3zarvary0c5xw7k7grplx",
			[]byte{0x81, 0x28, 0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54,
				0x94, 0x1c, 0x45, 0xd1, 0xb3, 0xa3, 0x23, 0xf1, 0x43, 0x3b, 0xd6,
				0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54, 0x94, 0x1c,
				0x45, 0xd1, 0xb3, 0xa3, 0x23, 0xf1, 0x43, 0x3b, 0xd6},
		},
		{
			"BC1SW50QA3JX3S",
			[]byte{0x90, 0x02, 0x75, 0x1e},
		},
		{
			"bc1zw508d6qejxtdg4y5r3zarvaryvg6kdaj",
			[]byte{
				0x82, 0x10, 0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54,
				0x94, 0x1c, 0x45, 0xd1, 0xb3, 0xa3, 0x23},
		},
		{
			"tb1qqqqqp399et2xygdj5xreqhjjvcmzhxw4aywxecjdzew6hylgvsesrxh6hy",
			[]byte{0x00, 0x20, 0x00, 0x00, 0x00, 0xc4, 0xa5, 0xca, 0xd4, 0x62, 0x21,
				0xb2, 0xa1, 0x87, 0x90, 0x5e, 0x52, 0x66, 0x36, 0x2b, 0x99, 0xd5,
				0xe9, 0x1c, 0x6c, 0xe2, 0x4d, 0x16, 0x5d, 0xab, 0x93, 0xe8, 0x64,
				0x33},
		},
	}

	invalidAddress = []string{
		"bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kg3g4ty",
		"bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t5",
		"BC13W508D6QEJXTDG4Y5R3ZARVARY0C5XW7KN40WF2",
		"bc1rw5uspcuh",
		"bc10w508d6qejxtdg4y5r3zarvary0c5xw7kw508d6qejxtdg4y5r3zarvary0c5xw7kw5rljs90",
		"BC1QR508D6QEJXTDG4Y5R3ZARVARYV98GJ9P",
		"tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sL5k7",
		"tb1pw508d6qejxtdg4y5r3zarqfsj6c3",
		"tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3pjxtptv",
	}
)

// TestRandomEncodeDecode makes some random addresses and makes sure
// the same data comes out as went in
func TestRandomEncodeDecode(t *testing.T) {

	tHrp := "testhrp"

	tbHrp := "tb"

	for i := 0; i < 20; i++ {
		data := make([]byte, 20)
		_, _ = rand.Read(data)
		rand.Int63()
		// create an arbitrary, non-segwit address
		nonSegWitAdr := Encode(tHrp, data)
		tHrp2, data2, err := Decode(nonSegWitAdr)
		if err != nil {
			t.Fatal(err)
		}
		if tHrp2 != tHrp {
			t.Fatalf("hrp mismatch %s, %s", tHrp2, tHrp)
		}
		if !bytes.Equal(data, data2) {
			t.Fatalf("data mismatch %x, %x", data, data2)
		}

		// append a 0x00, 0x14 to make it a p2wpkh script
		data = append([]byte{0x00, 0x14}, data...)

		// make a testnet address with the same 20 byte pubkeyhash
		segWitAdr, err := SegWitAddressEncode(tbHrp, data)
		if err != nil {
			t.Fatal(err)
		}

		// parse the segwit address we just created back into data
		data2, err = SegWitAddressDecode(segWitAdr)
		if err != nil {
			t.Fatal(err)
		}
		// check that the data is still intact
		if !bytes.Equal(data, data2) {
			t.Fatalf("data mismatch %x, %x", data, data2)
		}
	}
}

// TestHardCoded checks the hard-coded test vectors descrbed in the BIP
func TestHardCoded(t *testing.T) {
	// check that all the valid addresses check out as valid
	for _, adr := range validChecksum {
		_, _, err := Decode(adr)
		if err != nil {
			t.Fatalf("address %s invalid:%s", adr, err)
		}
	}

	// invalid addresses should all have some kind of error
	for _, adr := range invalidAddress {
		data, err := SegWitAddressDecode(adr)
		if err == nil {
			t.Logf("data %x\n", data)
			t.Fatalf("address %s should fail but didn't", adr)
		}
	}

	// check that all the valid segwit addresses come out valid, and that
	// they match the data provided
	for _, swadr := range validSegwitAddresses {
		data, err := SegWitAddressDecode(swadr.address)
		if err != nil {
			t.Logf("data: %x\n", data)
			t.Fatalf("address %s failed: %s", swadr.address, err.Error())
		}
		if !bytes.Equal(data, swadr.data) {
			t.Fatalf("address %s data mismatch %x, %x",
				swadr.address, swadr.data, data)
		}
	}
}
