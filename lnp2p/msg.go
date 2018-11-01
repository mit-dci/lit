package lnp2p

import (
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
)

// FIXME This is a stub function that just calls out to the lnutil lib for later.
func processMessage(b []byte, peer *Peer) (lnutil.LitMsg, error) {
	m, err := lnutil.LitMsgFromBytes(b, peer.Idx)
	return m, err
}

// Message is any kind of message that can go over the network.
type Message interface {
	Type() uint8
	Bytes() []byte
}

type outgoingmsg struct {
	peer       *Peer
	message    *Message
	finishchan *chan error
}

func sendMessages(queue chan outgoingmsg) {

	// NOTE Should we really be using the "peermgr" for log messages here?

	for {
		recv := <-queue
		m := *recv.message

		// Sending a message with a nil peer is how we signal to "stop sending things".
		if recv.peer == nil {
			if recv.finishchan != nil {
				*recv.finishchan <- nil
			}
			break
		}

		// Sanity check.
		if recv.message == nil {
			logging.Warnf("peermgr: Directed to send nil message, somehow\n")
			if recv.finishchan != nil {
				*recv.finishchan <- nil
			}
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
			logging.Warnf("peermgr: Tried to send message to disconnected peer %d, %s\n", recv.peer.Idx, recv.peer.Nickname)
			if recv.finishchan != nil {
				*recv.finishchan <- nil
			}
			continue
		}

		// Actually write it.
		_, err := conn.Write(buf)
		if err != nil {
			logging.Warnf("peermgr: Error sending message to peer: %s\n", err.Error())
		}

		// Responses, if applicable.
		if recv.finishchan != nil {
			*recv.finishchan <- err
		}

	}

	logging.Infof("peermgr: send message queue terminating")
}
