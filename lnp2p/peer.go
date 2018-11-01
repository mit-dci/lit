package lnp2p

import (
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lndc"
)

// A Peer is a remote client that's somehow connected to us.
type Peer struct {
	Addr     string
	Idx      uint32 // deprecated, default value for all uint32s are 0
	Nickname string

	conn   *lndc.Conn
	Pubkey *koblitz.PublicKey
	pmgr   *PeerManager
}

// GetRemoteAddr returns the remote node's public key.
func (p *Peer) GetRemoteAddr() string {
	return p.conn.RemoteAddr().String()
}

// SendQueuedMessage adds the message to the queue to be sent to this peer.
// This queue is shared across all peers.
func (p *Peer) SendQueuedMessage(msg Message) error {
	return p.pmgr.queueMessageToPeer(p, msg, nil)
}

// SendImmediateMessage adds a message to the queue but waits for the message to
// be sent before returning, also returning errors that might have occurred when
// sending the message, like the peer disconnecting.
func (p *Peer) SendImmediateMessage(msg Message) error {
	var err error
	errchan := make(chan error)

	// Send it to the queue, as above.
	err = p.pmgr.queueMessageToPeer(p, msg, &errchan)
	if err != nil {
		return err
	}

	// Catches errors if there are any.
	err = <-errchan
	if err != nil {
		return err
	}

	return nil
}

// IntoPeerInfo generates the PeerInfo DB struct for the Peer.
func (p *Peer) IntoPeerInfo() lncore.PeerInfo {
	var raddr string
	if p.conn != nil {
		raddr = p.conn.RemoteAddr().String()
	}
	return lncore.PeerInfo{
		Addr:     p.Addr,
		Nickname: p.Nickname,
		NetAddr:  raddr,
	}
}
