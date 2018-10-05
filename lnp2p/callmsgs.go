package lnp2p

import (
	"bytes"
	"encoding/binary"
)

const (

	// MsgidCall is the id for calling.
	MsgidCall = 0xC0

	// MsgidResponse is the id for responses.
	MsgidResponse = 0xC1

	// MsgidError is for when there was an error processing the call.
	MsgidError = 0xC2
)

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

type errmsg struct {
	id     uint32
	errmsg string // maybe this should be richer?  does it matter?
}

func (m errmsg) Bytes() []byte {

	w := bytes.Buffer{}

	// Write the call ID
	binary.Write(&w, binary.BigEndian, m.id)

	// Write the message len.
	binary.Write(&w, binary.BigEndian, uint16(len(m.errmsg)))

	// Write the message.
	w.WriteString(m.errmsg)

	return w.Bytes()

}

func (errmsg) Type() uint8 {
	return MsgidError
}
