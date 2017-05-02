package lnutil

import (
	"math/rand"
	"testing"

	"github.com/adiabat/btcd/chaincfg/chainhash"
)

func TestChatMsg(t *testing.T) {
	peerid := rand.Uint32()
	text := "hello"

	msg := NewChatMsg(peerid, text)
	b := msg.Bytes()

	msg2, err := NewChatMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:1], peerid) //purposely error to check working

	if err == nil {
		t.Fatalf("Should have errored Chat Msg, but didn't")
	}
}

func TestPointReqMsg(t *testing.T) {
	peerid := rand.Uint32()
	cointype := rand.Uint32()

	msg := NewPointReqMsg(peerid, cointype)
	b := msg.Bytes()

	msg2, err := NewPointReqMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:3], peerid) //purposely error to check working

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestPointRespMsg(t *testing.T) {
	peerid := rand.Uint32()
	channelPub := make([]byte, 33)
	refundPub := make([]byte, 33)
	HAKDbase := make([]byte, 33)
	_, _ = rand.Read(channelPub)
	_, _ = rand.Read(refundPub)
	_, _ = rand.Read(HAKDbase)

	var cp [33]byte
	copy(cp[:], channelPub[:])
	var rp [33]byte
	copy(rp[:], refundPub[:])
	var hb [33]byte
	copy(hb[:], HAKDbase[:])

	msg := NewPointRespMsg(peerid, cp, rp, hb)
	b := msg.Bytes()

	msg2, err := NewPointRespMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:98], peerid) //purposely error to check working

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestChanDescMsg(t *testing.T) {
	peerid := rand.Uint32()
	var outPoint [36]byte
	var pubKey [33]byte
	var refundPub [33]byte
	var hakd [33]byte
	capacity := rand.Int63()
	payment := rand.Int63()
	var elkZero [33]byte
	var elkOne [33]byte
	var elkTwo [33]byte

	_, _ = rand.Read(outPoint[:])
	_, _ = rand.Read(pubKey[:])
	_, _ = rand.Read(refundPub[:])
	_, _ = rand.Read(hakd[:])
	_, _ = rand.Read(elkZero[:])
	_, _ = rand.Read(elkOne[:])
	_, _ = rand.Read(elkTwo[:])

	op := *OutPointFromBytes(outPoint)

	msg := NewChanDescMsg(peerid, op, pubKey, refundPub, hakd,
		capacity, payment, elkZero, elkOne, elkTwo)
	b := msg.Bytes()

	msg2, err := NewChanDescMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:250], peerid) //purposely error to check working

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}

}

func TestChanAckMsg(t *testing.T) {
	peerid := rand.Uint32()
	var outPoint [36]byte
	var elkZero [33]byte
	var elkOne [33]byte
	var elkTwo [33]byte
	var sig [64]byte

	_, _ = rand.Read(outPoint[:])
	_, _ = rand.Read(sig[:])
	_, _ = rand.Read(elkZero[:])
	_, _ = rand.Read(elkOne[:])
	_, _ = rand.Read(elkTwo[:])

	op := *OutPointFromBytes(outPoint)

	msg := NewChanAckMsg(peerid, op, elkZero, elkOne, elkTwo, sig)
	b := msg.Bytes()

	msg2, err := NewChanAckMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:98], peerid) //purposely error to check working by not sending enough bytes

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestSigProofMsg(t *testing.T) {
	peerid := rand.Uint32()
	var outPoint [36]byte
	var sig [64]byte

	_, _ = rand.Read(outPoint[:])
	_, _ = rand.Read(sig[:])

	op := *OutPointFromBytes(outPoint)

	msg := NewSigProofMsg(peerid, op, sig)
	b := msg.Bytes()

	msg2, err := NewSigProofMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:99], peerid) //purposely error to check working by not sending enough bytes

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestCloseReqMsg(t *testing.T) {
	peerid := rand.Uint32()
	var outPoint [36]byte
	var sig [64]byte

	_, _ = rand.Read(outPoint[:])
	_, _ = rand.Read(sig[:])

	op := *OutPointFromBytes(outPoint)

	msg := NewCloseReqMsg(peerid, op, sig)
	b := msg.Bytes()

	msg2, err := NewCloseReqMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:99], peerid) //purposely error to check working by not sending enough bytes

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestDeltaSigMsg(t *testing.T) {
	peerid := rand.Uint32()
	var outPoint [36]byte
	delta := rand.Int31()
	var sig [64]byte

	_, _ = rand.Read(outPoint[:])
	_, _ = rand.Read(sig[:])

	op := *OutPointFromBytes(outPoint)

	msg := NewDeltaSigMsg(peerid, op, delta, sig)
	b := msg.Bytes()

	msg2, err := NewDeltaSigMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:99], peerid) //purposely error to check working by not sending enough bytes

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestSigRevMsg(t *testing.T) {
	peerid := rand.Uint32()
	var outPoint [36]byte
	var sig [64]byte
	var elk [32]byte
	var n2elk [33]byte

	_, _ = rand.Read(outPoint[:])
	_, _ = rand.Read(sig[:])
	_, _ = rand.Read(elk[:])
	_, _ = rand.Read(n2elk[:])

	op := *OutPointFromBytes(outPoint)
	Elk, _ := chainhash.NewHash(elk[:])

	msg := NewSigRev(peerid, op, sig, *Elk, n2elk)
	b := msg.Bytes()

	msg2, err := NewSigRevFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:99], peerid) //purposely error to check working by not sending enough bytes

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestGapSigRevMsg(t *testing.T) {
	peerid := rand.Uint32()
	var outPoint [36]byte
	var sig [64]byte
	var elk [32]byte
	var n2elk [33]byte

	_, _ = rand.Read(outPoint[:])
	_, _ = rand.Read(sig[:])
	_, _ = rand.Read(elk[:])
	_, _ = rand.Read(n2elk[:])

	op := *OutPointFromBytes(outPoint)
	Elk, _ := chainhash.NewHash(elk[:])

	msg := NewGapSigRev(peerid, op, sig, *Elk, n2elk)
	b := msg.Bytes()

	msg2, err := NewGapSigRevFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:99], peerid) //purposely error to check working by not sending enough bytes

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestRevMsg(t *testing.T) {
	peerid := rand.Uint32()
	var outPoint [36]byte
	var elk [32]byte
	var n2elk [33]byte

	_, _ = rand.Read(outPoint[:])
	_, _ = rand.Read(elk[:])
	_, _ = rand.Read(n2elk[:])

	op := *OutPointFromBytes(outPoint)
	Elk, _ := chainhash.NewHash(elk[:])

	msg := NewRevMsg(peerid, op, *Elk, n2elk)
	b := msg.Bytes()

	msg2, err := NewRevMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:99], peerid) //purposely error to check working by not sending enough bytes

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestWatchDescMsg(t *testing.T) {
	peerid := rand.Uint32()
	var pkh [20]byte
	delay := uint16(rand.Int())
	fee := rand.Int63()
	var customerBP [33]byte
	var adBP [33]byte

	_, _ = rand.Read(pkh[:])
	_, _ = rand.Read(customerBP[:])
	_, _ = rand.Read(adBP[:])

	msg := NewWatchDescMsg(peerid, pkh, delay, fee, customerBP, adBP)
	b := msg.Bytes()

	msg2, err := NewWatchDescMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:95], peerid) //purposely error to check working by not sending enough bytes

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}

func TestComMsg(t *testing.T) {
	peerid := rand.Uint32()
	var parTxid [16]byte
	var pkh [20]byte
	var elk [32]byte
	var sig [64]byte

	_, _ = rand.Read(parTxid[:])
	_, _ = rand.Read(elk[:])
	_, _ = rand.Read(pkh[:])
	_, _ = rand.Read(sig[:])

	Elk, _ := chainhash.NewHash(elk[:])

	msg := NewComMsg(peerid, pkh, *Elk, parTxid, sig)
	b := msg.Bytes()

	msg2, err := NewComMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg, msg2) {
		t.Fatalf("from bytes mismatch:\n%x\n%x\n", msg.Bytes(), msg2.Bytes())
	}

	msg3, err := LitMsgFromBytes(b, peerid)

	if err != nil {
		t.Fatal(err)
	}

	if !LitMsgEqual(msg2, msg3) {
		t.Fatalf("interface mismatch:\n%x\n%x\n", msg2.Bytes(), msg3.Bytes())
	}

	_, err = LitMsgFromBytes(b[:99], peerid) //purposely error to check working by not sending enough bytes

	if err == nil {
		t.Fatalf("Should have errored, but didn't")
	}
}
