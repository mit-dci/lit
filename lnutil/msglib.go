package lnutil

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

// all the messages to and from peers look like this internally
type Msg interface {
	setData(...[]byte)
}
type LitMsg struct { // RETRACTED
	PeerIdx uint32
	ChanIdx uint32 // optional, may be 0
	MsgType uint8
	Data    []byte
}

type NewLitMsg interface {
	Peer() uint32
	//ChanIdx() uint32 // optional, may be 0
	MsgType() uint8
	Bytes() []byte
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

type DeltaSigMsg struct {
	PeerIdx   uint32
	Outpoint  wire.OutPoint
	Delta     int32
	Signature [64]byte
}

func NewDeltaSigMsg(peerid uint32, OP wire.OutPoint, DELTA int32, SIG [64]byte) *DeltaSigMsg {
	d := new(DeltaSigMsg)
	d.PeerIdx = peerid
	d.Outpoint = OP
	d.Delta = DELTA
	d.Signature = SIG
	return d
}

func (self *DeltaSigMsg) Bytes() []byte {
	var msg []byte
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, I32tB(self.Delta)...)
	msg = append(msg, self.Signature[:]...)
	return msg
}

func (self *DeltaSigMsg) Peer() uint32    { return self.PeerIdx }
func (self *DeltaSigMsg) MsgType() uint32 { return MSGID_DELTASIG }

//----------

type SigRevMsg struct {
	PeerIdx    uint32
	Outpoint   wire.OutPoint
	Signature  [64]byte
	Elk        chainhash.Hash
	N2ElkPoint [33]byte
}

func NewSigRev(peerid uint32, OP wire.OutPoint, SIG [64]byte, ELK chainhash.Hash, N2ELK [33]byte) *SigRevMsg {
	s := new(SigRevMsg)
	s.PeerIdx = peerid
	s.Outpoint = OP
	s.Signature = SIG
	s.Elk = ELK
	s.N2ElkPoint = N2ELK
	return s
}

func (self *SigRevMsg) Bytes() []byte {
	var msg []byte
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.Signature[:]...)
	msg = append(msg, self.Elk[:]...)
	msg = append(msg, self.N2ElkPoint[:]...)
	return msg
}

func (self *SigRevMsg) Peer() uint32    { return self.PeerIdx }
func (self *SigRevMsg) MsgType() uint32 { return MSGID_SIGREV }

//----------

type GapSigRevMsg struct {
	PeerIdx    uint32
	Outpoint   wire.OutPoint
	Signature  [64]byte
	Elk        chainhash.Hash
	N2ElkPoint [33]byte
}

func NewGapSigRev(peerid uint32, OP wire.OutPoint, SIG [64]byte, ELK chainhash.Hash, N2ELK [33]byte) *GapSigRevMsg {
	g := new(GapSigRevMsg)
	g.PeerIdx = peerid
	g.Outpoint = OP
	g.Signature = SIG
	g.Elk = ELK
	g.N2ElkPoint = N2ELK
	return g
}

func (self *GapSigRevMsg) Bytes() []byte {
	var msg []byte
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.Signature[:]...)
	msg = append(msg, self.Elk[:]...)
	msg = append(msg, self.N2ElkPoint[:]...)
	return msg
}

func (self *GapSigRevMsg) Peer() uint32    { return self.PeerIdx }
func (self *GapSigRevMsg) MsgType() uint32 { return MSGID_GAPSIGREV }

//----------

type RevMsg struct {
	PeerIdx    uint32
	Outpoint   wire.OutPoint
	Elk        chainhash.Hash
	N2ElkPoint [33]byte
}

func NewRevMsg(peerid uint32, OP wire.OutPoint, ELK chainhash.Hash, N2ELK [33]byte) *RevMsg {
	r := new(RevMsg)
	r.PeerIdx = peerid
	r.Outpoint = OP
	r.Elk = ELK
	r.N2ElkPoint = N2ELK
	return r
}

func (self *RevMsg) Bytes() []byte {
	var msg []byte
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.Elk[:]...)
	msg = append(msg, self.N2ElkPoint[:]...)
	return msg
}

func (self *RevMsg) Peer() uint32    { return self.PeerIdx }
func (self *RevMsg) MsgType() uint32 { return MSGID_REV }

//----------
