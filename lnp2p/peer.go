package lnp2p

import (
	"fmt"

	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lndc"
)

// A Peer is a remote client that's somehow connected to us.
type Peer struct {
	lnaddr   lncore.LnAddr
	nickname *string
	conn     *lndc.Conn
	idpubkey pubkey

	idx  *uint32 // deprecated
	pmgr *PeerManager
}

// GetIdx is a compatibility function.
func (p *Peer) GetIdx() uint32 {
	if p.idx == nil {
		return 0
	}
	return *p.idx
}

// GetNickname returns the nickname, or an empty string if unset.
func (p *Peer) GetNickname() string {
	if p.nickname == nil {
		return ""
	}
	return *p.nickname
}

// SetNickname sets the peer's nickname.
func (p *Peer) SetNickname(name string) {
	p.nickname = &name
	if name == "" {
		p.nickname = nil
	}
}

// GetLnAddr returns the lightning network address for this peer.
func (p *Peer) GetLnAddr() lncore.LnAddr {
	return p.lnaddr
}

// GetRemoteAddr does something.
func (p *Peer) GetRemoteAddr() string {
	return p.conn.RemoteAddr().String()
}

// GetPubkey gets the public key for the user.
func (p *Peer) GetPubkey() koblitz.PublicKey {
	return *p.idpubkey
}

const prettyLnAddrPrefixLen = 10

// GetPrettyName returns a more human-readable name, such as the nickname if
// available or a trucated version of the LN address otherwise.
func (p *Peer) GetPrettyName() string {
	if p.nickname != nil {
		return *p.nickname
	}

	return string(p.GetLnAddr()[:prettyLnAddrPrefixLen]) + "~"
}

// SendQueuedMessage adds the message to the queue to be sent to this peer.
// This queue is shared across all peers.
func (p *Peer) SendQueuedMessage(msg Message) error {
	return p.pmgr.queueMessageToPeer(p, msg, nil)
}

// SendImmediateMessage adds a message to the queue but waits for the message to
// be sent before returning, also returning errors that might have occurred when
// sending the message, like the peer disconnecting.
func (p *Peer) SendImmediateMessage(msg Message) error {
	var err error
	errchan := make(chan error)

	// Send it to the queue, as above.
	err = p.pmgr.queueMessageToPeer(p, msg, &errchan)
	if err != nil {
		return err
	}

	// Catches errors if there are any.
	err = <-errchan
	if err != nil {
		return err
	}

	return nil
}

// IntoPeerInfo generates the PeerInfo DB struct for the Peer.
func (p *Peer) IntoPeerInfo() lncore.PeerInfo {
	var raddr string
	if p.conn != nil {
		raddr = p.conn.RemoteAddr().String()
	}
	return lncore.PeerInfo{
		LnAddr:   &p.lnaddr,
		Nickname: p.nickname,
		NetAddr:  &raddr,
	}
}

const defaulttimeout = 60000 // 1 minute

// InvokeAsyncCall sends a call to the peer, returning immediately and passing
// the response to the callback.
func (p *Peer) InvokeAsyncCall(arg PeerCallMessage, callback func(PeerCallMessage, error) (bool, error)) error {
	return p.pmgr.crouter.initInvokeCall(p, defaulttimeout, arg, PeerCallback(callback), nil)
}

// InvokeAsyncCallTimeout sends a call to the peer, returning immediately and passing
// the response to the callback, with a timeout condition.
func (p *Peer) InvokeAsyncCallTimeout(arg PeerCallMessage, tout uint64, callback func(PeerCallMessage, error) (bool, error), tohandler func()) error {
	return p.pmgr.crouter.initInvokeCall(p, tout, arg, PeerCallback(callback), PeerTimeoutHandler(tohandler))
}

// InvokeBlockingCall sends a call to the peer, waiting for a response or
// waiting until a timeout.
func (p *Peer) InvokeBlockingCall(arg PeerCallMessage, timeout uint64) (PeerCallMessage, error) {

	// Set up some structuring to wait until the remote call returns or times out.
	resc := make(chan PeerCallMessage)
	errc := make(chan error)

	cb := func(pcm PeerCallMessage, err error) (bool, error) {
		// If we got a good message, then just pass it along.
		if pcm != nil {
			resc <- pcm
		}

		// Basically, just pass through whichever thing was an error.
		if err != nil {
			errc <- err
		}

		// This should always be fine because the "error" here is only for the context
		// of the actual *handling* of the function.
		return true, nil
	}

	// Very simple, and the actual work of the timeout is handled for us already.
	th := func() {
		errc <- fmt.Errorf("peer call timed out")
	}

	// This is where all the cool stuff happens and sends off the call to the remote peer.
	err := p.pmgr.crouter.initInvokeCall(p, timeout, arg, PeerCallback(cb), PeerTimeoutHandler(th))
	if err != nil {
		return nil, err
	}

	// Now this is where we figure out what we're actually returning.
	select {
	case r := <-resc:
		return r, nil
	case t := <-errc:
		return nil, t
	}
}
