package lnutil

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
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
	MSGID_WATCH_DESC   = 0x60 // desc describes a new channel
	MSGID_WATCH_COMMSG = 0x61 // commsg is a single state in the channel
	MSGID_WATCH_DELETE = 0x62 // Watch_clear marks a channel as ok to delete.  No further updates possible.
)

//interface that all messages follow, for easy use
type LitMsg interface {
	Peer() uint32   //return PeerIdx
	MsgType() uint8 //returns Message Type (see constants above)
	Bytes() []byte  //returnns data of message as []byte with the MsgType() preceeding it
}

//method for finding what type of message a generic []byte is

func LitMsgFromBytes(b []byte, peerid uint32) (LitMsg, error) {
	msgType := b[0] // first byte signifies what type of message is

	switch msgType {
	case MSGID_TEXTCHAT:
		return NewChatMsgFromBytes(b, peerid)
	case MSGID_POINTREQ:
		return NewPointReqMsgFromBytes(peerid)
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
	case MSGID_WATCH_COMMSG:
		return NewComMsgFromBytes(b, peerid)
	/*
		case MSGID_WATCH_DELETE:
	*/

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

func NewChatMsg(peerid uint32, text string) *ChatMsg {
	t := new(ChatMsg)
	t.PeerIdx = peerid
	t.Text = text
	return t
}

func NewChatMsgFromBytes(b []byte, peerid uint32) (*ChatMsg, error) {
	c := new(ChatMsg)
	c.PeerIdx = peerid
	b = b[1:]
	c.Text = string(b)

	return c, nil // no way to error
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
	PeerIdx uint32
}

func NewPointReqMsg(peerid uint32) *PointReqMsg {
	p := new(PointReqMsg)
	p.PeerIdx = peerid
	return p
}

func NewPointReqMsgFromBytes(peerid uint32) (*PointReqMsg, error) {
	return NewPointReqMsg(peerid), nil // no way to have error
}

func (self PointReqMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
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

func NewPointRespMsg(peerid uint32, chanpub [33]byte, refundpub [33]byte, HAKD [33]byte) *PointRespMsg {
	pr := new(PointRespMsg)
	pr.PeerIdx = peerid
	pr.ChannelPub = chanpub
	pr.RefundPub = refundpub
	pr.HAKDbase = HAKD
	return pr
}

func NewPointRespMsgFromBytes(b []byte, peerid uint32) (*PointRespMsg, error) {
	pm := new(PointRespMsg)
	b = b[1:] // get rid of messageType
	pm.PeerIdx = peerid

	if len(b) != 99 {
		return nil, fmt.Errorf("PointRespHandler err: msg %d bytes, expect 99\n", len(b))
	}

	copy(pm.ChannelPub[:], b[:33])
	copy(pm.RefundPub[:], b[33:36])
	copy(pm.HAKDbase[:], b[36:])

	return pm, nil
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

	Capacity    int64
	InitPayment int64

	ElkZero   [33]byte //consider changing into array in future
	ElkOne    [33]byte
	ElkTwo    [33]byte
	Signature [64]byte
}

func NewChanDescMsg(peerid uint32, OP wire.OutPoint, pubkey [33]byte, refund [33]byte, hakd [33]byte,
	capacity int64, payment int64, ELKZero [33]byte, ELKOne [33]byte, ELKTwo [33]byte) *ChanDescMsg {

	cd := new(ChanDescMsg)
	cd.PeerIdx = peerid
	cd.Outpoint = OP
	cd.PubKey = pubkey
	cd.RefundPub = refund
	cd.HAKDbase = hakd
	cd.Capacity = capacity
	cd.InitPayment = payment
	cd.ElkZero = ELKZero
	cd.ElkOne = ELKOne
	cd.ElkTwo = ELKTwo
	return cd
}

func NewChanDescMsgFromBytes(b []byte, peerid uint32) (*ChanDescMsg, error) {
	cm := new(ChanDescMsg)
	b = b[1:] // get rid of messageType
	cm.PeerIdx = peerid

	if len(b) != 250 {
		return nil, fmt.Errorf("got %d byte channel description, expect 250", len(b))
	}

	var op [36]byte
	copy(op[:], b[:36])
	cm.Outpoint = *OutPointFromBytes(op)
	copy(cm.PubKey[:], b[36:69])
	copy(cm.RefundPub[:], b[69:102])
	copy(cm.HAKDbase[:], b[102:135])
	cm.Capacity = BtI64(b[135:143])
	cm.InitPayment = BtI64(b[143:151])
	copy(cm.ElkZero[:], b[151:184])
	copy(cm.ElkOne[:], b[184:217])
	copy(cm.ElkTwo[:], b[217:])
	return cm, nil
}

func (self ChanDescMsg) Bytes() []byte {
	capBin := make([]byte, 8) // turn int64 to []byte
	binary.LittleEndian.PutUint64(capBin, uint64(self.Capacity))
	initBin := make([]byte, 8)
	binary.LittleEndian.PutUint64(initBin, uint64(self.InitPayment))

	var msg []byte
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, self.MsgType())
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.PubKey[:]...)
	msg = append(msg, self.RefundPub[:]...)
	msg = append(msg, self.HAKDbase[:]...)
	msg = append(msg, capBin[:]...)
	msg = append(msg, initBin[:]...)
	msg = append(msg, self.ElkZero[:]...)
	msg = append(msg, self.ElkOne[:]...)
	msg = append(msg, self.ElkTwo[:]...)
	msg = append(msg, self.Signature[:]...)
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

func NewChanAckMsg(peerid uint32, OP wire.OutPoint, ELKZero [33]byte, ELKOne [33]byte, ELKTwo [33]byte, SIG [64]byte) *ChanAckMsg {
	ca := new(ChanAckMsg)
	ca.PeerIdx = peerid
	ca.Outpoint = OP
	ca.ElkZero = ELKZero
	ca.ElkOne = ELKOne
	ca.ElkTwo = ELKTwo
	ca.Signature = SIG
	return ca
}

func NewChanAckMsgFromBytes(b []byte, peerid uint32) (*ChanAckMsg, error) {
	cm := new(ChanAckMsg)
	b = b[1:] // get rid of messageType
	cm.PeerIdx = peerid

	if len(b) != 199 {
		return nil, fmt.Errorf("got %d byte multiAck, expect 199", len(b))
	}

	var op [36]byte
	copy(op[:], b[:36])
	cm.Outpoint = *OutPointFromBytes(op)
	copy(cm.ElkZero[:], b[36:69])
	copy(cm.ElkOne[:], b[69:102])
	copy(cm.ElkTwo[:], b[102:135])
	copy(cm.Signature[:], b[135:])
	return cm, nil
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

func NewSigProofMsg(peerid uint32, OP wire.OutPoint, SIG [64]byte) *SigProofMsg {
	sp := new(SigProofMsg)
	sp.PeerIdx = peerid
	sp.Outpoint = OP
	sp.Signature = SIG
	return sp
}

func NewSigProofMsgFromBytes(b []byte, peerid uint32) (*SigProofMsg, error) {
	sm := new(SigProofMsg)
	b = b[1:] // get rid of messageType
	sm.PeerIdx = peerid

	if len(b) != 100 {
		return nil, fmt.Errorf("got %d byte Sigproof, expect ~100\n", len(b))
	}

	var op [36]byte
	copy(op[:], b[:36])
	sm.Outpoint = *OutPointFromBytes(op)
	copy(sm.Signature[:], b[36:])
	return sm, nil
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

func NewCloseReqMsg(peerid uint32, OP wire.OutPoint, SIG [64]byte) *CloseReqMsg {
	cr := new(CloseReqMsg)
	cr.PeerIdx = peerid
	cr.Outpoint = OP
	cr.Signature = SIG
	return cr
}

func NewCloseReqMsgFromBytes(b []byte, peerid uint32) (*DeltaSigMsg, error) {
	crm := new(DeltaSigMsg)
	b = b[1:] // get rid of messageType
	crm.PeerIdx = peerid

	if len(b) != 100 {
		return crm, fmt.Errorf("got %d byte closereq, expect 100ish\n", len(b))
	}

	var op [36]byte
	copy(op[:], b[:36])
	crm.Outpoint = *OutPointFromBytes(op)

	copy(crm.Signature[:], b[36:])
	return crm, nil
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
	Delta     uint32
	Signature [64]byte
}

func NewDeltaSigMsg(peerid uint32, OP wire.OutPoint, DELTA uint32, SIG [64]byte) *DeltaSigMsg {
	d := new(DeltaSigMsg)
	d.PeerIdx = peerid
	d.Outpoint = OP
	d.Delta = DELTA
	d.Signature = SIG
	return d
}

func NewDeltaSigMsgFromBytes(b []byte, peerid uint32) (*DeltaSigMsg, error) {
	ds := new(DeltaSigMsg)
	b = b[1:] // get rid of messageType
	ds.PeerIdx = peerid

	if len(b) != 104 {
		return ds, fmt.Errorf("got %d byte DeltaSig, expect 104", len(b))
	}

	var op [36]byte
	copy(op[:], b[:36])
	ds.Outpoint = *OutPointFromBytes(op)

	// deserialize DeltaSig
	ds.Delta = BtU32(b[36:40])
	copy(ds.Signature[:], b[40:])
	return ds, nil
}

func (self DeltaSigMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, U32tB(self.Delta)...)
	msg = append(msg, self.Signature[:]...)
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

func NewSigRev(peerid uint32, OP wire.OutPoint, SIG [64]byte, ELK chainhash.Hash, N2ELK [33]byte) *SigRevMsg {
	s := new(SigRevMsg)
	s.PeerIdx = peerid
	s.Outpoint = OP
	s.Signature = SIG
	s.Elk = ELK
	s.N2ElkPoint = N2ELK
	return s
}

func NewSigRevFromBytes(b []byte, peerid uint32) (*SigRevMsg, error) {
	sr := new(SigRevMsg)
	b = b[1:] // get rid of messageType
	sr.PeerIdx = peerid

	if len(b) != 165 {
		return nil, fmt.Errorf("got %d byte SIGREV, expect 165", len(b))
	}

	var op [36]byte
	copy(op[:], b[:36])
	sr.Outpoint = *OutPointFromBytes(op)
	copy(sr.Signature[:], b[36:100])
	elk, _ := chainhash.NewHash(b[100:132])
	sr.Elk = *elk
	copy(sr.N2ElkPoint[:], b[132:])
	return sr, nil
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

func NewGapSigRev(peerid uint32, OP wire.OutPoint, SIG [64]byte, ELK chainhash.Hash, N2ELK [33]byte) *GapSigRevMsg {
	g := new(GapSigRevMsg)
	g.PeerIdx = peerid
	g.Outpoint = OP
	g.Signature = SIG
	g.Elk = ELK
	g.N2ElkPoint = N2ELK
	return g
}

func NewGapSigRevFromBytes(b []byte, peerId uint32) (*GapSigRevMsg, error) {
	gs := new(GapSigRevMsg)
	gs.PeerIdx = peerId
	b = b[1:] // get rid of messageType
	if len(b) != 165 {
		return nil, fmt.Errorf("got %d byte GAPSIGREV, expect 165", len(b))
	}

	var op [36]byte
	copy(op[:], b[:36])
	gs.Outpoint = *OutPointFromBytes(op)
	copy(gs.Signature[:], b[36:100])
	elk, _ := chainhash.NewHash(b[100:132])
	gs.Elk = *elk
	copy(gs.N2ElkPoint[:], b[132:])
	return gs, nil
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

//send message accross channel using Elk info
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

func NewRevMsgFromBytes(b []byte, peerId uint32) (*RevMsg, error) {
	rv := new(RevMsg)
	rv.PeerIdx = peerId
	b = b[1:] // get rid of messageType
	if len(b) != 101 {
		return nil, fmt.Errorf("got %d byte REV, expect 101", len(b))
	}

	var op [36]byte
	copy(op[:], b[:36])
	rv.Outpoint = *OutPointFromBytes(op)
	elk, _ := chainhash.NewHash(b[36:68])
	rv.Elk = *elk
	copy(rv.N2ElkPoint[:], b[68:])
	return rv, nil
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
	DestPKHScript [20]byte // PKH to grab to; main unique identifier.

	Delay uint16 // timeout in blocks
	Fee   int64  // fee to use for grab tx.  Or fee rate...?

	CustomerBasePoint  [33]byte // client's HAKD key base point
	AdversaryBasePoint [33]byte // potential attacker's timeout basepoint
}

// NewWatchDescMsg turns 96 bytes into a WatchannelDescriptor
// Silently fails with incorrect size input, watch out.
func NewWatchDescMsg(peeridx uint32, destScript [20]byte, delay uint16, fee int64, customerBase [33]byte, adversaryBase [33]byte) *WatchDescMsg {
	wd := new(WatchDescMsg)
	wd.PeerIdx = peeridx
	wd.DestPKHScript = destScript
	wd.Delay = delay
	wd.Fee = fee
	wd.CustomerBasePoint = customerBase
	wd.AdversaryBasePoint = adversaryBase
	return wd
}

func NewWatchDescMsgFromBytes(b []byte, peerIDX uint32) (*WatchDescMsg, error) {
	sd := new(WatchDescMsg)
	b = b[1:] // get rid of messageType
	sd.PeerIdx = peerIDX
	if len(b) != 96 && len(b) != 128 {
		return sd, fmt.Errorf("WatchannelDescriptor %d bytes, expect 128 or 96", len(b))
	}
	buf := bytes.NewBuffer(b)

	copy(sd.DestPKHScript[:], buf.Next(20))
	_ = binary.Read(buf, binary.BigEndian, &sd.Delay)

	_ = binary.Read(buf, binary.BigEndian, &sd.Fee)

	copy(sd.CustomerBasePoint[:], buf.Next(33))
	copy(sd.AdversaryBasePoint[:], buf.Next(33))

	return sd, nil
}

// Bytes turns a WatchannelDescriptor into 100 bytes
func (self WatchDescMsg) Bytes() []byte {
	var buf bytes.Buffer
	msgType := U32tB(uint32(self.MsgType()))
	buf.Write(msgType)
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

// ComMsg are 132 bytes.
// PKH 20
// txid 16
// sig 64
// elk 32
type ComMsg struct {
	PeerIdx uint32
	DestPKH [20]byte       // identifier for channel; could be optimized away
	Elk     chainhash.Hash // elkrem for this state index
	ParTxid [16]byte       // 16 bytes of txid
	Sig     [64]byte       // 64 bytes of sig
}

func NewComMsg(peerIdx uint32, destPKH [20]byte, elk chainhash.Hash, parTxid [16]byte, sig [64]byte) *ComMsg {
	cm := new(ComMsg)
	cm.PeerIdx = peerIdx
	cm.DestPKH = destPKH
	cm.Elk = elk
	cm.ParTxid = parTxid
	cm.Sig = sig
	return cm
}

// ComMsgFromBytes turns 132 bytes into a SorceMsg
// Silently fails with wrong size input.
func NewComMsgFromBytes(b []byte, peerIDX uint32) (*ComMsg, error) {
	sm := new(ComMsg)
	b = b[1:] // get rid of messageType
	sm.PeerIdx = peerIDX
	if len(b) != 132 {
		return sm, fmt.Errorf(
			"WatchComMsg %d bytes, expect 132", len(b))
	}

	copy(sm.DestPKH[:], b[:20])
	copy(sm.ParTxid[:], b[20:36])
	copy(sm.Sig[:], b[36:100])
	copy(sm.Elk[:], b[100:])
	return sm, nil
}

// ToBytes turns a ComMsg into 132 bytes
func (self ComMsg) Bytes() []byte {
	var buf bytes.Buffer
	msgType := U32tB(uint32(self.MsgType()))
	buf.Write(msgType)
	buf.Write(self.DestPKH[:])
	buf.Write(self.ParTxid[:])
	buf.Write(self.Sig[:])
	buf.Write(self.Elk.CloneBytes())
	return buf.Bytes()
}

func (self ComMsg) Peer() uint32   { return self.PeerIdx }
func (self ComMsg) MsgType() uint8 { return MSGID_WATCH_COMMSG }

//----------
