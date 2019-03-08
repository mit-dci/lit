package qln

import (
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lnp2p"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
)

func makeTmpNewPeerHandler(nd *LitNode) func(eventbus.Event) eventbus.EventHandleResult {
	return func(e eventbus.Event) eventbus.EventHandleResult {

		ee := e.(lnp2p.NewPeerEvent)

		peerIdx := uint32(nd.PeerMan.GetPeerIdx(ee.Peer))

		logging.Debugf("creating new fake RemotePeer %d\n", peerIdx)

		rpeer := &RemotePeer{
			Idx:      peerIdx,
			Con:      ee.Conn,
			Nickname: ee.Peer.GetNickname(),
			QCs:      make(map[uint32]*Qchan),
		}

		nd.PeerMapMtx.Lock()
		nd.RemoteMtx.Lock()

		nd.RemoteCons[peerIdx] = rpeer
		nd.PeerMap[ee.Peer] = rpeer

		nd.RemoteMtx.Unlock()
		nd.PeerMapMtx.Unlock()

		// populate things
		nd.PopulateQchanMap(rpeer)

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

func makeTmpDisconnectPeerHandler(nd *LitNode) func(eventbus.Event) eventbus.EventHandleResult {
	return func(e eventbus.Event) eventbus.EventHandleResult {
		ee := e.(lnp2p.PeerDisconnectEvent)
		rpeer := nd.PeerMap[ee.Peer]

		nd.RemoteMtx.Lock()
		delete(nd.RemoteCons, rpeer.Idx)
		delete(nd.PeerMap, ee.Peer)
		nd.RemoteMtx.Unlock()

		return eventbus.EHANDLE_OK
	}
}

func makeTmpSigProofHandler(nd *LitNode) func(eventbus.Event) eventbus.EventHandleResult {
	return func(e eventbus.Event) eventbus.EventHandleResult {
		ee, ok := e.(ChannelStateUpdateEvent)
		if !ok {
			logging.Errorf("Wrong type of event, why are you publishing this")
			// I think this is the right way to cancel the event?
			return eventbus.EHANDLE_CANCEL
		}

		logging.Infof("Sig proof!")

		logging.Infof("Channel succeeded: %t, MyAmt: %d", !ee.State.Failed, ee.State.MyAmt)

		logging.Infof("got a fund event: %d", ee.ChanIdx)
		return eventbus.EHANDLE_OK
	}
}
