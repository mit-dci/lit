package lnp2p

import (
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lnutil"
)

// FIXME This is a stub function that just calls out to the lnutil lib for later.
func processMessage(b []byte, peer *Peer) (lnutil.LitMsg, error) {
	m, err := lnutil.LitMsgFromBytes(b, peer.GetIdx())
	return m, err
}

// ---------- TEMPORARY ---------- //
/*
 * This is here to help interface with the rest of qln.  There's a handler for
 * these events that just passes them into the OmniHandler like the previous
 * implementation would.
 */

// NetMessageRecvEvent is fired when a network message is recieved.
type NetMessageRecvEvent struct {
	Msg    *lnutil.LitMsg
	Peer   *Peer
	Rawbuf []byte
}

// Name .
func (e NetMessageRecvEvent) Name() string {
	return "TMP!lnp2p.msgrecv"
}

// Flags .
func (e NetMessageRecvEvent) Flags() uint8 {
	return eventbus.EFLAG_UNCANCELLABLE
}
