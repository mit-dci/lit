package lnutil

import (
	"bytes"
	"math/rand"
	"testing"
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

	if !bytes.Equal(b, msg2.Bytes()) {
		t.Fatalf("bytes mismatch:\n%x\n%x\n", b, msg2.Bytes())
	}

	if msg.Peer() != msg2.Peer() {
		t.Fatalf("peer mismatch:\n%x\n%x\n", msg.Peer(), msg2.Peer())
	}

}
