package elkrem

import (
	"testing"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
)

// TestElkremBig tries 10K hashes
func TestElkremBig(t *testing.T) {
	sndr := NewElkremSender(chainhash.DoubleHashH([]byte("elktest")))
	var rcv ElkremReceiver
	//	SenderSerdesTest(t, sndr)
	for n := uint64(0); n < 10000; n++ {
		sha, err := sndr.AtIndex(n)
		if err != nil {
			t.Fatal(err)
		}
		err = rcv.AddNext(sha)
		if err != nil {
			t.Fatal(err)
		}
		if n%1000 == 999 {
			t.Logf("stack with %d received hashes\n", n+1)
			for i, n := range rcv.Nodes {
				t.Logf("Stack element %d: index %d height %d %s\n",
					i, n.I, n.H, n.Sha.String())
			}
		}
	}
	//	SenderSerdesTest(t, sndr)
	ReceiverSerdesTest(t, &rcv)
	for n := uint64(0); n < 10000; n += 500 {
		sha, err := rcv.AtIndex(n)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Retreived index %d %s\n", n, sha.String())
	}
}

// TestElkremLess tries 10K hashes
func TestElkremLess(t *testing.T) {
	sndr := NewElkremSender(chainhash.DoubleHashH([]byte("elktest2")))
	var rcv ElkremReceiver
	for n := uint64(0); n < 5000; n++ {
		sha, err := sndr.AtIndex(n)
		if err != nil {
			t.Fatal(err)
		}
		err = rcv.AddNext(sha)
		if err != nil {
			t.Fatal(err)
		}
		if n%1000 == 999 {
			t.Logf("stack with %d received hashes\n", n+1)
			for i, n := range rcv.Nodes {
				t.Logf("Stack element %d: index %d height %d %s\n",
					i, n.I, n.H, n.Sha.String())
			}
		}
	}
	for n := uint64(0); n < 5000; n += 500 {
		sha, err := rcv.AtIndex(n)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Retreived index %d %s\n",
			n, sha.String())
	}
}

// TestElkremIngestLeftFail puts a bed hash in such that the left child will fail
func TestElkremIngestLeftFail(t *testing.T) {
	sndr := NewElkremSender(chainhash.DoubleHashH([]byte("elkfailL")))
	var rcv ElkremReceiver
	for n := uint64(0); n < 31; n++ {
		sha, err := sndr.AtIndex(n)
		if err != nil {
			t.Fatal(err)
		}
		err = rcv.AddNext(sha)
		if err != nil {
			t.Fatal(err)
		}
	}

	// This is correct but we can't check; anything will be accepted
	sha, err := sndr.AtIndex(31)
	if err != nil {
		t.Fatal(err)
	}
	// flip all the bits in the first byte
	sha[0] ^= 0xff
	err = rcv.AddNext(sha)
	if err != nil {
		t.Fatal(err)
	}
	// give the right thing here, but it's too late as 31 was wrong
	sha, err = sndr.AtIndex(32)
	if err != nil {
		t.Fatal(err)
	}

	err = rcv.AddNext(sha)
	if err != nil {
		t.Fatal(err)
	}
	sha, err = sndr.AtIndex(33)
	if err != nil {
		t.Fatal(err)
	}
	err = rcv.AddNext(sha)
	if err == nil {
		t.Fatalf("Should have a left child mismatch, but everything went OK!")
	}
}

// TestElkremIngestRightFail puts a bed hash in such that the left child will fail
func TestElkremIngestRightFail(t *testing.T) {
	sndr := NewElkremSender(chainhash.DoubleHashH([]byte("elkfailR")))
	var rcv ElkremReceiver
	for n := uint64(0); n < 31; n++ {
		sha, err := sndr.AtIndex(n)
		if err != nil {
			t.Fatal(err)
		}
		err = rcv.AddNext(sha)
		if err != nil {
			t.Fatal(err)
		}
	}

	// This is correct but we can't check; anything will be accepted
	sha, err := sndr.AtIndex(31)
	if err != nil {
		t.Fatal(err)
	}
	err = rcv.AddNext(sha)
	if err != nil {
		t.Fatal(err)
	}
	sha, err = sndr.AtIndex(32)
	if err != nil {
		t.Fatal(err)
	}
	// flip all the bits in the first byte
	sha[0] ^= 0xff
	err = rcv.AddNext(sha)
	if err != nil {
		t.Fatal(err)
	}
	sha, err = sndr.AtIndex(33)
	if err != nil {
		t.Fatal(err)
	}
	err = rcv.AddNext(sha)
	if err == nil {
		t.Fatalf("Should have a right child mismatch, but everything went OK!")
	}
}

func TestFixed(t *testing.T) {
	root, _ := chainhash.NewHashFromStr(
		"b43614f251760d689adf84211148a40d7dee13967b7109e13c8d1437a4966d58")

	sndr := NewElkremSender(*root)

	zero, _ := chainhash.NewHashFromStr(
		"2a124935e0713149b71ff17cb43465e9828bacd1e833f0dc08460783a6a42cb4")

	thousand, _ := chainhash.NewHashFromStr(
		"0151a39169940cdd8ccf1ba619f254ddbf16ce260a243528839b2634eaa63d0a")

	for n := uint64(0); n < 5000; n += 500 {
		sha, err := sndr.AtIndex(n)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("elk %d: %s\n", n, sha.String())

		if n == 0 && !sha.IsEqual(zero) {
			t.Fatalf("Elk %d expected %s, got %s", n, zero.String(), sha.String())
		}
		if n == 1000 && !sha.IsEqual(thousand) {
			t.Fatalf("Elk %d expected %s, got %s", n, thousand.String(), sha.String())
		}

	}

}
