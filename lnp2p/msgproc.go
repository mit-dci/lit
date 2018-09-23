package lnp2p

import (
	"fmt"
	"sync"

	"github.com/mit-dci/lit/logging"
)

// ParseFuncType is the type of a Message parser function.
type ParseFuncType func([]byte) (Message, error)

// HandleFuncType is the type of a Message handler function, handling for a particular peer.
type HandleFuncType func(*Peer, Message) error

type messagehandler struct {
	parseFunc  ParseFuncType
	handleFunc HandleFuncType
}

// MessageProcessor is can be given messages and figures out how to parse them and which function to call.
type MessageProcessor struct {
	handlers [256]*messagehandler

	// This is for keeping track of if we're making changes to the message
	// processor.  Locking and unlocking is slow and shouldn't be necessary when
	// handling messages, but we don't want to be handling messages while making
	// changes to the handlerset.
	// TODO Evaluate if this mutex is even necessary?
	active bool
	actmtx *sync.Mutex
}

// NewMessageProcessor processes messages coming in from over the network.
func NewMessageProcessor() MessageProcessor {
	return MessageProcessor{
		handlers: [256]*messagehandler{},
		active:   false,
		actmtx:   &sync.Mutex{},
	}
}

// DefineMessage defines processing routines for a particular message type.
func (mp *MessageProcessor) DefineMessage(mtype uint8, pfunc ParseFuncType, hfunc HandleFuncType) {
	mp.actmtx.Lock()
	act := mp.active
	mp.active = false

	// Actually set the handler.
	mp.handlers[mtype] = &messagehandler{
		parseFunc:  pfunc,
		handleFunc: hfunc,
	}

	logging.Debugf("msgproc: Setup message type %x\n", mtype)

	mp.active = act
	mp.actmtx.Unlock()
}

// Activate sets the MessageProcessor to be "active"
func (mp *MessageProcessor) Activate() {
	mp.active = true
}

// IsActive returns the activiation state for the MessageProcessor.
func (mp *MessageProcessor) IsActive() bool {
	return mp.active
}

// HandleMessage runs through the normal handling procedure for the message, returning any errors.
func (mp *MessageProcessor) HandleMessage(peer *Peer, buf []byte) error {
	if !mp.active {
		return fmt.Errorf("message processor not active, retry later")
	}

	var err error

	// First see if we have handlers defined for this message type.
	mtype := buf[0]
	h := mp.handlers[mtype]
	if h == nil {
		return fmt.Errorf("no handler found for messasge of type %x", mtype)
	}

	// Parse the message.
	parsed, err := h.parseFunc(buf[1:])
	if err != nil {
		logging.Warnf("msgproc: Malformed message of type %x from peer %s\n", mtype, peer.GetPrettyName())
		return err
	}

	// If ok, then actually handle the message.
	err = h.handleFunc(peer, parsed)
	if err != nil {
		return err
	}

	return nil
}
