package lnp2p

import (
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnio"
)

// A Peer is a remote client that's somehow connected to us.
type Peer struct {
	lnaddr   lnio.LnAddr
	nickname *string
	conn     *lndc.Conn
	idpubkey pubkey

	idx *uint32 // deprecated
}

// GetIdx is a compatibility function.
func (p *Peer) GetIdx() uint32 {
	if p.idx == nil {
		return 0xffffffff
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
