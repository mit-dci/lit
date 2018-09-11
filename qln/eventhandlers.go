package qln

import (
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lnp2p"
	"github.com/mit-dci/lit/lnutil"
	"log"
)

func makeTmpNewPeerHandler(nd *LitNode) func(eventbus.Event) eventbus.EventHandleResult {
	return func(e eventbus.Event) eventbus.EventHandleResult {

		ee := e.(lnp2p.NewPeerEvent)

		peerIdx := uint32(nd.PeerMan.GetPeerIdx(ee.Peer))

		rpeer := &RemotePeer{
			Idx:      peerIdx,
			Con:      ee.Conn,
			Nickname: ee.Peer.GetNickname(),
		}

		nd.PeerMapMtx.Lock()
		nd.RemoteMtx.Lock()

		nd.RemoteCons[peerIdx] = rpeer
		nd.PeerMap[ee.Peer] = rpeer

		nd.RemoteMtx.Unlock()
		nd.PeerMapMtx.Unlock()

		return eventbus.EHANDLE_OK
	}
}

// Mostly taken from LNDCReader in msghandler.go, then horribly changed.
func makeTmpMsgHandler(nd *LitNode) func(eventbus.Event) eventbus.EventHandleResult {

	/*
	 * Bless me father for I have sinned...
	 */

	var err error

	// this is to keep track of peers to see which ones we need to do late-setup for
	loaded := make([]*lnp2p.Peer, 0)

	// idk what this is lol
	var opArrs map[*lnp2p.Peer][36]byte

	return func(e eventbus.Event) eventbus.EventHandleResult {
		ev := e.(lnp2p.NetMessageRecvEvent)

		msg := ev.Msg
		rawbuf := ev.Rawbuf
		npeer := ev.Peer
		peer := nd.PeerMap[npeer]

		// Check to see if we're done the prep for this one already.
		found := false
		for _, p := range loaded {
			if ev.Peer == p {
				found = true
			}
		}

		// If not, then do so.
		if !found {
			// init the qchan map thingy
			err = nd.PopulateQchanMap(peer)
			if err != nil {
				log.Printf("error initing peer: %s", err.Error())
				return eventbus.EHANDLE_OK
			}

			// make a local map of outpoints to channel indexes
			// iterate through all this peer's channels to extract outpoints
			peer.OpMap = make(map[[36]byte]uint32)
			for _, q := range peer.QCs {
				opArrs[npeer] = lnutil.OutPointToBytes(q.Op)
				peer.OpMap[opArrs[npeer]] = q.Idx()
			}
		}

		var chanIdx uint32
		chanIdx = 0
		if len(rawbuf) > 38 {
			opArr := opArrs[npeer]
			copy(opArr[:], rawbuf[1:37])
			chanCheck, ok := peer.OpMap[opArr]
			if ok {
				chanIdx = chanCheck
			}
		}

		log.Printf("chanIdx is %x\n", chanIdx)

		if chanIdx != 0 {
			err = nd.PeerHandler(*msg, peer.QCs[chanIdx], peer)
		} else {
			err = nd.PeerHandler(*msg, nil, peer)
		}

		if err != nil {
			log.Printf("PeerHandler error with %d: %s\n", npeer.GetIdx(), err.Error())
		}

		return eventbus.EHANDLE_OK
	}
}
