package qln

import (
	"log"
	"sync"

	"github.com/mit-dci/lit/lnutil"
)

var subscribedPeers []uint32
var subscriptionMtx sync.Mutex

func (nd *LitNode) SubscribePeerToUIEvents(peerIdx uint32) {
	subscriptionMtx.Lock()
	present := false
	for _, subscribedPeer := range subscribedPeers {
		if subscribedPeer == peerIdx {
			present = true
			break
		}
	}
	if !present {
		subscribedPeers = append(subscribedPeers, peerIdx)
	}

	log.Printf("UI Events now has %d subscribers\n", len(subscribedPeers))

	subscriptionMtx.Unlock()
}

func (nd *LitNode) UnsubscribePeerToUIEvents(peerIdx uint32) {
	subscriptionMtx.Lock()
	newSubscribers := []uint32{}
	for _, subscribedPeer := range subscribedPeers {
		if subscribedPeer != peerIdx {
			newSubscribers = append(newSubscribers, subscribedPeer)
		}
	}
	subscribedPeers = newSubscribers

	log.Printf("UI Events now has %d subscribers\n", len(subscribedPeers))

	subscriptionMtx.Unlock()
}

func (nd *LitNode) PublishUIEvent(msg *lnutil.UIEventMsg) {
	for _, subscribedPeer := range subscribedPeers {
		msg.PeerIdx = subscribedPeer
		log.Printf("Sending UI Event to peer %d\n", subscribedPeer)

		nd.OmniOut <- msg
	}
}
