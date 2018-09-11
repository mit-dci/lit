package lnp2p

import (
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnio"
)

// NewPeerEvent is fired when a new peer is registered.
type NewPeerEvent struct {
	Addr            lnio.LnAddr
	Peer            *Peer
	RemoteInitiated bool

	// TODO REFACTORING: Remove these
	RemotePub *btcec.PublicKey
	Conn      *lndc.Conn
}

// Name .
func (e NewPeerEvent) Name() string {
	return "lnp2p.peer.new"
}

// Flags .
func (e NewPeerEvent) Flags() uint8 {
	return eventbus.EFLAG_UNCANCELLABLE
}

// PeerDisconnectEvent is fired when a peer is disconnected.
type PeerDisconnectEvent struct {
	Peer   *Peer
	Reason string
}

// Name .
func (e PeerDisconnectEvent) Name() string {
	return "lnp2p.peer.disconnect"
}

// Flags .
func (e PeerDisconnectEvent) Flags() uint8 {
	return eventbus.EFLAG_UNCANCELLABLE
}

// NewListeningPortEvent .
type NewListeningPortEvent struct {
	ListenAddr string
}

// Name .
func (e NewListeningPortEvent) Name() string {
	return "lnp2p.listen.start"
}

// Flags .
func (e NewListeningPortEvent) Flags() uint8 {
	return eventbus.EFLAG_NORMAL
}

// StopListeningPortEvent .
type StopListeningPortEvent struct {
	ListenAddr string
	Reason     string
}

// Name .
func (e StopListeningPortEvent) Name() string {
	return "lnp2p.listen.stop"
}

// Flags .
func (e StopListeningPortEvent) Flags() uint8 {
	return eventbus.EFLAG_ASYNC
}
