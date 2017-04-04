package lnutil

import (
	"crypto/rand"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func TestAdr(t *testing.T) {

	for i := 0; i < 10; i++ {
		data := make([]byte, 16)
		_, _ = rand.Read(data)

		h := chainhash.DoubleHashH(data)
		pub := PubFromHash(h)
		adr := LitFullAdrEncode(pub)
		t.Logf("%d\tadr %s\n", i, adr)
		rePub, err := LitFullAdrDecode(adr)
		if err != nil {
			t.Fatal(err)
		}
		if pub != rePub {
			t.Fatalf("pubkey mismatch:\n%x\n%x\n", pub, rePub)
		}
		t.Logf("restore %x\n", rePub[:])

	}
}
