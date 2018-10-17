package lnp2p

import (
	"time"

	"github.com/mit-dci/lit/lnutil"
)

const pingperiod = 30

// we should change this at some point to ping *all* the peers from one goroutine
func pingPeer(p *Peer) {

	for {
		select {
		case v := <-p.ping:
			if !v { // this is how we know we've disconnected
				break
			}
		case <-time.After(pingperiod * time.Second):
			{
				// TODO This could be better, use the pingpong P2P call.
				p.SendQueuedMessage(tmpwrapper{
					buf: lnutil.NewChatMsg(p.GetIdx(), "").Bytes()[1:],
				})
			}
		}
	}

}

// TEMP delet this
type tmpwrapper struct {
	buf []byte
}

func (w tmpwrapper) Bytes() []byte {
	return w.buf
}

func (tmpwrapper) Type() uint8 {
	return lnutil.MSGID_TEXTCHAT
}
