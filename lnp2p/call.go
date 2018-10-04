package lnp2p

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/mit-dci/lit/logging"
)

const (

	// MsgidCall is the id for calling.
	MsgidCall = 0xC0

	// MsgidResponse is the id for responses.
	MsgidResponse = 0xC1
)

// PeerCallMessage is a serialized version of what the remote peer send.
type PeerCallMessage interface {
	Bytes() []byte
	FuncID() uint16
}

// PcParser is a function that parses P2P message call bodies.
type PcParser func([]byte) (PeerCallMessage, error)

// PcFunction is the implementation of a P2P call.
// (inbound message) -> (outbound message, error)
type PcFunction func(*Peer, PeerCallMessage) (PeerCallMessage, error) // should we make the result have a different type?

type peercallhandlerdef struct {
	parser       PcParser
	callFunction PcFunction
}

// CallRouter is the thing that handles doing p2p calls.
type CallRouter struct {

	// handlers
	fnhandlers map[uint16]peercallhandlerdef

	// ID of the next call to be invoked.  Starts counting at 0.
	currentMsgID uint32

	// the in-progress calls
	inprog map[uint32]callinprogress

	// mutex for everything, not needed for everything though
	mtx sync.Mutex
}

// NewCallRouter creates a new empty CallRouter with default settings.
func NewCallRouter() CallRouter {
	return CallRouter{
		currentMsgID: 0,
		inprog:       make(map[uint32]callinprogress),
		mtx:          sync.Mutex{},
	}
}

type callmsg struct {
	id     uint32
	funcID uint16
	body   []byte
}

func (m callmsg) Bytes() []byte {

	w := bytes.Buffer{}

	// Write the call ID and message type
	binary.Write(&w, binary.BigEndian, m.id)
	binary.Write(&w, binary.BigEndian, m.funcID)

	// Write the body len
	blen := uint32(len(m.body))
	binary.Write(&w, binary.BigEndian, blen)

	// Write the body.
	w.Write(m.body)

	return w.Bytes()

}

func (callmsg) Type() uint8 {
	return MsgidCall
}

type respmsg struct {
	id   uint32
	body []byte
}

func (m respmsg) Bytes() []byte {

	w := bytes.Buffer{}

	// Write the call ID
	binary.Write(&w, binary.BigEndian, m.id)

	// Write the body len
	blen := uint32(len(m.body))
	binary.Write(&w, binary.BigEndian, blen)

	// Write the body.
	w.Write(m.body)

	return w.Bytes()

}

func (respmsg) Type() uint8 {
	return MsgidResponse
}

// PeerCallback is the function that's called when we got a response to a messsage.
type PeerCallback func(PeerCallMessage, error, error) (bool, error)

// PeerTimeoutHandler is the function called when we timed out.
type PeerTimeoutHandler func()

type callinprogress struct {

	// millis time for when it started
	started uint64

	// how many millis to wait until giving up
	timeout uint64

	// if a true is sent on this channel then we don't have to invoke the timeout handler
	ok chan bool

	// parses the bytes from the remote peer
	messageParser func([]byte) (PeerCallMessage, error)

	// what we pass the parsed message to
	// if the bool in the reponse is false, then don't consider the call "complete"
	// (inbound message, parse error, network error) -> (delete, processing error)
	callback PeerCallback

	// if we exceed the timeout and haven't been sent any messages then run this before removing the entry
	timeoutHandler PeerTimeoutHandler
}

func makeTimestamp() uint64 {
	return uint64(time.Now().UnixNano()) / uint64(time.Millisecond)
}

func (cr *CallRouter) initInvokeCall(peer *Peer, timeout uint64, call PeerCallMessage, cb PeerCallback, toh PeerTimeoutHandler) error {

	cr.mtx.Lock()
	cid := cr.currentMsgID
	cr.currentMsgID++
	cr.mtx.Unlock()

	// Create the record for the in-progres call.
	cip := callinprogress{
		started:        makeTimestamp(),
		timeout:        timeout,
		ok:             make(chan bool),
		messageParser:  nil, // TODO
		callback:       cb,
		timeoutHandler: toh,
	}

	// Create the actual message to send off to the remote peer.
	m := callmsg{
		id:     cid,
		funcID: call.FuncID(), // TODO
		body:   call.Bytes(),
	}

	// Install the call record.
	cr.mtx.Lock()
	cr.inprog[cid] = cip
	cr.mtx.Unlock()

	// Spin off a thread to wait for the timeout.
	if toh != nil {
		go (func() {
			select {
			case ok := <-cip.ok:
				if !ok {
					logging.Warnf("something bad happened!") // TODO more detials
				}
			case <-time.After(time.Duration(cip.timeout) * time.Millisecond):
				toh()
			}
		})()
	}

	// Now actually send the message and return.
	return peer.SendQueuedMessage(m)

}

func (cr *CallRouter) processCall(peer *Peer, msg Message) error {

	cmsg, ok := msg.(callmsg)
	if !ok {
		return fmt.Errorf("bad message type") // is this really necessary?
	}

	id := cmsg.id

	// Figure out which function handler to use.
	cr.mtx.Lock()
	fnh, ok := cr.fnhandlers[cmsg.funcID]
	cr.mtx.Unlock()

	// Check that we actually found one.
	if !ok {
		// TODO Figure out what to do if we don't have a handler set up.
	}

	m, err := fnh.parser(cmsg.body)
	if err != nil {
		return err // TODO Better error handling for this.
	}

	go (func() {
		res, err := fnh.callFunction(peer, m)
		if err != nil {
			// TODO
		}

		if res.FuncID() < 0x00f0 || res.FuncID() == 0xffff {
			// Actually send off the response here.
			peer.SendQueuedMessage(respmsg{
				id:   id,
				body: res.Bytes(),
			})
		} else {
			// It thinks it's some other call, that's wrong.
			// TODO
		}
	})()

	return nil
}

func (cr *CallRouter) processResponse(peer *Peer, msg Message) error {

	cmsg, ok := msg.(respmsg)
	if !ok {
		return fmt.Errorf("bad message type")
	}

	cr.mtx.Lock()
	_, ok = cr.inprog[cmsg.id]
	if !ok {
		return fmt.Errorf("message ID doesn't match any in-progress call")
	}
	cr.mtx.Unlock()

	// TODO the rest of the handling

	return nil
}

// DefineFunction sets up implementation for some P2P call.
func (cr *CallRouter) DefineFunction(fnid uint16, mparser PcParser, impl PcFunction) error {
	cr.mtx.Lock()
	defer cr.mtx.Unlock()

	if fnid < 0x00f0 || fnid == 0xffff {
		return fmt.Errorf("function ID must be >=0x00f0 and !=0xffff, special values are used for signalling")
	}

	// Just add it to the table, pretty easy.
	cr.fnhandlers[fnid] = peercallhandlerdef{
		parser:       mparser,
		callFunction: impl,
	}

	return nil
}
