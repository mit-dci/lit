package lnutil

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/adiabat/btcd/chaincfg/chainhash"
	"github.com/adiabat/btcd/wire"
	"github.com/mit-dci/lit/portxo"
)

//id numbers for messages, semi-arbitrary
const (
	MSGID_TEXTCHAT = 0x00 // send a text message

	//Channel creation messages
	MSGID_POINTREQ  = 0x10
	MSGID_POINTRESP = 0x11
	MSGID_CHANDESC  = 0x12
	MSGID_CHANACK   = 0x13
	MSGID_SIGPROOF  = 0x14

	//Channel destruction messages
	MSGID_CLOSEREQ  = 0x20 // close channel
	MSGID_CLOSERESP = 0x21

	//Push Pull Messages
	MSGID_DELTASIG  = 0x30 // pushing funds in channel; request to send
	MSGID_SIGREV    = 0x31 // pulling funds; signing new state and revoking old
	MSGID_GAPSIGREV = 0x32 // resolving collision
	MSGID_REV       = 0x33 // pushing funds; revoking previous channel state

	//not implemented
	MSGID_FWDMSG     = 0x40
	MSGID_FWDAUTHREQ = 0x41

	//not implemented
	MSGID_SELFPUSH = 0x50

	//Tower Messages
	MSGID_WATCH_DESC     = 0x60 // desc describes a new channel
	MSGID_WATCH_STATEMSG = 0x61 // commsg is a single state in the channel
	MSGID_WATCH_DELETE   = 0x62 // Watch_clear marks a channel as ok to delete.  No further updates possible.

	//Routing messages
	MSGID_LINK_DESC = 0x70 // Describes a new channel for routing

	//Dual funding messages
	MSGID_DUALFUNDINGREQ     = 0x80 // Requests funding details (UTXOs, Change address, Pubkey), including our own details and amount needed.
	MSGID_DUALFUNDINGRESP    = 0x81 // Responds with funding details
	MSGID_DUALFUNDINGDECL    = 0x82 // Declines the funding request
	MSGID_DUALFUNDINGSIGREQ  = 0x83 // Requests signatures for the funding TX while transmitting our own.
	MSGID_DUALFUNDINGSIGRESP = 0x84 // Responds with funding TX signatures
)

//interface that all messages follow, for easy use
type LitMsg interface {
	Peer() uint32   //return PeerIdx
	MsgType() uint8 //returns Message Type (see constants above)
	Bytes() []byte  //returns data of message as []byte with the MsgType() preceding it
}

func LitMsgEqual(msg LitMsg, msg2 LitMsg) bool {
	if msg.Peer() != msg2.Peer() || msg.MsgType() != msg2.MsgType() || !bytes.Equal(msg.Bytes(), msg2.Bytes()) {
		return false
	}
	return true
}

//method for finding what type of message a generic []byte is
func LitMsgFromBytes(b []byte, peerid uint32) (LitMsg, error) {
	if len(b) < 1 {
		return nil, fmt.Errorf("The byte slice sent is empty")
	}
	msgType := b[0] // first byte signifies what type of message is

	switch msgType {
	case MSGID_TEXTCHAT:
		return NewChatMsgFromBytes(b, peerid)
	case MSGID_POINTREQ:
		return NewPointReqMsgFromBytes(b, peerid)
	case MSGID_POINTRESP:
		return NewPointRespMsgFromBytes(b, peerid)
	case MSGID_CHANDESC:
		return NewChanDescMsgFromBytes(b, peerid)
	case MSGID_CHANACK:
		return NewChanAckMsgFromBytes(b, peerid)
	case MSGID_SIGPROOF:
		return NewSigProofMsgFromBytes(b, peerid)

	case MSGID_CLOSEREQ:
		return NewCloseReqMsgFromBytes(b, peerid)
	/* not implemented
	case MSGID_CLOSERESP:
	*/

	case MSGID_DELTASIG:
		return NewDeltaSigMsgFromBytes(b, peerid)
	case MSGID_SIGREV:
		return NewSigRevFromBytes(b, peerid)
	case MSGID_GAPSIGREV:
		return NewGapSigRevFromBytes(b, peerid)
	case MSGID_REV:
		return NewRevMsgFromBytes(b, peerid)

	/*
		case MSGID_FWDMSG:
		case MSGID_FWDAUTHREQ:

		case MSGID_SELFPUSH:
	*/

	case MSGID_WATCH_DESC:
		return NewWatchDescMsgFromBytes(b, peerid)
	case MSGID_WATCH_STATEMSG:
		return NewWatchStateMsgFromBytes(b, peerid)
	/*
		case MSGID_WATCH_DELETE:
	*/

	case MSGID_LINK_DESC:
		return NewLinkMsgFromBytes(b, peerid)

	case MSGID_DUALFUNDINGREQ:
		return NewDualFundingReqMsgFromBytes(b, peerid)
	case MSGID_DUALFUNDINGDECL:
		return NewDualFundingDeclMsgFromBytes(b, peerid)

		/*
			case MSGID_DUALFUNDINGRESP:
				return NewDualFundingRespMsgFromBytes(b, peerid)
			case MSGID_DUALFUNDINGSIGREQ:
				return NewDualFundingSigReqMsgFromBytes(b, peerid)
			case MSGID_DUALFUNDINGSIGRESP:
				return NewDualFundingSigRespMsgFromBytes(b, peerid)*/

	default:
		return nil, fmt.Errorf("Unknown message of type %d ", msgType)
	}
}

//----------

//text message
type ChatMsg struct {
	PeerIdx uint32
	Text    string
}

func NewChatMsg(peerid uint32, text string) ChatMsg {
	t := new(ChatMsg)
	t.PeerIdx = peerid
	t.Text = text
	return *t
}

func NewChatMsgFromBytes(b []byte, peerid uint32) (ChatMsg, error) {
	c := new(ChatMsg)
	c.PeerIdx = peerid

	if len(b) <= 1 {
		return *c, fmt.Errorf("got %d bytes, expect 2 or more", len(b))
	}

	b = b[1:]
	c.Text = string(b)

	return *c, nil
}

func (self ChatMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	msg = append(msg, []byte(self.Text)...)
	return msg
}

func (self ChatMsg) Peer() uint32   { return self.PeerIdx }
func (self ChatMsg) MsgType() uint8 { return MSGID_TEXTCHAT }

//----------

//message with no information, just shows a point is requested
type PointReqMsg struct {
	PeerIdx  uint32
	Cointype uint32
}

func NewPointReqMsg(peerid uint32, cointype uint32) PointReqMsg {
	p := new(PointReqMsg)
	p.PeerIdx = peerid
	p.Cointype = cointype
	return *p
}

func NewPointReqMsgFromBytes(b []byte, peerid uint32) (PointReqMsg, error) {

	pr := new(PointReqMsg)
	pr.PeerIdx = peerid

	if len(b) < 5 {
		return *pr, fmt.Errorf("PointReq msg %d bytes, expect 5\n", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	coin := buf.Next(4)
	pr.Cointype = BtU32(coin)

	return *pr, nil
}

func (self PointReqMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	coin := U32tB(self.Cointype)
	msg = append(msg, coin[:]...)
	return msg
}

func (self PointReqMsg) Peer() uint32   { return self.PeerIdx }
func (self PointReqMsg) MsgType() uint8 { return MSGID_POINTREQ }

//message to be used for reply to point request
type PointRespMsg struct {
	PeerIdx    uint32
	ChannelPub [33]byte
	RefundPub  [33]byte
	HAKDbase   [33]byte
}

func NewPointRespMsg(peerid uint32, chanpub [33]byte, refundpub [33]byte, HAKD [33]byte) PointRespMsg {
	pr := new(PointRespMsg)
	pr.PeerIdx = peerid
	pr.ChannelPub = chanpub
	pr.RefundPub = refundpub
	pr.HAKDbase = HAKD
	return *pr
}

func NewPointRespMsgFromBytes(b []byte, peerid uint32) (PointRespMsg, error) {
	pm := new(PointRespMsg)

	if len(b) < 100 {
		return *pm, fmt.Errorf("PointResp err: msg %d bytes, expect 100\n", len(b))
	}

	pm.PeerIdx = peerid
	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	copy(pm.ChannelPub[:], buf.Next(33))
	copy(pm.RefundPub[:], buf.Next(33))
	copy(pm.HAKDbase[:], buf.Next(33))

	return *pm, nil
}

func (self PointRespMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	msg = append(msg, self.ChannelPub[:]...)
	msg = append(msg, self.RefundPub[:]...)
	msg = append(msg, self.HAKDbase[:]...)
	return msg
}

func (self PointRespMsg) Peer() uint32   { return self.PeerIdx }
func (self PointRespMsg) MsgType() uint8 { return MSGID_POINTRESP }

//message with a channel's description
type ChanDescMsg struct {
	PeerIdx   uint32
	Outpoint  wire.OutPoint
	PubKey    [33]byte
	RefundPub [33]byte
	HAKDbase  [33]byte

	CoinType    uint32
	Capacity    int64
	InitPayment int64

	ElkZero [33]byte //consider changing into array in future
	ElkOne  [33]byte
	ElkTwo  [33]byte

	Data [32]byte
}

func NewChanDescMsg(
	peerid uint32, OP wire.OutPoint,
	pubkey, refund, hakd [33]byte,
	cointype uint32,
	capacity int64, payment int64,
	ELKZero, ELKOne, ELKTwo [33]byte, data [32]byte) ChanDescMsg {

	cd := new(ChanDescMsg)
	cd.PeerIdx = peerid
	cd.Outpoint = OP
	cd.PubKey = pubkey
	cd.RefundPub = refund
	cd.HAKDbase = hakd
	cd.CoinType = cointype
	cd.Capacity = capacity
	cd.InitPayment = payment
	cd.ElkZero = ELKZero
	cd.ElkOne = ELKOne
	cd.ElkTwo = ELKTwo
	cd.Data = data
	return *cd
}

func NewChanDescMsgFromBytes(b []byte, peerid uint32) (ChanDescMsg, error) {
	cm := new(ChanDescMsg)
	cm.PeerIdx = peerid

	if len(b) < 283 {
		return *cm, fmt.Errorf("got %d byte channel description, expect 283", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	var op [36]byte
	copy(op[:], buf.Next(36))
	cm.Outpoint = *OutPointFromBytes(op)
	copy(cm.PubKey[:], buf.Next(33))
	copy(cm.RefundPub[:], buf.Next(33))
	copy(cm.HAKDbase[:], buf.Next(33))
	cm.CoinType = BtU32(buf.Next(4))
	cm.Capacity = BtI64(buf.Next(8))
	cm.InitPayment = BtI64(buf.Next(8))
	copy(cm.ElkZero[:], buf.Next(33))
	copy(cm.ElkOne[:], buf.Next(33))
	copy(cm.ElkTwo[:], buf.Next(33))
	copy(cm.Data[:], buf.Next(32))

	return *cm, nil
}

func (self ChanDescMsg) Bytes() []byte {
	coinTypeBin := U32tB(self.CoinType)
	capBin := I64tB(self.Capacity)
	initBin := I64tB(self.InitPayment)

	var msg []byte
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, self.MsgType())
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.PubKey[:]...)
	msg = append(msg, self.RefundPub[:]...)
	msg = append(msg, self.HAKDbase[:]...)
	msg = append(msg, coinTypeBin[:]...)
	msg = append(msg, capBin[:]...)
	msg = append(msg, initBin[:]...)
	msg = append(msg, self.ElkZero[:]...)
	msg = append(msg, self.ElkOne[:]...)
	msg = append(msg, self.ElkTwo[:]...)
	msg = append(msg, self.Data[:]...)
	return msg
}

func (self ChanDescMsg) Peer() uint32   { return self.PeerIdx }
func (self ChanDescMsg) MsgType() uint8 { return MSGID_CHANDESC }

//message for channel acknowledgement after description message
type ChanAckMsg struct {
	PeerIdx   uint32
	Outpoint  wire.OutPoint
	ElkZero   [33]byte
	ElkOne    [33]byte
	ElkTwo    [33]byte
	Signature [64]byte
}

func NewChanAckMsg(peerid uint32, OP wire.OutPoint, ELKZero [33]byte, ELKOne [33]byte, ELKTwo [33]byte, SIG [64]byte) ChanAckMsg {
	ca := new(ChanAckMsg)
	ca.PeerIdx = peerid
	ca.Outpoint = OP
	ca.ElkZero = ELKZero
	ca.ElkOne = ELKOne
	ca.ElkTwo = ELKTwo
	ca.Signature = SIG
	return *ca
}

func NewChanAckMsgFromBytes(b []byte, peerid uint32) (ChanAckMsg, error) {
	cm := new(ChanAckMsg)
	cm.PeerIdx = peerid

	if len(b) < 200 {
		return *cm, fmt.Errorf("got %d byte multiAck, expect 200", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	cm.Outpoint = *OutPointFromBytes(op)
	copy(cm.ElkZero[:], buf.Next(33))
	copy(cm.ElkOne[:], buf.Next(33))
	copy(cm.ElkTwo[:], buf.Next(33))
	copy(cm.Signature[:], buf.Next(64))
	return *cm, nil
}

func (self ChanAckMsg) Bytes() []byte {
	var msg []byte
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, self.MsgType())
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.ElkZero[:]...)
	msg = append(msg, self.ElkOne[:]...)
	msg = append(msg, self.ElkTwo[:]...)
	msg = append(msg, self.Signature[:]...)
	return msg
}

func (self ChanAckMsg) Peer() uint32   { return self.PeerIdx }
func (self ChanAckMsg) MsgType() uint8 { return MSGID_CHANACK }

//message for proof for a signature
type SigProofMsg struct {
	PeerIdx   uint32
	Outpoint  wire.OutPoint
	Signature [64]byte
}

func NewSigProofMsg(peerid uint32, OP wire.OutPoint, SIG [64]byte) SigProofMsg {
	sp := new(SigProofMsg)
	sp.PeerIdx = peerid
	sp.Outpoint = OP
	sp.Signature = SIG
	return *sp
}

func NewSigProofMsgFromBytes(b []byte, peerid uint32) (SigProofMsg, error) {
	sm := new(SigProofMsg)
	sm.PeerIdx = peerid

	if len(b) < 101 {
		return *sm, fmt.Errorf("got %d byte Sigproof, expect ~101\n", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	sm.Outpoint = *OutPointFromBytes(op)
	copy(sm.Signature[:], buf.Next(64))
	return *sm, nil
}

func (self SigProofMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.Signature[:]...)
	return msg
}

func (self SigProofMsg) Peer() uint32   { return self.PeerIdx }
func (self SigProofMsg) MsgType() uint8 { return MSGID_SIGPROOF }

//----------

//message for closing a channel
type CloseReqMsg struct {
	PeerIdx   uint32
	Outpoint  wire.OutPoint
	Signature [64]byte
}

func NewCloseReqMsg(peerid uint32, OP wire.OutPoint, SIG [64]byte) CloseReqMsg {
	cr := new(CloseReqMsg)
	cr.PeerIdx = peerid
	cr.Outpoint = OP
	cr.Signature = SIG
	return *cr
}

func NewCloseReqMsgFromBytes(b []byte, peerid uint32) (CloseReqMsg, error) {
	crm := new(CloseReqMsg)
	crm.PeerIdx = peerid

	if len(b) < 101 {
		return *crm, fmt.Errorf("got %d byte closereq, expect 101ish\n", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	crm.Outpoint = *OutPointFromBytes(op)

	copy(crm.Signature[:], buf.Next(64))
	return *crm, nil
}

func (self CloseReqMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.Signature[:]...)
	return msg
}

func (self CloseReqMsg) Peer() uint32   { return self.PeerIdx }
func (self CloseReqMsg) MsgType() uint8 { return MSGID_CLOSEREQ }

//----------

//message for sending an amount with the signature
type DeltaSigMsg struct {
	PeerIdx   uint32
	Outpoint  wire.OutPoint
	Delta     int32
	Signature [64]byte
	Data      [32]byte
}

func NewDeltaSigMsg(peerid uint32, OP wire.OutPoint, DELTA int32, SIG [64]byte, data [32]byte) DeltaSigMsg {
	d := new(DeltaSigMsg)
	d.PeerIdx = peerid
	d.Outpoint = OP
	d.Delta = DELTA
	d.Signature = SIG
	d.Data = data
	return *d
}

func NewDeltaSigMsgFromBytes(b []byte, peerid uint32) (DeltaSigMsg, error) {
	ds := new(DeltaSigMsg)
	ds.PeerIdx = peerid

	if len(b) < 105 {
		return *ds, fmt.Errorf("got %d byte DeltaSig, expect 105", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	ds.Outpoint = *OutPointFromBytes(op)

	// deserialize DeltaSig
	ds.Delta = BtI32(buf.Next(4))
	copy(ds.Signature[:], buf.Next(64))
	copy(ds.Data[:], buf.Next(32))
	return *ds, nil
}

func (self DeltaSigMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, I32tB(self.Delta)...)
	msg = append(msg, self.Signature[:]...)
	msg = append(msg, self.Data[:]...)
	return msg
}

func (self DeltaSigMsg) Peer() uint32   { return self.PeerIdx }
func (self DeltaSigMsg) MsgType() uint8 { return MSGID_DELTASIG }

//a message that pushes using channel information
type SigRevMsg struct {
	PeerIdx    uint32
	Outpoint   wire.OutPoint
	Signature  [64]byte
	Elk        chainhash.Hash
	N2ElkPoint [33]byte
}

func NewSigRev(peerid uint32, OP wire.OutPoint, SIG [64]byte, ELK chainhash.Hash, N2ELK [33]byte) SigRevMsg {
	s := new(SigRevMsg)
	s.PeerIdx = peerid
	s.Outpoint = OP
	s.Signature = SIG
	s.Elk = ELK
	s.N2ElkPoint = N2ELK
	return *s
}

func NewSigRevFromBytes(b []byte, peerid uint32) (SigRevMsg, error) {
	sr := new(SigRevMsg)
	sr.PeerIdx = peerid

	if len(b) < 166 {
		return *sr, fmt.Errorf("got %d byte SIGREV, expect 166", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	sr.Outpoint = *OutPointFromBytes(op)
	copy(sr.Signature[:], buf.Next(64))
	elk, _ := chainhash.NewHash(buf.Next(32))
	sr.Elk = *elk
	copy(sr.N2ElkPoint[:], buf.Next(33))
	return *sr, nil
}

func (self SigRevMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.Signature[:]...)
	msg = append(msg, self.Elk[:]...)
	msg = append(msg, self.N2ElkPoint[:]...)
	return msg
}

func (self SigRevMsg) Peer() uint32   { return self.PeerIdx }
func (self SigRevMsg) MsgType() uint8 { return MSGID_SIGREV }

//message for signaling state has moved, revoking old state
type GapSigRevMsg struct {
	PeerIdx    uint32
	Outpoint   wire.OutPoint
	Signature  [64]byte
	Elk        chainhash.Hash
	N2ElkPoint [33]byte
}

func NewGapSigRev(peerid uint32, OP wire.OutPoint, SIG [64]byte, ELK chainhash.Hash, N2ELK [33]byte) GapSigRevMsg {
	g := new(GapSigRevMsg)
	g.PeerIdx = peerid
	g.Outpoint = OP
	g.Signature = SIG
	g.Elk = ELK
	g.N2ElkPoint = N2ELK
	return *g
}

func NewGapSigRevFromBytes(b []byte, peerId uint32) (GapSigRevMsg, error) {
	gs := new(GapSigRevMsg)
	gs.PeerIdx = peerId

	if len(b) < 166 {
		return *gs, fmt.Errorf("got %d byte GAPSIGREV, expect 166", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	gs.Outpoint = *OutPointFromBytes(op)
	copy(gs.Signature[:], buf.Next(64))
	elk, _ := chainhash.NewHash(buf.Next(32))
	gs.Elk = *elk
	copy(gs.N2ElkPoint[:], buf.Next(33))
	return *gs, nil
}

func (self GapSigRevMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.Signature[:]...)
	msg = append(msg, self.Elk[:]...)
	msg = append(msg, self.N2ElkPoint[:]...)
	return msg
}

func (self GapSigRevMsg) Peer() uint32   { return self.PeerIdx }
func (self GapSigRevMsg) MsgType() uint8 { return MSGID_GAPSIGREV }

//send message across channel using Elk info
type RevMsg struct {
	PeerIdx    uint32
	Outpoint   wire.OutPoint
	Elk        chainhash.Hash
	N2ElkPoint [33]byte
}

func NewRevMsg(peerid uint32, OP wire.OutPoint, ELK chainhash.Hash, N2ELK [33]byte) RevMsg {
	r := new(RevMsg)
	r.PeerIdx = peerid
	r.Outpoint = OP
	r.Elk = ELK
	r.N2ElkPoint = N2ELK
	return *r
}

func NewRevMsgFromBytes(b []byte, peerId uint32) (RevMsg, error) {
	rv := new(RevMsg)
	rv.PeerIdx = peerId

	if len(b) < 102 {
		return *rv, fmt.Errorf("got %d byte REV, expect 102", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	rv.Outpoint = *OutPointFromBytes(op)
	elk, _ := chainhash.NewHash(buf.Next(32))
	rv.Elk = *elk
	copy(rv.N2ElkPoint[:], buf.Next(33))
	return *rv, nil
}

func (self RevMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.Elk[:]...)
	msg = append(msg, self.N2ElkPoint[:]...)
	return msg
}

func (self RevMsg) Peer() uint32   { return self.PeerIdx }
func (self RevMsg) MsgType() uint8 { return MSGID_REV }

//----------

// 2 structs that the watchtower gets from clients: Descriptors and Msgs

// Descriptors are 128 bytes
// PKH 20
// Delay 2
// Fee 8
// HAKDbase 33
// Timebase 33
// Elk0 32

// WatchannelDescriptor is the initial message setting up a Watchannel
type WatchDescMsg struct {
	PeerIdx       uint32
	CoinType      uint32   // what network this channel is on
	DestPKHScript [20]byte // PKH to grab to; main unique identifier.

	Delay uint16 // timeout in blocks
	Fee   int64  // fee to use for grab tx.  Or fee rate...?

	CustomerBasePoint  [33]byte // client's HAKD key base point
	AdversaryBasePoint [33]byte // potential attacker's timeout basepoint
}

// NewWatchDescMsg turns 96 bytes into a WatchannelDescriptor
// Silently fails with incorrect size input, watch out.
func NewWatchDescMsg(
	peeridx, coinType uint32, destScript [20]byte,
	delay uint16, fee int64, customerBase [33]byte,
	adversaryBase [33]byte) WatchDescMsg {

	wd := new(WatchDescMsg)
	wd.PeerIdx = peeridx
	wd.CoinType = coinType
	wd.DestPKHScript = destScript
	wd.Delay = delay
	wd.Fee = fee
	wd.CustomerBasePoint = customerBase
	wd.AdversaryBasePoint = adversaryBase
	return *wd
}

func NewWatchDescMsgFromBytes(b []byte, peerIDX uint32) (WatchDescMsg, error) {
	sd := new(WatchDescMsg)
	sd.PeerIdx = peerIDX

	if len(b) < 97 {
		return *sd, fmt.Errorf("WatchannelDescriptor %d bytes, expect 97", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	_ = binary.Read(buf, binary.BigEndian, &sd.CoinType)

	copy(sd.DestPKHScript[:], buf.Next(20))
	_ = binary.Read(buf, binary.BigEndian, &sd.Delay)

	_ = binary.Read(buf, binary.BigEndian, &sd.Fee)

	copy(sd.CustomerBasePoint[:], buf.Next(33))
	copy(sd.AdversaryBasePoint[:], buf.Next(33))

	return *sd, nil
}

// Bytes turns a WatchannelDescriptor into 100 bytes
func (self WatchDescMsg) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(self.MsgType())
	binary.Write(&buf, binary.BigEndian, self.CoinType)
	buf.Write(self.DestPKHScript[:])
	binary.Write(&buf, binary.BigEndian, self.Delay)
	binary.Write(&buf, binary.BigEndian, self.Fee)
	buf.Write(self.CustomerBasePoint[:])
	buf.Write(self.AdversaryBasePoint[:])
	return buf.Bytes()
}

func (self WatchDescMsg) Peer() uint32   { return self.PeerIdx }
func (self WatchDescMsg) MsgType() uint8 { return MSGID_WATCH_DESC }

// the message describing the next commitment tx, sent from the client to the watchtower

// ComMsg are 137 bytes.
// msgtype
// CoinType 4
// PKH 20
// txid 16
// sig 64
// elk 32
type WatchStateMsg struct {
	PeerIdx  uint32
	CoinType uint32         // could figure it out from PKH but this is easier
	DestPKH  [20]byte       // identifier for channel; could be optimized away
	Elk      chainhash.Hash // elkrem for this state index
	ParTxid  [16]byte       // 16 bytes of txid
	Sig      [64]byte       // 64 bytes of sig
}

func NewComMsg(peerIdx, cointype uint32, destPKH [20]byte,
	elk chainhash.Hash, parTxid [16]byte, sig [64]byte) WatchStateMsg {
	cm := new(WatchStateMsg)
	cm.PeerIdx = peerIdx
	cm.CoinType = cointype
	cm.DestPKH = destPKH
	cm.Elk = elk
	cm.ParTxid = parTxid
	cm.Sig = sig
	return *cm
}

// ComMsgFromBytes turns 132 bytes into a SourceMsg
// Silently fails with wrong size input.
func NewWatchStateMsgFromBytes(b []byte, peerIDX uint32) (WatchStateMsg, error) {
	sm := new(WatchStateMsg)
	sm.PeerIdx = peerIDX

	if len(b) < 137 {
		return *sm, fmt.Errorf("WatchComMsg %d bytes, expect 137", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	_ = binary.Read(buf, binary.BigEndian, &sm.CoinType)
	copy(sm.DestPKH[:], buf.Next(20))
	copy(sm.ParTxid[:], buf.Next(16))
	copy(sm.Sig[:], buf.Next(64))
	copy(sm.Elk[:], buf.Next(32))

	return *sm, nil
}

// ToBytes turns a ComMsg into 132 bytes
func (self WatchStateMsg) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(self.MsgType())
	binary.Write(&buf, binary.BigEndian, self.CoinType)
	buf.Write(self.DestPKH[:])
	buf.Write(self.ParTxid[:])
	buf.Write(self.Sig[:])
	buf.Write(self.Elk.CloneBytes())
	return buf.Bytes()
}

func (self WatchStateMsg) Peer() uint32   { return self.PeerIdx }
func (self WatchStateMsg) MsgType() uint8 { return MSGID_WATCH_STATEMSG }

//----------

type WatchDelMsg struct {
	PeerIdx  uint32
	DestPKH  [20]byte // identifier for channel; could be optimized away
	RevealPK [33]byte // reveal this pubkey, matches DestPKH
	// Don't actually have to send DestPKH huh.  Send anyway.
}

// Bytes turns a ComMsg into 132 bytes
func (self WatchDelMsg) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(self.MsgType())
	buf.Write(self.DestPKH[:])
	buf.Write(self.RevealPK[:])
	return buf.Bytes()
}

// ComMsgFromBytes turns 132 bytes into a SourceMsg
// Silently fails with wrong size input.
func NewWatchDelMsgFromBytes(b []byte, peerIDX uint32) (WatchDelMsg, error) {
	sm := new(WatchDelMsg)
	sm.PeerIdx = peerIDX

	if len(b) < 54 {
		return *sm, fmt.Errorf("WatchDelMsg %d bytes, expect 54", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	copy(sm.DestPKH[:], buf.Next(20))
	copy(sm.RevealPK[:], buf.Next(33))

	return *sm, nil
}
func (self WatchDelMsg) Peer() uint32   { return self.PeerIdx }
func (self WatchDelMsg) MsgType() uint8 { return MSGID_WATCH_DELETE }

// Link message

type LinkMsg struct {
	PeerIdx   uint32
	PKHScript [20]byte // ChanPKH (channel ID)
	APKH      [20]byte // APKH (A's LN address)
	ACapacity int64    // ACapacity (A's channel balance)
	BPKH      [20]byte // BPKH (B's LN address)
	CoinType  uint32   // CoinType (Network of the channel)
	Seq       uint32   // seq (Link state sequence #)
	Timestamp int64
}

func NewLinkMsgFromBytes(b []byte, peerIDX uint32) (LinkMsg, error) {
	sm := new(LinkMsg)
	sm.PeerIdx = peerIDX

	if len(b) < 76 {
		return *sm, fmt.Errorf("LinkMsg %d bytes, expect 76", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	copy(sm.PKHScript[:], buf.Next(20))
	copy(sm.APKH[:], buf.Next(20))
	_ = binary.Read(buf, binary.BigEndian, &sm.ACapacity)
	copy(sm.BPKH[:], buf.Next(20))
	_ = binary.Read(buf, binary.BigEndian, &sm.CoinType)
	_ = binary.Read(buf, binary.BigEndian, &sm.Seq)

	return *sm, nil
}

// ToBytes turns a LinkMsg into 88 bytes
func (self LinkMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(self.MsgType())

	buf.Write(self.PKHScript[:])

	buf.Write(self.APKH[:])
	binary.Write(&buf, binary.BigEndian, self.ACapacity)

	buf.Write(self.BPKH[:])

	binary.Write(&buf, binary.BigEndian, self.CoinType)
	binary.Write(&buf, binary.BigEndian, self.Seq)

	return buf.Bytes()
}

func (self LinkMsg) Peer() uint32   { return self.PeerIdx }
func (self LinkMsg) MsgType() uint8 { return MSGID_LINK_DESC }

// Dual funding messages

type DualFundingReqMsg struct {
	PeerIdx             uint32
	CoinType            uint32          // Cointype we are funding
	OurAmount           int64           // The amount we are funding
	TheirAmount         int64           // The amount we are requesting the counterparty to fund
	OurChangeAddressPKH [20]byte        // The address we want to receive change for funding
	OurUTXOs            []wire.OutPoint // The UTXOs we will use for funding
}

func NewDualFundingReqMsg(peerIdx, cointype uint32, ourAmount int64, theirAmount int64, ourChangeAddressPKH [20]byte, ourTxos []*portxo.PorTxo) DualFundingReqMsg {
	msg := new(DualFundingReqMsg)
	msg.PeerIdx = peerIdx
	msg.CoinType = cointype
	msg.OurAmount = ourAmount
	msg.TheirAmount = theirAmount
	msg.OurChangeAddressPKH = ourChangeAddressPKH

	msg.OurUTXOs = make([]wire.OutPoint, len(ourTxos))
	for i := 0; i < len(ourTxos); i++ {
		msg.OurUTXOs[i] = ourTxos[i].Op
	}

	return *msg
}

func NewDualFundingReqMsgFromBytes(b []byte, peerIdx uint32) (DualFundingReqMsg, error) {
	msg := new(DualFundingReqMsg)
	msg.PeerIdx = peerIdx

	if len(b) < 45 {
		return *msg, fmt.Errorf("DualFundingReqMsg %d bytes, expect at least 45", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	_ = binary.Read(buf, binary.BigEndian, &msg.CoinType)
	_ = binary.Read(buf, binary.BigEndian, &msg.OurAmount)
	_ = binary.Read(buf, binary.BigEndian, &msg.TheirAmount)
	copy(msg.OurChangeAddressPKH[:], buf.Next(20))

	var utxoCount uint32
	_ = binary.Read(buf, binary.BigEndian, &utxoCount)
	expectedLength := uint32(45) + 36*utxoCount

	if uint32(len(b)) < expectedLength {
		return *msg, fmt.Errorf("DualFundingReqMsg %d bytes, expect at least %d for %d txos", len(b), expectedLength, utxoCount)
	}

	msg.OurUTXOs = make([]wire.OutPoint, utxoCount)
	var op [36]byte
	for i := uint32(0); i < utxoCount; i++ {
		copy(op[:], buf.Next(36))
		msg.OurUTXOs[i] = *OutPointFromBytes(op)
	}

	return *msg, nil
}

// ToBytes turns a DualFundingReqMsg into bytes
func (self DualFundingReqMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(self.MsgType())

	binary.Write(&buf, binary.BigEndian, self.CoinType)
	binary.Write(&buf, binary.BigEndian, self.OurAmount)
	binary.Write(&buf, binary.BigEndian, self.TheirAmount)
	buf.Write(self.OurChangeAddressPKH[:])

	binary.Write(&buf, binary.BigEndian, uint32(len(self.OurUTXOs)))

	for i := 0; i < len(self.OurUTXOs); i++ {
		opArr := OutPointToBytes(self.OurUTXOs[i])
		buf.Write(opArr[:])
	}

	return buf.Bytes()
}

func (self DualFundingReqMsg) Peer() uint32   { return self.PeerIdx }
func (self DualFundingReqMsg) MsgType() uint8 { return MSGID_DUALFUNDINGREQ }

type DualFundingDeclMsg struct {
	PeerIdx uint32
	Reason  uint8 // Reason for declining the funding request
}

func NewDualFundingDeclMsg(peerIdx uint32, reason uint8) DualFundingDeclMsg {
	msg := new(DualFundingDeclMsg)
	msg.PeerIdx = peerIdx
	msg.Reason = reason
	return *msg
}

func NewDualFundingDeclMsgFromBytes(b []byte, peerIdx uint32) (DualFundingDeclMsg, error) {
	msg := new(DualFundingDeclMsg)
	msg.PeerIdx = peerIdx

	if len(b) < 2 {
		return *msg, fmt.Errorf("DualFundingDeclMsg %d bytes, expect at least 2", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	_ = binary.Read(buf, binary.BigEndian, &msg.Reason)

	return *msg, nil
}

// ToBytes turns a DualFundingReqMsg into bytes
func (self DualFundingDeclMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(self.MsgType())

	binary.Write(&buf, binary.BigEndian, self.Reason)
	return buf.Bytes()
}

func (self DualFundingDeclMsg) Peer() uint32   { return self.PeerIdx }
func (self DualFundingDeclMsg) MsgType() uint8 { return MSGID_DUALFUNDINGDECL }
