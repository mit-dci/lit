package lnp2p

import (
	"fmt"
	"sync"
	"time"

	"github.com/mit-dci/lit/logging"
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
	name         string
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

// PeerCallback is the function that's called when we got a response to a messsage.
// (inbound message, remote error) -> (delete, processing error)
type PeerCallback func(PeerCallMessage, error) (bool, error)

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
	dur := time.Duration(cip.timeout) * time.Millisecond
	if toh != nil {
		go (func() {
			select {
			case ok := <-cip.ok:
				if !ok {
					logging.Warnf("callrouter: expected true when skipping call timeout, got false") // TODO more detials
				}
			case <-time.After(dur):
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
		logging.Warnf("callrouter: peer %s tried to call a function we don't know %4x\n", peer.GetPrettyName(), id)
		peer.SendQueuedMessage(errmsg{
			id:     id,
			errmsg: "function ID not found",
		})
		return fmt.Errorf("unknown message ID")
	}

	m, err := fnh.parser(cmsg.body)
	if err != nil {
		peer.SendQueuedMessage(errmsg{
			id:     id,
			errmsg: fmt.Sprintf("parse error: %s", err.Error()),
		})
		return err
	}

	go (func() {
		res, err := fnh.callFunction(peer, m)
		if err != nil {
			// There was an error, just return it as a string.
			peer.SendQueuedMessage(errmsg{
				id:     id,
				errmsg: err.Error(),
			})
			return
		}

		// We're not supposed to have responses have a type of a regular fnid, but just warn on it.
		if res.FuncID() >= 0x00f0 && res.FuncID() != 0xffff {
			logging.Warnf("callrouter: response to %s returning with response fnid out of normal range (%4x)\n", fnh.name, res.FuncID())
		}

		// Still send it off.
		peer.SendQueuedMessage(respmsg{
			id:   id,
			body: res.Bytes(),
		})
	})()

	return nil
}

func (cr *CallRouter) processReturnResponse(peer *Peer, cmsg respmsg) error {

	cr.mtx.Lock()

	// Pick out the message from the in-progress call set.
	ip, ok := cr.inprog[cmsg.id]
	if !ok {
		cr.mtx.Unlock()
		return fmt.Errorf("callrouter: message ID doesn't match any in-progress call")
	}

	// now delete it from the map because it's returned.
	delete(cr.inprog, cmsg.id)

	cr.mtx.Unlock()

	// Tell the waiter for the timeout that it's all ok now.
	ip.ok <- true

	// Parse it, hopefully.
	pcm, err := ip.messageParser(cmsg.body)
	if err != nil {
		logging.Warnf("callrouter: problem parsing call %d: %s\n", cmsg.id, err.Error())
		return err
	}

	// Now actually invoke the call if it's successful.
	_, err = ip.callback(pcm, nil) // TODO support repeated returns later
	if err != nil {
		logging.Warnf("callrouter: problem when processing callback to %d: %s\n", cmsg.id, err.Error())
	}

	return nil

}

func (cr *CallRouter) processErrorResponse(peer *Peer, emsg errmsg) error {

	cr.mtx.Lock()

	// Pick out the message from the in-progress call set.
	ip, ok := cr.inprog[emsg.id]
	if !ok {
		cr.mtx.Unlock()
		return fmt.Errorf("callrouter: error message ID doesn't match any in-progress call")
	}

	// now delete it from the map because it's returned.
	delete(cr.inprog, emsg.id)

	cr.mtx.Unlock()

	// Tell the waiter for the timeout that it's all ok now.
	ip.ok <- true

	// Now actually invoke the call.
	logging.Warnf("callrouter: error on call %d to remote peer: %s\n", emsg.id, emsg.errmsg)
	_, err := ip.callback(nil, fmt.Errorf(emsg.errmsg)) // TODO support repeated returns later
	if err != nil {
		logging.Warnf("callrouter: problem when processing (error) callback to %d: %s\n", emsg.id, err.Error())
	}

	return nil

}

// DefineFunction sets up implementation for some P2P call.
func (cr *CallRouter) DefineFunction(fnid uint16, name string, mparser PcParser, impl PcFunction) error {
	cr.mtx.Lock()
	defer cr.mtx.Unlock()

	if fnid < 0x00f0 || fnid == 0xffff {
		return fmt.Errorf("callrouter: function ID must be >=0x00f0 and !=0xffff, special values are used for signalling")
	}

	// Just add it to the table, pretty easy.
	cr.fnhandlers[fnid] = peercallhandlerdef{
		name:         name,
		parser:       mparser,
		callFunction: impl,
	}

	return nil
}
