package lnp2p

import (
	"github.com/mit-dci/lit/eventbus"
)

func makePeerDisconnectHandler(pm *PeerManager) func(eventbus.Event) eventbus.EventHandleResult {
	return func(event eventbus.Event) eventbus.EventHandleResult {

		dce, ok := event.(PeerDisconnectEvent)
		if !ok {
			return eventbus.EHANDLE_OK
		}

		// if it was a timeout, then just try to reconnect
		if dce.Reason == "timeout" {
			pm.TryConnectAddress(string(dce.Peer.GetLnAddr()), nil) // TODO NetSettings
		}

		return eventbus.EHANDLE_OK
	}
}
