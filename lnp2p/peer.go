package lnp2p

import (
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lndc"
)

// A Peer is a remote client that's somehow connected to us.
type Peer struct {
	lnaddr   lncore.LnAddr
	nickname *string
	conn     *lndc.Conn
	idpubkey pubkey

	alive bool

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

// SetNickname sets the peer's nickname.
func (p *Peer) SetNickname(name string) {
	p.nickname = &name
	if name == "" {
		p.nickname = nil
	}
}

// GetLnAddr returns the lightning network address for this peer.
func (p *Peer) GetLnAddr() lncore.LnAddr {
	return p.lnaddr
}

// GetRemoteAddr does something.
func (p *Peer) GetRemoteAddr() string {
	return p.conn.RemoteAddr().String()
}

// GetPubkey gets the public key for the user.
func (p *Peer) GetPubkey() koblitz.PublicKey {
	return *p.idpubkey
}

const prettyLnAddrPrefixLen = 10

// GetPrettyName returns a more human-readable name, such as the nickname if
// available or a trucated version of the LN address otherwise.
func (p *Peer) GetPrettyName() string {
	if p.nickname != nil {
		return *p.nickname
	}

	return string(p.GetLnAddr()[:prettyLnAddrPrefixLen]) + "~"
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
		LnAddr:   &p.lnaddr,
		Nickname: p.nickname,
		NetAddr:  &raddr,
	}
}

// IsAlive returns whethere or not this particular peer is still valid
func (p *Peer) IsAlive() bool {
	return p.alive
}
