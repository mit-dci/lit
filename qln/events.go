package qln

import (
	"github.com/mit-dci/lit/eventbus"
)

type ChannelStateUpdateEvent struct {

	// Because a lot of these update events are similar, we can use the same
	// structure for all of them and have a dynamic name, which you wouldn't
	// normally do.
	action string

	chanIdx uint32
	state   *StatCom
}

func (e ChannelStateUpdateEvent) Name() string {
	return "qln.chanupdate." + e.action
}

func (e ChannelStateUpdateEvent) Flags() uint8 {
	return eventbus.EFLAG_ASYNC
}
