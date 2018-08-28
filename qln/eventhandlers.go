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

		// make a local map of outpoints to channel indexes
		// iterate through all this peer's channels to extract outpoints
		rpeer.OpMap = make(map[[36]byte]uint32)
		for _, q := range rpeer.QCs {
			opArr := lnutil.OutPointToBytes(q.Op)
			rpeer.OpMap[opArr] = q.Idx()
		}

		return eventbus.EHANDLE_OK
	}
}

// Mostly taken from LNDCReader in msghandler.go, then horribly changed.
func makeTmpMsgHandler(nd *LitNode) func(eventbus.Event) eventbus.EventHandleResult {

	/*
	 * Bless me father for I have sinned...
	 */

	return func(e eventbus.Event) eventbus.EventHandleResult {
		ev := e.(lnp2p.NetMessageRecvEvent)

		msg := ev.Msg
		rawbuf := ev.Rawbuf
		npeer := ev.Peer
		peer := nd.PeerMap[npeer]

		var err error

		// init the qchan map thingy, this is quite inefficient
		err = nd.PopulateQchanMap(peer)
		if err != nil {
			log.Printf("error initing peer: %s", err.Error())
			return eventbus.EHANDLE_OK
		}

		// TODO Remove this.  Also it's quite inefficient the way it's written at the moment.
		var chanIdx uint32
		chanIdx = 0
		if len(rawbuf) > 38 {
			var opArr [36]byte
			for _, q := range peer.QCs {
				b := lnutil.OutPointToBytes(q.Op)
				peer.OpMap[b] = q.Idx()
			}
			copy(opArr[:], rawbuf[1:37]) // yay for magic numbers /s
			chanCheck, ok := peer.OpMap[opArr]
			if ok {
				chanIdx = chanCheck
			}
		}

		log.Printf("chanIdx is %d, InProg is %d\n", chanIdx, nd.InProg.ChanIdx)

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
