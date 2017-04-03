package lnutil

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

// all the messages to and from peers look like this internally
type LitMsg struct {
	PeerIdx uint32
	ChanIdx uint32 // optional, may be 0
	MsgType uint8
	Data    []byte
}

const (
	MSGID_POINTREQ  = 0x30
	MSGID_POINTRESP = 0x31
	MSGID_CHANDESC  = 0x32
	MSGID_CHANACK   = 0x33
	MSGID_SIGPROOF  = 0x34

	MSGID_CLOSEREQ  = 0x40 // close channel
	MSGID_CLOSERESP = 0x41

	MSGID_TEXTCHAT = 0x60 // send a text message

	MSGID_DELTASIG  = 0x70 // pushing funds in channel; request to send
	MSGID_SIGREV    = 0x72 // pulling funds; signing new state and revoking old
	MSGID_GAPSIGREV = 0x73 // resolving collision
	MSGID_REV       = 0x74 // pushing funds; revoking previous channel state

	MSGID_FWDMSG     = 0x20
	MSGID_FWDAUTHREQ = 0x21

	MSGID_SELFPUSH = 0x80
)

type RevMsg struct {
	outpoit   wire.OutPoint
	revelk    chainhash.Hash // 32 bytes
	nextpoint [33]byte
}

//	msg = append(msg, opArr[:]...)
//	msg = append(msg, elk[:]...)
//	msg = append(msg, n2ElkPoint[:]...)
