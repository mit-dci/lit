package lnp2p

import (
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnio"
)

// A Peer is a remote client that's somehow connected to us.
type Peer struct {
	lnaddr   lnio.LnAddr
	nickname *string
	conn     *lndc.Conn
	idpubkey pubkey

	idx  *uint32 // deprecated
	pmgr *PeerManager
}

// GetIdx is a compatibility function.
func (p *Peer) GetIdx() uint32 {
	if p.idx == nil {
		return 0
	}
	return *p.idx
}

// GetNickname returns the nickname, or an empty string if unset.
func (p *Peer) GetNickname() string {
	if p.nickname == nil {
		return ""
	}
	return *p.nickname
}

// GetLnAddr returns the lightning network address for this peer.
func (p *Peer) GetLnAddr() lnio.LnAddr {
	return p.lnaddr
}

// GetRemoteAddr does something.
func (p *Peer) GetRemoteAddr() string {
	return p.conn.RemoteAddr().String()
}

// GetPubkey gets the public key for the user.
func (p *Peer) GetPubkey() btcec.PublicKey {
	return *p.idpubkey
}

// GetPrettyName returns a more human-readable name, such as the nickname if
// available or a trucated version of the LN address otherwise.
func (p *Peer) GetPrettyName() string {
	if p.nickname != nil {
		return *p.nickname
	}

	return string(p.GetLnAddr()[:10]) + "~"
}

// SendQueuedMessage adds the message to the queue to be sent to this peer.
// This queue is shared across all peers.
func (p *Peer) SendQueuedMessage(msg Message) error {
	return p.pmgr.queueMessageToPeer(p, msg)
}
