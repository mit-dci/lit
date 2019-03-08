package qln

import (
	"sync"

	"github.com/mit-dci/lit/lnp2p"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
)

// LitMsgWrapperMessage is a wrapper type for adapting things to other things.
type LitMsgWrapperMessage struct {
	mtype  uint8
	rawbuf []byte
}

// Type .
func (wm LitMsgWrapperMessage) Type() uint8 {
	return wm.mtype
}

// Bytes .
func (wm LitMsgWrapperMessage) Bytes() []byte {
	return wm.rawbuf
}

func makeNeoOmniParser(mtype uint8) lnp2p.ParseFuncType {
	return func(buf []byte) (lnp2p.Message, error) {
		fullbuf := make([]byte, len(buf)+1)
		fullbuf[0] = mtype
		copy(fullbuf[1:], buf)
		return LitMsgWrapperMessage{mtype, fullbuf}, nil
	}
}

// Mostly taken from LNDCReader in msghandler.go, then horribly changed.
func makeNeoOmniHandler(nd *LitNode) lnp2p.HandleFuncType {

	inited := make(map[*RemotePeer]struct{})
	mtx := &sync.Mutex{}

	return func(p *lnp2p.Peer, m lnp2p.Message) error {

		// Idk how much locking I need to do here, but doing it across the whole
		// function probably wouldn't hurt.
		mtx.Lock()
		defer mtx.Unlock()

		var err error

		wm := m.(LitMsgWrapperMessage)
		rawbuf := wm.rawbuf
		msg, err := lnutil.LitMsgFromBytes(rawbuf, p.GetIdx())
		if err != nil {
			return err
		}

		peer := nd.PeerMap[p]

		if _, ok := inited[peer]; !ok {

			// init the qchan map thingy, this is quite inefficient
			err = nd.PopulateQchanMap(peer)
			if err != nil {
				logging.Errorf("error initing peer: %s", err.Error())
				return err
			}

			inited[peer] = struct{}{} // :thinking:

		}

		// TODO Fix this.  Also it's quite inefficient the way it's written at the moment.
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

		logging.Infof("chanIdx is %d, InProg is %d\n", chanIdx, nd.InProg.ChanIdx)

		if chanIdx != 0 {
			err = nd.PeerHandler(msg, peer.QCs[chanIdx], peer)
		} else {
			err = nd.PeerHandler(msg, nil, peer)
		}

		if err != nil {
			logging.Errorf("PeerHandler error with %d: %s\n", p.GetIdx(), err.Error())
		}

		return nil

	}
}
