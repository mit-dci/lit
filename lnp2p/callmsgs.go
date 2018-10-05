package lnp2p

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

// ParseCallMessage parses a callmsg from bytes.
func ParseCallMessage(buf []byte) (Message, error) {

	var err error
	res := callmsg{}
	r := bytes.NewBuffer(buf)

	// Read the message ID
	err = binary.Read(r, binary.BigEndian, &res.id)
	if err != nil {
		return nil, err
	}

	// Read the fnid
	err = binary.Read(r, binary.BigEndian, &res.funcID)
	if err != nil {
		return nil, err
	}

	// Now do a dance to read the message body.
	var blen uint32
	err = binary.Read(r, binary.BigEndian, &blen)
	if err != nil {
		return nil, err
	}

	body := make([]byte, blen)
	n, err := r.Read(body)
	if err != nil {
		return nil, err
	}

	if n != int(blen) {
		return nil, fmt.Errorf("unexpected EOF (parse call)")
	}

	res.body = body

	return res, nil

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

// ParseRespMessage parses a respmsg from bytes.
func ParseRespMessage(buf []byte) (Message, error) {

	var err error
	res := respmsg{}
	r := bytes.NewBuffer(buf)

	// Read the message (response) ID
	err = binary.Read(r, binary.BigEndian, &res.id)
	if err != nil {
		return nil, err
	}

	// Read the body with some magic.
	var blen uint32
	err = binary.Read(r, binary.BigEndian, &blen)
	if err != nil {
		return nil, err
	}

	body := make([]byte, blen)
	n, err := r.Read(body)
	if err != nil {
		return nil, err
	}

	if n != int(blen) {
		return nil, fmt.Errorf("unexpected EOF (parse resp)")
	}

	res.body = body

	return res, nil

}

func (respmsg) Type() uint8 {
	return MsgidResponse
}

type errmsg struct {
	id  uint32
	msg string // maybe this should be richer?  does it matter?
}

func (m errmsg) Bytes() []byte {

	w := bytes.Buffer{}

	// Write the call ID
	binary.Write(&w, binary.BigEndian, m.id)

	// Write the message len.
	binary.Write(&w, binary.BigEndian, uint16(len(m.msg)))

	// Write the message.
	w.Write([]byte(m.msg))

	return w.Bytes()

}

// ParseErrMessage parses a errmsg from bytes.
func ParseErrMessage(buf []byte) (Message, error) {

	var err error
	res := errmsg{}
	r := bytes.NewBuffer(buf)

	// Read the (response) ID
	err = binary.Read(r, binary.BigEndian, &res.id)
	if err != nil {
		return nil, err
	}

	// More magic to read the error message.
	var mlen uint16
	err = binary.Read(r, binary.BigEndian, &mlen)
	if err != nil {
		return nil, err
	}

	body := make([]byte, mlen)
	n, err := r.Read(body)
	if err != nil {
		return nil, err
	}

	if n != int(mlen) {
		return nil, fmt.Errorf("unexpected EOF (parse err)")
	}

	res.msg = string(body) // apparently this "just werks"

	return res, nil

}

func (errmsg) Type() uint8 {
	return MsgidError
}

// PcPing .
type PcPing struct {
	buf []byte
}

func NewPingMsg(buf []byte) PcPing {
	return PcPing{buf}
}

// Bytes .
func (p PcPing) Bytes() []byte {
	return p.buf
}

// FuncID .
func (PcPing) FuncID() uint16 {
	return 0xff00
}

// PcPong .
type PcPong struct {
	buf []byte
}

// Bytes .
func (p PcPong) Bytes() []byte {
	return p.buf
}

// FuncID .
func (PcPong) FuncID() uint16 {
	return 0xffff
}

func (p PcPong) GetBody() []byte {
	return p.buf
}

// ParsePing is for deserializing a buffer into a ping message.
func ParsePing(buf []byte) (PeerCallMessage, error) {
	return PcPing{buf}, nil
}

// ParsePong is for deserializing a buffer into a pong message.
func ParsePong(buf []byte) (PeerCallMessage, error) {
	return PcPong{buf}, nil
}
