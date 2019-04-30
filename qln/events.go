package qln

import (
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/eventbus"
)

// ChannelStateUpdateEvent is a struct for a channel state update event
type ChannelStateUpdateEvent struct {

	// Because a lot of these update events are similar, we can use the same
	// structure for all of them and have a dynamic name, which you wouldn't
	// normally do.
	Action string

	// We include ChanIdx so we can do stuff internally if implemented, and we include State because it's a state
	// update event, so we'd like to know the new state.
	ChanIdx uint32
	State   *StatCom

	// in case an external application is using this and needs the public key for some reason.
	TheirPub koblitz.PublicKey

	// We need to know which coin this was for
	CoinType uint32
}

// Name returns the name of the channel state update event
func (e ChannelStateUpdateEvent) Name() string {
	return "qln.chanupdate." + e.Action
}

// Flags returns the flags for the event
func (e ChannelStateUpdateEvent) Flags() uint8 {
	return eventbus.EFLAG_ASYNC
}
