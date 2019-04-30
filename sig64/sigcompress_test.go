package sig64

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/btcutil/txscript"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/wire"
)

var (
	sig1big, _ = hex.DecodeString("304402206048246c95429555d265472d936b71e728f468a84412f9423941b4b9cbbab2f002204eb1bf82879c72adc3390a638a221792adecf74a097de9bd1257b5bc3e17a407")
	sig2big, _ = hex.DecodeString("3045022100b88e7a9137efe437cae4f3dd5e0d05bdf9cf519c10c36d2f657d6bbb0906a50f022041c95b40dc5423864022f5d110a810c50c1dd72c4b679da75e0134f4f5903609")
	sig3big, _ = hex.DecodeString("3045022059f28edc62e4b744ff7097717b7d4701614e4af6a30dfa2081ef3e8e279241840221008279ca7eb40a4bd04c923b96110b00d472d648c67df09ad39945130b8f7e4dc8")
)

// TestRandom makes random signatures and compresses / decompresses them
func TestRandom(t *testing.T) {
	for i := 0; i < 8; i++ {
		priv, _ := koblitz.NewPrivateKey(koblitz.S256())
		sig, err := priv.Sign(chainhash.DoubleHashB([]byte{byte(i)}))
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

// TestShortSignature is short signature case
func TestShortSignature(t *testing.T) {
	tx := wire.NewMsgTx()
	hash, _ := chainhash.NewHashFromStr("000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f")
	op := wire.NewOutPoint(hash, 0)
	txin := wire.NewTxIn(op, nil, nil)
	tx.AddTxIn(txin)
	pkScript := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}
	txout := wire.NewTxOut(100000000, pkScript)
	tx.AddTxOut(txout)
	sigHashes := txscript.NewTxSigHashes(tx)
	idx := 0
	subScript := []byte{0x14, 0x98, 0x97, 0xfd, 0x2b, 0x98, 0x0f, 0xec, 0xca, 0xeb, 0x9c, 0x63, 0xc2, 0x74, 0x9b, 0x38, 0x9c, 0x77, 0x2a, 0x9d, 0x75}
	amt := int64(100001000)
	hashType := txscript.SigHashAll
	key, _ := koblitz.PrivKeyFromBytes(koblitz.S256(), []byte("privatekey"))
	sign, err := txscript.RawTxInWitnessSignature(tx, sigHashes, idx, amt, subScript, hashType, key)
	if err != nil {
		t.Fatal(err)
	}
	sign = sign[:len(sign)-1]
	t.Logf("sign:%x", sign)
	csig, err := SigCompress(sign)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("csig:%x", csig)
	dsig := SigDecompress(csig)
	t.Logf("dsig:%x", dsig)
	for i := range dsig {
		if sign[i] != dsig[i] {
			t.Fatalf("unmatch SigCompress/SigDecompress:%x/%x", sign, dsig)
		}
	}
}
