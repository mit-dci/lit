package sig64

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/roasbeef/btcd/btcec"
	"github.com/roasbeef/btcd/wire"
)

var (
	sig1big, _ = hex.DecodeString("304402206048246c95429555d265472d936b71e728f468a84412f9423941b4b9cbbab2f002204eb1bf82879c72adc3390a638a221792adecf74a097de9bd1257b5bc3e17a407")
	sig2big, _ = hex.DecodeString("3045022100b88e7a9137efe437cae4f3dd5e0d05bdf9cf519c10c36d2f657d6bbb0906a50f022041c95b40dc5423864022f5d110a810c50c1dd72c4b679da75e0134f4f5903609")
	sig3big, _ = hex.DecodeString("3045022059f28edc62e4b744ff7097717b7d4701614e4af6a30dfa2081ef3e8e279241840221008279ca7eb40a4bd04c923b96110b00d472d648c67df09ad39945130b8f7e4dc8")
)

// TestRandom makes random signatures and compresses / decompresses them
func TestRandom(t *testing.T) {
	for i := 0; i < 1000; i++ {
		priv, _ := btcec.NewPrivateKey(btcec.S256())
		sig, err := priv.Sign(wire.DoubleSha256([]byte{byte(i)}))
		if err != nil {
			t.Fatal(err)
		}
		bigsig := sig.Serialize()
		lilsig, err := SigCompress(bigsig)
		if err != nil {
			t.Fatal(err)
		}
		decsig := SigDecompress(lilsig)
		if !bytes.Equal(bigsig, decsig) {
			t.Fatalf("big/recover/comp:\n%x\n%x\n%x\n", bigsig, decsig, lilsig)

		}
		t.Logf("big/recover:\n%x\n%x\n", bigsig, decsig)

	}
}

// TestHardCoded tries compressing / decompressing hardcoded sigs
func TestHardCoded(t *testing.T) {
	c1, err := SigCompress(sig1big)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("big1:\n%x\ncom1:\n%x\n", sig1big, c1)

	c2, err := SigCompress(sig2big)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("big2:\n%x\ncom2:\n%x\n", sig2big, c2)

	c3, err := SigCompress(sig3big)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("big3:\n%x\ncom3:\n%x\n", sig3big, c3)

	r1 := SigDecompress(c1)
	t.Logf("dec1:\n%x\n", r1)

	r2 := SigDecompress(c2)
	t.Logf("dec1:\n%x\n", r2)

	r3 := SigDecompress(c3)
	t.Logf("dec1:\n%x\n", r3)

}
