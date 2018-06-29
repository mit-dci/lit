package portxo

import (
	"testing"

	"github.com/mit-dci/lit/btcutil/btcd/chaincfg/chainhash"
)

// TestHardCoded tests serializing / deserializing a portxo
func TestHardCoded(t *testing.T) {
	var u1 PorTxo

	b1, err := u1.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	u2, err := PorTxoFromBytes(b1)
	if err != nil {
		t.Fatal(err)
	}

	//	u2.Op.Hash = chainhash.DoubleHashH([]byte("test"))
	u2.Op.Hash = chainhash.DoubleHashH([]byte("test"))
	u2.Op.Index = 3
	u2.Value = 1234567890
	u2.Mode = TxoP2PKHComp
	u2.Seq = 65535
	u2.KeyGen.Depth = 3
	u2.KeyGen.Step[0] = 0x8000002C
	u2.KeyGen.Step[1] = 1
	u2.KeyGen.Step[2] = 0x80000000

	//	u2.PrivKey[0] = 0x11
	u2.PkScript = []byte("1234567890123456")
	b2, err := u2.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	u3, err := PorTxoFromBytes(b2)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("b2: %x\n", b2)

	t.Logf("u2: %s", u2.String())

	b3, err := u3.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("b3: %x\n", b3)

	t.Logf("u3: %s", u3.String())
	if !u2.Equal(u3) {
		t.Fatalf("u2, u3 should be the same")
	}
}

// TestWithoutStuff tests serializing / deserializing a portxo without some things
func TestWithoutStuff(t *testing.T) {
	var u1 PorTxo

	b1, err := u1.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	u2, err := PorTxoFromBytes(b1)
	if err != nil {
		t.Fatal(err)
	}

	//	u2.Op.Hash = chainhash.DoubleHashH([]byte("test"))
	u2.Op.Hash = chainhash.DoubleHashH([]byte("test1"))
	u2.Op.Index = 0
	u2.Value = 5565989
	u2.Mode = TxoP2WSHComp
	//	u2.Seq = 0
	u2.PkScript = []byte("pub key script")
	u2.KeyGen.Depth = 1
	u2.KeyGen.Step[0] = 0x80000011

	b2, err := u2.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	u3, err := PorTxoFromBytes(b2)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("b2: %x\n", b2)

	t.Logf("u2: %s", u2.String())

	b3, err := u3.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("b3: %x\n", b3)

	t.Logf("u3: %s", u3.String())
	if !u2.Equal(u3) {
		t.Fatalf("u2, u3 should be the same")
	}
}

// TestWithStack tests serializing / deserializing a portxo with a presig stack
func TestWithStack(t *testing.T) {
	var u1 PorTxo

	b1, err := u1.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	u2, err := PorTxoFromBytes(b1)
	if err != nil {
		t.Fatal(err)
	}

	u2.Op.Hash = chainhash.DoubleHashH([]byte("test3"))
	u2.Op.Index = 3
	u2.Value = 5565989
	u2.Mode = TxoP2WSHComp
	u2.Seq = 0
	u2.KeyGen.Depth = 1
	u2.KeyGen.Step[0] = 0x8000002C

	//	u2.PrivKey[0] = 0x11
	u2.PkScript = []byte("00112233")
	u2.PreSigStack = make([][]byte, 3)
	u2.PreSigStack[0] = []byte("SIGSTACK00000")
	u2.PreSigStack[1] = []byte(".....STACK001")
	// PreSigStack[2] is nil

	b2, err := u2.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	u3, err := PorTxoFromBytes(b2)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("b2: %x\n", b2)

	t.Logf("u2: %s", u2.String())

	b3, err := u3.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("b3: %x\n", b3)

	t.Logf("u3: %s", u3.String())
	if !u2.Equal(u3) {
		t.Fatalf("u2, u3 should be the same")
	}
}
