package qln

import (
	"github.com/mit-dci/lit/eventbus"
)

// ChannelStateUpdateEvent is a struct for a channel state update event
type ChannelStateUpdateEvent struct {

	// Because a lot of these update events are similar, we can use the same
	// structure for all of them and have a dynamic name, which you wouldn't
	// normally do.
	Action string

	ChanIdx uint32
	State   *StatCom
}

// Name returns the name of the channel state update event
func (e ChannelStateUpdateEvent) Name() string {
	return "qln.chanupdate." + e.Action
}

// Flags returns the flags for the event
func (e ChannelStateUpdateEvent) Flags() uint8 {
	return eventbus.EFLAG_ASYNC
}

// FundEvent is a struct for a channel state update event
type FundEvent struct {

	// ChanIdx is the ChanIdx we get when we make sure that the fund is done
	ChanIdx uint32
	State   *InFlightFund
}

// Name returns the name of the channel state update event
func (e FundEvent) Name() string {
	return "qln.fundevent.fundchannel"
}

// Flags returns the flags for the event
func (e FundEvent) Flags() uint8 {
	return eventbus.EFLAG_ASYNC
}
