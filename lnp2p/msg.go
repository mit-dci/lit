package lnp2p

import (
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lnutil"
	"log"
)

// FIXME This is a stub function that just calls out to the lnutil lib for later.
func processMessage(b []byte, peer *Peer) (lnutil.LitMsg, error) {
	m, err := lnutil.LitMsgFromBytes(b, peer.GetIdx())
	return m, err
}

type outgoingmsg struct {
	peer    *Peer
	message *Message
}

func sendMessages(queue chan outgoingmsg) {

	// NOTE Should we really be using the "peermgr" for log messages here?

	for {
		recv := <-queue
		m := *recv.message

		// Sending a message with a nil peer is how we signal to "stop sending things".
		if recv.peer == nil {
			break
		}

		// Sanity check.
		if recv.message == nil {
			log.Printf("peermgr: Directed to send nil message, somehow\n")
			continue
		}

		// Assemble the final message, with type prepended.
		outbytes := m.Bytes()
		buf := make([]byte, len(outbytes)+1)
		buf[0] = m.Type()
		copy(buf[1:], outbytes)

		// Make sure the connection isn't closed.  This can happen if the message was queued but then we disconnected from the peer before it was sent.
		conn := recv.peer.conn
		if conn == nil {
			log.Printf("peermgr: Tried to send message to disconnected peer %s\n", recv.peer.GetPrettyName())
			continue
		}

		// Actually write it.
		_, err := conn.Write(buf)
		if err != nil {
			log.Printf("peermgr: Error sending message to peer: %s\n", err.Error())
		}

	}

	log.Printf("peermgr: send message queue terminating")
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
