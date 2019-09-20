package lnutil

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/wire"
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

	// HTLC messages
	MSGID_HASHSIG     = 0x34 // Like a deltasig but offers an HTLC
	MSGID_PREIMAGESIG = 0x35 // Like a hashsig but clears an HTLC

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

	//Multihop payment messages
	MSGID_PAY_REQ   = 0x75 // Request payment
	MSGID_PAY_ACK   = 0x76 // Acknowledge payment (share preimage hash)
	MSGID_PAY_SETUP = 0x77 // Setup a payment route

	//Discreet log contracts messages
	MSGID_DLC_OFFER               = 0x90 // Offer a contract
	MSGID_DLC_ACCEPTOFFER         = 0x91 // Accept the contract
	MSGID_DLC_DECLINEOFFER        = 0x92 // Decline the contract
	MSGID_DLC_CONTRACTACK         = 0x93 // Acknowledge an acceptance
	MSGID_DLC_CONTRACTFUNDINGSIGS = 0x94 // Funding signatures
	MSGID_DLC_SIGPROOF            = 0x95 // Sigproof

	//Dual funding messages
	MSGID_DUALFUNDINGREQ     = 0xA0 // Requests funding details (UTXOs, Change address, Pubkey), including our own details and amount needed.
	MSGID_DUALFUNDINGACCEPT  = 0xA1 // Responds with funding details
	MSGID_DUALFUNDINGDECL    = 0xA2 // Declines the funding request
	MSGID_DUALFUNDINGCHANACK = 0xA3 // Acknowledges channel and sends along signatures for funding

	//Remote control messages
	MSGID_REMOTE_RPCREQUEST  = 0xB0 // Contains an RPC request from a remote peer
	MSGID_REMOTE_RPCRESPONSE = 0xB1 // Contains an RPC response to send to a remote peer

	MSGID_CHUNKS_BEGIN uint8 = 0xB2
	MSGID_CHUNK_BODY uint8 = 0xB3
	MSGID_CHUNKS_END uint8 = 0xB4


	DIGEST_TYPE_SHA256    = 0x00
	DIGEST_TYPE_RIPEMD160 = 0x01
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
	case MSGID_HASHSIG:
		return NewHashSigMsgFromBytes(b, peerid)
	case MSGID_PREIMAGESIG:
		return NewPreimageSigMsgFromBytes(b, peerid)

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

	case MSGID_PAY_REQ:
		return NewMultihopPaymentRequestMsgFromBytes(b, peerid)
	case MSGID_PAY_ACK:
		return NewMultihopPaymentAckMsgFromBytes(b, peerid)
	case MSGID_PAY_SETUP:
		return NewMultihopPaymentSetupMsgFromBytes(b, peerid)

	case MSGID_DUALFUNDINGREQ:
		return NewDualFundingReqMsgFromBytes(b, peerid)
	case MSGID_DUALFUNDINGACCEPT:
		return NewDualFundingAcceptMsgFromBytes(b, peerid)
	case MSGID_DUALFUNDINGDECL:
		return NewDualFundingDeclMsgFromBytes(b, peerid)
	case MSGID_DUALFUNDINGCHANACK:
		return NewDualFundingChanAckMsgFromBytes(b, peerid)

	case MSGID_DLC_OFFER:
		return NewDlcOfferMsgFromBytes(b, peerid)
	case MSGID_DLC_ACCEPTOFFER:
		return NewDlcOfferAcceptMsgFromBytes(b, peerid)
	case MSGID_DLC_DECLINEOFFER:
		return NewDlcOfferDeclineMsgFromBytes(b, peerid)
	case MSGID_DLC_CONTRACTACK:
		return NewDlcContractAckMsgFromBytes(b, peerid)
	case MSGID_DLC_CONTRACTFUNDINGSIGS:
		return NewDlcContractFundingSigsMsgFromBytes(b, peerid)
	case MSGID_DLC_SIGPROOF:
		return NewDlcContractSigProofMsgFromBytes(b, peerid)

	case MSGID_REMOTE_RPCREQUEST:
		return NewRemoteControlRpcRequestMsgFromBytes(b, peerid)
	case MSGID_REMOTE_RPCRESPONSE:
		return NewRemoteControlRpcResponseMsgFromBytes(b, peerid)

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

	NextHTLCBase [33]byte
	N2HTLCBase   [33]byte
}

func NewPointRespMsg(peerid uint32, chanpub [33]byte, refundpub [33]byte,
	HAKD [33]byte, nextHTLCBase [33]byte, N2HTLCBase [33]byte) PointRespMsg {
	pr := new(PointRespMsg)
	pr.PeerIdx = peerid
	pr.ChannelPub = chanpub
	pr.RefundPub = refundpub
	pr.HAKDbase = HAKD
	pr.NextHTLCBase = nextHTLCBase
	pr.N2HTLCBase = N2HTLCBase
	return *pr
}

// NewPointRespMsgFromBytes takes a byte slice and a peerid and constructs a
// PointRespMsg object from the bytes. Expects at least 1 + 33 + 33 + 33 +
// 33 + 33 = 166.
func NewPointRespMsgFromBytes(b []byte, peerid uint32) (PointRespMsg, error) {
	pm := new(PointRespMsg)

	if len(b) < 166 {
		return *pm, fmt.Errorf("PointResp err: msg %d bytes, expect 166\n", len(b))
	}

	pm.PeerIdx = peerid
	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	copy(pm.ChannelPub[:], buf.Next(33))
	copy(pm.RefundPub[:], buf.Next(33))
	copy(pm.HAKDbase[:], buf.Next(33))
	copy(pm.NextHTLCBase[:], buf.Next(33))
	copy(pm.N2HTLCBase[:], buf.Next(33))

	return *pm, nil
}

func (self PointRespMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	msg = append(msg, self.ChannelPub[:]...)
	msg = append(msg, self.RefundPub[:]...)
	msg = append(msg, self.HAKDbase[:]...)
	msg = append(msg, self.NextHTLCBase[:]...)
	msg = append(msg, self.N2HTLCBase[:]...)
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

	NextHTLCBase [33]byte
	N2HTLCBase   [33]byte

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
	pubkey, refund, hakd [33]byte, nextHTLCBase [33]byte, N2HTLCBase [33]byte,
	cointype uint32,
	capacity int64, payment int64,
	ELKZero, ELKOne, ELKTwo [33]byte, data [32]byte) ChanDescMsg {

	cd := new(ChanDescMsg)
	cd.PeerIdx = peerid
	cd.Outpoint = OP
	cd.PubKey = pubkey
	cd.RefundPub = refund
	cd.HAKDbase = hakd
	cd.NextHTLCBase = nextHTLCBase
	cd.N2HTLCBase = N2HTLCBase
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
	copy(cm.NextHTLCBase[:], buf.Next(33))
	copy(cm.N2HTLCBase[:], buf.Next(33))
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
	msg = append(msg, self.NextHTLCBase[:]...)
	msg = append(msg, self.N2HTLCBase[:]...)
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
	HTLCSigs  [][64]byte
}

func NewDeltaSigMsg(peerid uint32, OP wire.OutPoint, DELTA int32, SIG [64]byte, HTLCSigs [][64]byte, data [32]byte) DeltaSigMsg {
	d := new(DeltaSigMsg)
	d.PeerIdx = peerid
	d.Outpoint = OP
	d.Delta = DELTA
	d.Signature = SIG
	d.Data = data
	d.HTLCSigs = HTLCSigs
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

	nHTLCs := buf.Len() / 64
	for i := 0; i < nHTLCs; i++ {
		var HTLCSig [64]byte
		copy(HTLCSig[:], buf.Next(64))
		ds.HTLCSigs = append(ds.HTLCSigs, HTLCSig)
	}

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
	for _, sig := range self.HTLCSigs {
		msg = append(msg, sig[:]...)
	}
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
	HTLCSigs   [][64]byte
	N2HTLCBase [33]byte
}

func NewSigRev(peerid uint32, OP wire.OutPoint, SIG [64]byte, ELK chainhash.Hash,
	N2ELK [33]byte, HTLCSigs [][64]byte, N2HTLCBase [33]byte) SigRevMsg {
	s := new(SigRevMsg)
	s.PeerIdx = peerid
	s.Outpoint = OP
	s.Signature = SIG
	s.Elk = ELK
	s.N2ElkPoint = N2ELK
	s.HTLCSigs = HTLCSigs
	s.N2HTLCBase = N2HTLCBase
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

	nHTLCs := (buf.Len() - 33) / 64
	for i := 0; i < nHTLCs; i++ {
		var HTLCSig [64]byte
		copy(HTLCSig[:], buf.Next(64))
		sr.HTLCSigs = append(sr.HTLCSigs, HTLCSig)
	}

	copy(sr.N2HTLCBase[:], buf.Next(33))

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
	for _, sig := range self.HTLCSigs {
		msg = append(msg, sig[:]...)
	}
	msg = append(msg, self.N2HTLCBase[:]...)
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
	N2HTLCBase [33]byte
	HTLCSigs   [][64]byte
}

func NewGapSigRev(peerid uint32, OP wire.OutPoint, SIG [64]byte, ELK chainhash.Hash, N2ELK [33]byte, HTLCSigs [][64]byte, N2HTLCBase [33]byte) GapSigRevMsg {
	g := new(GapSigRevMsg)
	g.PeerIdx = peerid
	g.Outpoint = OP
	g.Signature = SIG
	g.Elk = ELK
	g.N2ElkPoint = N2ELK
	g.N2HTLCBase = N2HTLCBase
	g.HTLCSigs = HTLCSigs
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
	copy(gs.N2HTLCBase[:], buf.Next(33))

	nHTLCs := buf.Len() / 64
	for i := 0; i < nHTLCs; i++ {
		var HTLCSig [64]byte
		copy(HTLCSig[:], buf.Next(64))
		gs.HTLCSigs = append(gs.HTLCSigs, HTLCSig)
	}

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
	msg = append(msg, self.N2HTLCBase[:]...)
	for _, sig := range self.HTLCSigs {
		msg = append(msg, sig[:]...)
	}
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
	N2HTLCBase [33]byte
}

func NewRevMsg(peerid uint32, OP wire.OutPoint, ELK chainhash.Hash, N2ELK [33]byte, N2HTLCBase [33]byte) RevMsg {
	r := new(RevMsg)
	r.PeerIdx = peerid
	r.Outpoint = OP
	r.Elk = ELK
	r.N2ElkPoint = N2ELK
	r.N2HTLCBase = N2HTLCBase
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
	copy(rv.N2HTLCBase[:], buf.Next(33))
	return *rv, nil
}

func (self RevMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, self.Elk[:]...)
	msg = append(msg, self.N2ElkPoint[:]...)
	msg = append(msg, self.N2HTLCBase[:]...)
	return msg
}

func (self RevMsg) Peer() uint32   { return self.PeerIdx }
func (self RevMsg) MsgType() uint8 { return MSGID_REV }

//----------

//message for offering an HTLC
type HashSigMsg struct {
	PeerIdx  uint32
	Outpoint wire.OutPoint

	Amt      int64
	Locktime uint32
	RHash    [32]byte

	Data [32]byte

	CommitmentSignature [64]byte
	// must be at least 36 + 4 + 32 + 33 + 32 + 64 = 169 bytes
	HTLCSigs [][64]byte
}

func NewHashSigMsg(peerid uint32, OP wire.OutPoint, amt int64, locktime uint32, RHash [32]byte, sig [64]byte, HTLCSigs [][64]byte, data [32]byte) HashSigMsg {
	d := new(HashSigMsg)
	d.PeerIdx = peerid
	d.Outpoint = OP
	d.Amt = amt
	d.CommitmentSignature = sig
	d.Data = data
	d.RHash = RHash
	d.Locktime = locktime
	d.HTLCSigs = HTLCSigs
	return *d
}

func NewHashSigMsgFromBytes(b []byte, peerid uint32) (HashSigMsg, error) {
	ds := new(HashSigMsg)
	ds.PeerIdx = peerid

	if len(b) < 169 {
		return *ds, fmt.Errorf("got %d byte HashSig, expect at least 169 bytes", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	ds.Outpoint = *OutPointFromBytes(op)

	// deserialize DeltaSig
	ds.Amt = BtI64(buf.Next(8))
	ds.Locktime = BtU32(buf.Next(4))
	copy(ds.RHash[:], buf.Next(32))

	copy(ds.Data[:], buf.Next(32))

	copy(ds.CommitmentSignature[:], buf.Next(64))

	nHTLCSigs := buf.Len() / 64

	for i := 0; i < nHTLCSigs; i++ {
		var sig [64]byte
		copy(sig[:], buf.Next(64))
		ds.HTLCSigs = append(ds.HTLCSigs, sig)
	}

	return *ds, nil
}

func (self HashSigMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, I64tB(self.Amt)...)
	msg = append(msg, U32tB(self.Locktime)...)
	msg = append(msg, self.RHash[:]...)
	msg = append(msg, self.Data[:]...)
	msg = append(msg, self.CommitmentSignature[:]...)
	for _, sig := range self.HTLCSigs {
		msg = append(msg, sig[:]...)
	}
	return msg
}

func (self HashSigMsg) Peer() uint32   { return self.PeerIdx }
func (self HashSigMsg) MsgType() uint8 { return MSGID_HASHSIG }

//----------

//message for clearing an HTLC
type PreimageSigMsg struct {
	PeerIdx  uint32
	Outpoint wire.OutPoint

	Idx uint32
	R   [16]byte

	Data [32]byte

	CommitmentSignature [64]byte
	// must be at least 36 + 4 + 16 + 32 + 64 = 152 bytes
	HTLCSigs [][64]byte
}

func NewPreimageSigMsg(peerid uint32, OP wire.OutPoint, Idx uint32, R [16]byte, sig [64]byte, HTLCSigs [][64]byte, data [32]byte) PreimageSigMsg {
	d := new(PreimageSigMsg)
	d.PeerIdx = peerid
	d.Outpoint = OP
	d.CommitmentSignature = sig
	d.Data = data
	d.R = R
	d.Idx = Idx
	d.HTLCSigs = HTLCSigs
	return *d
}

func NewPreimageSigMsgFromBytes(b []byte, peerid uint32) (PreimageSigMsg, error) {
	ps := new(PreimageSigMsg)
	ps.PeerIdx = peerid

	if len(b) < 152 {
		return *ps, fmt.Errorf("got %d byte PreimageSig, expect at least 152 bytes", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	ps.Outpoint = *OutPointFromBytes(op)

	ps.Idx = BtU32(buf.Next(4))

	copy(ps.R[:], buf.Next(16))

	copy(ps.Data[:], buf.Next(32))

	copy(ps.CommitmentSignature[:], buf.Next(64))

	nHTLCSigs := buf.Len() / 64

	for i := 0; i < nHTLCSigs; i++ {
		var sig [64]byte
		copy(sig[:], buf.Next(64))
		ps.HTLCSigs = append(ps.HTLCSigs, sig)
	}

	return *ps, nil
}

func (self PreimageSigMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.MsgType())
	opArr := OutPointToBytes(self.Outpoint)
	msg = append(msg, opArr[:]...)
	msg = append(msg, U32tB(self.Idx)...)
	msg = append(msg, self.R[:]...)
	msg = append(msg, self.Data[:]...)
	msg = append(msg, self.CommitmentSignature[:]...)
	for _, sig := range self.HTLCSigs {
		msg = append(msg, sig[:]...)
	}
	return msg
}

func (self PreimageSigMsg) Peer() uint32   { return self.PeerIdx }
func (self PreimageSigMsg) MsgType() uint8 { return MSGID_PREIMAGESIG }

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

// To find how much 1 satoshi of coin type A will cost you in coin type B,
// if Reciprocal is false, 1 satoshi of A will buy you `rate` satoshis of B.
// Otherwise, 1 satoshi of B will buy you `rate` satoshis of A. This avoids the
// use of floating point for deciding price.
type RateDesc struct {
	CoinType   uint32
	Rate       int64
	Reciprocal bool
}

func NewRateDescFromBytes(b []byte) (RateDesc, error) {
	var rd RateDesc

	buf := bytes.NewBuffer(b)

	err := binary.Read(buf, binary.BigEndian, &rd.CoinType)
	if err != nil {
		return rd, err
	}

	err = binary.Read(buf, binary.BigEndian, &rd.Rate)
	if err != nil {
		return rd, err
	}

	err = binary.Read(buf, binary.BigEndian, &rd.Reciprocal)
	if err != nil {
		return rd, err
	}

	return rd, nil
}

func (rd *RateDesc) Bytes() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, rd.CoinType)
	binary.Write(&buf, binary.BigEndian, rd.Rate)
	binary.Write(&buf, binary.BigEndian, rd.Reciprocal)

	return buf.Bytes()
}

type LinkMsg struct {
	PeerIdx   uint32
	APKH      [20]byte // APKH (A's LN address)
	ACapacity int64    // ACapacity (A's channel balance)
	BPKH      [20]byte // BPKH (B's LN address)
	CoinType  uint32   // CoinType (Network of the channel)
	Seq       uint32   // seq (Link state sequence #)
	Timestamp int64
	Rates     []RateDesc
}

func NewLinkMsgFromBytes(b []byte, peerIDX uint32) (LinkMsg, error) {
	sm := new(LinkMsg)
	sm.PeerIdx = peerIDX

	if len(b) < 61 {
		return *sm, fmt.Errorf("LinkMsg %d bytes, expect at least 61", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	copy(sm.APKH[:], buf.Next(20))
	err := binary.Read(buf, binary.BigEndian, &sm.ACapacity)
	if err != nil {
		return *sm, err
	}
	copy(sm.BPKH[:], buf.Next(20))
	err = binary.Read(buf, binary.BigEndian, &sm.CoinType)
	if err != nil {
		return *sm, err
	}
	err = binary.Read(buf, binary.BigEndian, &sm.Seq)
	if err != nil {
		return *sm, err
	}

	var nRates uint32
	err = binary.Read(buf, binary.BigEndian, &nRates)
	if err != nil {
		return *sm, err
	}

	for i := uint32(0); i < nRates; i++ {
		rd, err := NewRateDescFromBytes(buf.Next(13))
		if err != nil {
			return *sm, err
		}

		sm.Rates = append(sm.Rates, rd)
	}

	return *sm, nil
}

// ToBytes turns a LinkMsg into 88 bytes
func (self LinkMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(self.MsgType())

	buf.Write(self.APKH[:])
	binary.Write(&buf, binary.BigEndian, self.ACapacity)

	buf.Write(self.BPKH[:])

	binary.Write(&buf, binary.BigEndian, self.CoinType)
	binary.Write(&buf, binary.BigEndian, self.Seq)

	nRates := uint32(len(self.Rates))
	binary.Write(&buf, binary.BigEndian, nRates)

	for _, rate := range self.Rates {
		buf.Write(rate.Bytes())
	}

	return buf.Bytes()
}

func (self LinkMsg) Peer() uint32   { return self.PeerIdx }
func (self LinkMsg) MsgType() uint8 { return MSGID_LINK_DESC }

// Dual funding messages

type DualFundingReqMsg struct {
	PeerIdx             uint32
	CoinType            uint32 // Cointype we are funding
	OurAmount           int64  // The amount we are funding
	TheirAmount         int64  // The amount we are requesting the counterparty to fund
	OurPub              [33]byte
	OurRefundPub        [33]byte
	OurHAKDBase         [33]byte
	OurChangeAddressPKH [20]byte           // The address we want to receive change for funding
	OurInputs           []DualFundingInput // The inputs we will use for funding
}

type DualFundingInput struct {
	Outpoint wire.OutPoint
	Value    int64
}

func NewDualFundingReqMsg(peerIdx, cointype uint32, ourAmount int64, theirAmount int64, ourPub [33]byte, ourRefundPub [33]byte, ourHAKDBase [33]byte, ourChangeAddressPKH [20]byte, ourInputs []DualFundingInput) DualFundingReqMsg {
	msg := new(DualFundingReqMsg)
	msg.PeerIdx = peerIdx
	msg.CoinType = cointype
	msg.OurAmount = ourAmount
	msg.TheirAmount = theirAmount
	msg.OurPub = ourPub
	msg.OurRefundPub = ourRefundPub
	msg.OurHAKDBase = ourHAKDBase
	msg.OurChangeAddressPKH = ourChangeAddressPKH
	msg.OurInputs = ourInputs

	return *msg
}

func NewDualFundingReqMsgFromBytes(b []byte, peerIdx uint32) (DualFundingReqMsg, error) {
	msg := new(DualFundingReqMsg)
	msg.PeerIdx = peerIdx

	if len(b) < 144 {
		return *msg, fmt.Errorf("DualFundingReqMsg %d bytes, expect at least 144", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	_ = binary.Read(buf, binary.BigEndian, &msg.CoinType)
	_ = binary.Read(buf, binary.BigEndian, &msg.OurAmount)
	_ = binary.Read(buf, binary.BigEndian, &msg.TheirAmount)
	copy(msg.OurPub[:], buf.Next(33))
	copy(msg.OurRefundPub[:], buf.Next(33))
	copy(msg.OurHAKDBase[:], buf.Next(33))
	copy(msg.OurChangeAddressPKH[:], buf.Next(20))

	var utxoCount uint32
	_ = binary.Read(buf, binary.BigEndian, &utxoCount)
	expectedLength := uint32(144) + 44*utxoCount

	if uint32(len(b)) < expectedLength {
		return *msg, fmt.Errorf("DualFundingReqMsg %d bytes, expect at least %d for %d txos", len(b), expectedLength, utxoCount)
	}

	msg.OurInputs = make([]DualFundingInput, utxoCount)
	var op [36]byte
	for i := uint32(0); i < utxoCount; i++ {
		copy(op[:], buf.Next(36))
		msg.OurInputs[i].Outpoint = *OutPointFromBytes(op)
		_ = binary.Read(buf, binary.BigEndian, &msg.OurInputs[i].Value)
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
	buf.Write(self.OurPub[:])
	buf.Write(self.OurRefundPub[:])
	buf.Write(self.OurHAKDBase[:])
	buf.Write(self.OurChangeAddressPKH[:])

	binary.Write(&buf, binary.BigEndian, uint32(len(self.OurInputs)))

	for i := 0; i < len(self.OurInputs); i++ {
		opArr := OutPointToBytes(self.OurInputs[i].Outpoint)
		buf.Write(opArr[:])
		binary.Write(&buf, binary.BigEndian, self.OurInputs[i].Value)
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

type DualFundingAcceptMsg struct {
	PeerIdx             uint32
	CoinType            uint32 // Cointype we are funding
	OurPub              [33]byte
	OurRefundPub        [33]byte
	OurHAKDBase         [33]byte
	OurChangeAddressPKH [20]byte // The address we want to receive change for funding
	OurNextHTLCBase     [33]byte
	OurN2HTLCBase       [33]byte
	OurInputs           []DualFundingInput // The inputs we will use for funding
}

func NewDualFundingAcceptMsg(peerIdx uint32, coinType uint32, ourPub [33]byte, ourRefundPub [33]byte, ourHAKDBase [33]byte, ourChangeAddress [20]byte, ourInputs []DualFundingInput, ourNextHTLCBase [33]byte, ourN2HTLCBase [33]byte) DualFundingAcceptMsg {
	msg := new(DualFundingAcceptMsg)
	msg.PeerIdx = peerIdx
	msg.CoinType = coinType
	msg.OurPub = ourPub
	msg.OurRefundPub = ourRefundPub
	msg.OurHAKDBase = ourHAKDBase
	msg.OurChangeAddressPKH = ourChangeAddress
	msg.OurInputs = ourInputs
	msg.OurNextHTLCBase = ourNextHTLCBase
	msg.OurN2HTLCBase = ourN2HTLCBase
	return *msg
}

func NewDualFundingAcceptMsgFromBytes(b []byte, peerIdx uint32) (DualFundingAcceptMsg, error) {
	msg := new(DualFundingAcceptMsg)
	msg.PeerIdx = peerIdx

	if len(b) < 29 {
		return *msg, fmt.Errorf("DualFundingAcceptMsg %d bytes, expect at least 29", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	_ = binary.Read(buf, binary.BigEndian, &msg.CoinType)
	copy(msg.OurPub[:], buf.Next(33))
	copy(msg.OurRefundPub[:], buf.Next(33))
	copy(msg.OurHAKDBase[:], buf.Next(33))
	copy(msg.OurChangeAddressPKH[:], buf.Next(20))
	copy(msg.OurNextHTLCBase[:], buf.Next(33))
	copy(msg.OurN2HTLCBase[:], buf.Next(33))

	var utxoCount uint32
	_ = binary.Read(buf, binary.BigEndian, &utxoCount)
	expectedLength := uint32(29) + 44*utxoCount

	if uint32(len(b)) < expectedLength {
		return *msg, fmt.Errorf("DualFundingReqMsg %d bytes, expect at least %d for %d txos", len(b), expectedLength, utxoCount)
	}

	msg.OurInputs = make([]DualFundingInput, utxoCount)
	var op [36]byte
	for i := uint32(0); i < utxoCount; i++ {
		copy(op[:], buf.Next(36))
		msg.OurInputs[i].Outpoint = *OutPointFromBytes(op)
		_ = binary.Read(buf, binary.BigEndian, &msg.OurInputs[i].Value)
	}
	return *msg, nil
}

// ToBytes turns a DualFundingReqMsg into bytes
func (self DualFundingAcceptMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(self.MsgType())
	binary.Write(&buf, binary.BigEndian, self.CoinType)
	buf.Write(self.OurPub[:])
	buf.Write(self.OurRefundPub[:])
	buf.Write(self.OurHAKDBase[:])
	buf.Write(self.OurChangeAddressPKH[:])
	buf.Write(self.OurNextHTLCBase[:])
	buf.Write(self.OurN2HTLCBase[:])

	binary.Write(&buf, binary.BigEndian, uint32(len(self.OurInputs)))

	for i := 0; i < len(self.OurInputs); i++ {
		opArr := OutPointToBytes(self.OurInputs[i].Outpoint)
		buf.Write(opArr[:])
		binary.Write(&buf, binary.BigEndian, self.OurInputs[i].Value)
	}

	return buf.Bytes()
}

func (self DualFundingAcceptMsg) Peer() uint32   { return self.PeerIdx }
func (self DualFundingAcceptMsg) MsgType() uint8 { return MSGID_DUALFUNDINGACCEPT }

//message for channel acknowledgement and funding signatures
type DualFundingChanAckMsg struct {
	PeerIdx         uint32
	Outpoint        wire.OutPoint
	ElkZero         [33]byte
	ElkOne          [33]byte
	ElkTwo          [33]byte
	Signature       [64]byte
	SignedFundingTx *wire.MsgTx
}

func NewDualFundingChanAckMsg(peerid uint32, OP wire.OutPoint, ELKZero [33]byte, ELKOne [33]byte, ELKTwo [33]byte, SIG [64]byte, signedFundingTx *wire.MsgTx) DualFundingChanAckMsg {
	ca := new(DualFundingChanAckMsg)
	ca.PeerIdx = peerid
	ca.Outpoint = OP
	ca.ElkZero = ELKZero
	ca.ElkOne = ELKOne
	ca.ElkTwo = ELKTwo
	ca.Signature = SIG
	ca.SignedFundingTx = signedFundingTx
	return *ca
}

func NewDualFundingChanAckMsgFromBytes(b []byte, peerid uint32) (DualFundingChanAckMsg, error) {
	cm := new(DualFundingChanAckMsg)
	cm.PeerIdx = peerid

	if len(b) < 208 {
		return *cm, fmt.Errorf("got %d byte DualFundingChanAck, expect 212 or more", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	var op [36]byte
	copy(op[:], buf.Next(36))
	cm.Outpoint = *OutPointFromBytes(op)
	copy(cm.ElkZero[:], buf.Next(33))
	copy(cm.ElkOne[:], buf.Next(33))
	copy(cm.ElkTwo[:], buf.Next(33))
	copy(cm.Signature[:], buf.Next(64))

	var txLen uint64
	_ = binary.Read(buf, binary.BigEndian, &txLen)
	expectedLength := uint64(208) + txLen

	if uint64(len(b)) < expectedLength {
		return *cm, fmt.Errorf("DualFundingChanAckMsg %d bytes, expect at least %d for %d byte tx", len(b), expectedLength, txLen)
	}

	cm.SignedFundingTx = wire.NewMsgTx()
	cm.SignedFundingTx.Deserialize(buf)

	return *cm, nil
}

func (self DualFundingChanAckMsg) Bytes() []byte {
	var buf bytes.Buffer

	opArr := OutPointToBytes(self.Outpoint)
	buf.WriteByte(self.MsgType())
	buf.Write(opArr[:])
	buf.Write(self.ElkZero[:])
	buf.Write(self.ElkOne[:])
	buf.Write(self.ElkTwo[:])
	buf.Write(self.Signature[:])

	binary.Write(&buf, binary.BigEndian, uint64(self.SignedFundingTx.SerializeSize()))
	writer := bufio.NewWriter(&buf)
	self.SignedFundingTx.Serialize(writer)
	writer.Flush()

	return buf.Bytes()
}

func (self DualFundingChanAckMsg) Peer() uint32   { return self.PeerIdx }
func (self DualFundingChanAckMsg) MsgType() uint8 { return MSGID_DUALFUNDINGCHANACK }

// DlcOfferMsg is the message we send to a peer to offer that peer a
// particular contract
type DlcOfferMsg struct {
	PeerIdx  uint32
	Contract *DlcContract
}

// NewDlcOfferMsg creates a new DlcOfferMsg based on a peer and contract
func NewDlcOfferMsg(peerIdx uint32, contract *DlcContract) DlcOfferMsg {
	msg := new(DlcOfferMsg)
	msg.PeerIdx = peerIdx
	msg.Contract = contract
	return *msg
}

// NewDlcOfferMsgFromBytes parses a byte array back into a DlcOfferMsg
func NewDlcOfferMsgFromBytes(b []byte, peerIDX uint32) (DlcOfferMsg, error) {
	var err error
	sm := new(DlcOfferMsg)
	sm.PeerIdx = peerIDX
	sm.Contract, err = DlcContractFromBytes(b[1:])
	if err != nil {
		return *sm, err
	}
	return *sm, nil
}

// Bytes serializes a DlcOfferMsg into a byte array
func (msg DlcOfferMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	buf.Write(msg.Contract.Bytes())

	return buf.Bytes()
}

// Peer returns the peer index this message was received from/sent to
func (msg DlcOfferMsg) Peer() uint32 { return msg.PeerIdx }

// MsgType returns the type of this message
func (msg DlcOfferMsg) MsgType() uint8 { return MSGID_DLC_OFFER }

type DlcOfferDeclineMsg struct {
	PeerIdx uint32
	Idx     uint64 // The contract we are declining
	Reason  uint8  // Reason for declining the funding request

}

// NewDlcOfferDeclineMsg creates a new DlcOfferDeclineMsg based on a peer, a
// reason for declining and the index of the contract we're declining
func NewDlcOfferDeclineMsg(peerIdx uint32, reason uint8,
	theirIdx uint64) DlcOfferDeclineMsg {
	msg := new(DlcOfferDeclineMsg)
	msg.PeerIdx = peerIdx
	msg.Reason = reason
	msg.Idx = theirIdx
	return *msg
}

// NewDlcOfferDeclineMsgFromBytes deserializes a byte array into a
// DlcOfferDeclineMsg
func NewDlcOfferDeclineMsgFromBytes(b []byte,
	peerIdx uint32) (DlcOfferDeclineMsg, error) {

	msg := new(DlcOfferDeclineMsg)
	msg.PeerIdx = peerIdx

	if len(b) < 2 {
		return *msg, fmt.Errorf("DlcOfferDeclineMsg %d bytes, expect at"+
			" least 2", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	_ = binary.Read(buf, binary.BigEndian, &msg.Reason)
	msg.Idx, _ = wire.ReadVarInt(buf, 0)

	return *msg, nil
}

// Bytes serializes a DlcOfferDeclineMsg into a byte array
func (msg DlcOfferDeclineMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())

	binary.Write(&buf, binary.BigEndian, msg.Reason)
	wire.WriteVarInt(&buf, 0, msg.Idx)
	return buf.Bytes()
}

// Peer returns the peer index this message was received from/sent to
func (msg DlcOfferDeclineMsg) Peer() uint32 { return msg.PeerIdx }

// MsgType returns the type of this message
func (msg DlcOfferDeclineMsg) MsgType() uint8 { return MSGID_DLC_DECLINEOFFER }

// DlcContractSettlementSignature contains the signature for a particular
// settlement transaction
type DlcContractSettlementSignature struct {
	// The oracle value for which transaction these are the signatures
	Outcome int64
	// The signature for the transaction
	Signature [64]byte
}

// DlcOfferAcceptMsg is a message indicating we are accepting the contract
type DlcOfferAcceptMsg struct {
	// Index of the peer we are forming the contract with
	PeerIdx uint32
	// The index of the contract on the peer we're receiving this message on
	Idx uint64
	// The index of the contract on our side, so they know how to reference it
	OurIdx uint64
	// The PKH we want the change from funding to be paid back to
	OurChangePKH [20]byte
	// The Pubkey that is part of the multisig for spending the contract funds
	OurFundMultisigPub [33]byte
	// The Pubkey to be used to in the contract settlement
	OurPayoutBase [33]byte
	//OurRevokePub [33]byte
	OurRefundPKH [20]byte
	OurrefundTxSig64 [64]byte
	// The PKH to be paid to in the contract settlement
	OurPayoutPKH [20]byte
	// The UTXOs we are using to fund the contract
	FundingInputs []DlcContractFundingInput
	// The signatures for settling the contract at various values
	SettlementSignatures []DlcContractSettlementSignature
}

// NewDlcOfferAcceptMsg generates a new DlcOfferAcceptMsg struct based on the
// passed contract and signatures
func NewDlcOfferAcceptMsg(contract *DlcContract,
	signatures []DlcContractSettlementSignature) DlcOfferAcceptMsg {

	msg := new(DlcOfferAcceptMsg)
	msg.PeerIdx = contract.PeerIdx
	msg.Idx = contract.TheirIdx
	msg.OurIdx = contract.Idx
	msg.FundingInputs = contract.OurFundingInputs
	msg.OurChangePKH = contract.OurChangePKH
	msg.OurFundMultisigPub = contract.OurFundMultisigPub
	msg.OurPayoutBase = contract.OurPayoutBase
	msg.OurRefundPKH = contract.OurRefundPKH
	msg.OurrefundTxSig64 = contract.OurrefundTxSig64
	msg.OurPayoutPKH = contract.OurPayoutPKH
	msg.SettlementSignatures = signatures
	return *msg
}

// NewDlcOfferAcceptMsgFromBytes parses a byte array back into a
// DlcOfferAcceptMsg struct
func NewDlcOfferAcceptMsgFromBytes(b []byte,
	peerIdx uint32) (DlcOfferAcceptMsg, error) {

	msg := new(DlcOfferAcceptMsg)
	msg.PeerIdx = peerIdx

	if len(b) < 34 {
		return *msg, fmt.Errorf("DlcOfferAcceptMsg %d bytes, expect at"+
			" least 34", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	msg.Idx, _ = wire.ReadVarInt(buf, 0)
	msg.OurIdx, _ = wire.ReadVarInt(buf, 0)

	copy(msg.OurChangePKH[:], buf.Next(20))
	copy(msg.OurFundMultisigPub[:], buf.Next(33))
	copy(msg.OurPayoutBase[:], buf.Next(33))
	copy(msg.OurRefundPKH[:], buf.Next(20))
	copy(msg.OurrefundTxSig64[:], buf.Next(64))
	copy(msg.OurPayoutPKH[:], buf.Next(20))

	inputCount, _ := wire.ReadVarInt(buf, 0)

	msg.FundingInputs = make([]DlcContractFundingInput, inputCount)
	var op [36]byte
	for i := uint64(0); i < inputCount; i++ {
		val, _ := wire.ReadVarInt(buf, 0)
		msg.FundingInputs[i].Value = int64(val)
		copy(op[:], buf.Next(36))
		msg.FundingInputs[i].Outpoint = *OutPointFromBytes(op)

	}

	sigCount, _ := wire.ReadVarInt(buf, 0)
	msg.SettlementSignatures = make([]DlcContractSettlementSignature, sigCount)

	for i := uint64(0); i < sigCount; i++ {
		val, _ := wire.ReadVarInt(buf, 0)
		msg.SettlementSignatures[i].Outcome = int64(val)
		copy(msg.SettlementSignatures[i].Signature[:], buf.Next(64))
	}

	return *msg, nil
}

// Bytes turns a DlcOfferAcceptMsg into bytes
func (msg DlcOfferAcceptMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())

	wire.WriteVarInt(&buf, 0, msg.Idx)
	wire.WriteVarInt(&buf, 0, msg.OurIdx)

	buf.Write(msg.OurChangePKH[:])
	buf.Write(msg.OurFundMultisigPub[:])
	buf.Write(msg.OurPayoutBase[:])
	buf.Write(msg.OurRefundPKH[:])
	buf.Write(msg.OurrefundTxSig64[:])
	buf.Write(msg.OurPayoutPKH[:])

	inputCount := uint64(len(msg.FundingInputs))
	wire.WriteVarInt(&buf, 0, inputCount)

	for i := uint64(0); i < inputCount; i++ {
		wire.WriteVarInt(&buf, 0, uint64(msg.FundingInputs[i].Value))
		op := OutPointToBytes(msg.FundingInputs[i].Outpoint)
		buf.Write(op[:])
	}

	signatureCount := uint64(len(msg.SettlementSignatures))
	wire.WriteVarInt(&buf, 0, signatureCount)

	for i := uint64(0); i < signatureCount; i++ {
		wire.WriteVarInt(&buf, 0, uint64(msg.SettlementSignatures[i].Outcome))
		buf.Write(msg.SettlementSignatures[i].Signature[:])
	}
	return buf.Bytes()
}

// Peer returns the peer index this message was received from/sent to
func (msg DlcOfferAcceptMsg) Peer() uint32 {
	return msg.PeerIdx
}

// MsgType returns the type of this message
func (msg DlcOfferAcceptMsg) MsgType() uint8 {
	return MSGID_DLC_ACCEPTOFFER
}

// DlcContractAckMsg is sent from the offering party back to the peer when the
// contract acceptance is acknowledged. Includes the signatures from this peer
// for the settlement TXes.
type DlcContractAckMsg struct {
	// Peer we're sending the Ack to (or received it from)
	PeerIdx uint32
	// The index of the contract we're acknowledging
	Idx uint64
	// The settlement signatures of the party acknowledging
	SettlementSignatures []DlcContractSettlementSignature
	OurrefundTxSig64 [64]byte
}

// NewDlcContractAckMsg generates a new DlcContractAckMsg struct based on the
// passed contract and signatures
func NewDlcContractAckMsg(contract *DlcContract,
	signatures []DlcContractSettlementSignature, OurrefundTxSig64 [64]byte) DlcContractAckMsg {

	msg := new(DlcContractAckMsg)
	msg.PeerIdx = contract.PeerIdx
	msg.Idx = contract.TheirIdx
	msg.SettlementSignatures = signatures
	msg.OurrefundTxSig64 = OurrefundTxSig64
	return *msg
}

// NewDlcContractAckMsgFromBytes deserializes a byte array into a
// DlcContractAckMsg
func NewDlcContractAckMsgFromBytes(b []byte,
	peerIdx uint32) (DlcContractAckMsg, error) {

	msg := new(DlcContractAckMsg)
	msg.PeerIdx = peerIdx

	// TODO
	if len(b) < 34 {
		return *msg, fmt.Errorf("DlcContractAckMsg %d bytes, expect at"+
			" least 34", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	msg.Idx, _ = wire.ReadVarInt(buf, 0)

	var sigCount uint32
	binary.Read(buf, binary.BigEndian, &sigCount)
	msg.SettlementSignatures = make([]DlcContractSettlementSignature, sigCount)

	for i := uint32(0); i < sigCount; i++ {
		binary.Read(buf, binary.BigEndian, &msg.SettlementSignatures[i].Outcome)
		copy(msg.SettlementSignatures[i].Signature[:], buf.Next(64))
	}

	copy(msg.OurrefundTxSig64[:], buf.Next(64))

	return *msg, nil
}

// Bytes serializes a DlcContractAckMsg into a byte array
func (msg DlcContractAckMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	wire.WriteVarInt(&buf, 0, msg.Idx)

	signatureCount := uint32(len(msg.SettlementSignatures))
	binary.Write(&buf, binary.BigEndian, signatureCount)

	for i := uint32(0); i < signatureCount; i++ {
		outcome := msg.SettlementSignatures[i].Outcome
		binary.Write(&buf, binary.BigEndian, outcome)
		buf.Write(msg.SettlementSignatures[i].Signature[:])
	}

	buf.Write(msg.OurrefundTxSig64[:])

	return buf.Bytes()
}

// Peer returns the peer index this message was received from/sent to
func (msg DlcContractAckMsg) Peer() uint32 {
	return msg.PeerIdx
}

// MsgType returns the type of this message
func (msg DlcContractAckMsg) MsgType() uint8 {
	return MSGID_DLC_CONTRACTACK
}

// DlcContractFundingSigsMsg is sent by the counter party once the signatures
// for the settlement are verified and accepted. These signatures can be used
// to spend the peer's UTXOs for funding the contract into the actual contract
// output.
type DlcContractFundingSigsMsg struct {
	PeerIdx         uint32      // Peer we're exchanging the message with
	Idx             uint64      // The index of the concerning contract
	SignedFundingTx *wire.MsgTx // The funding TX containing the signatures
}

// NewDlcContractFundingSigsMsg creates a new DlcContractFundingSigsMsg based
// on the passed contract and signed funding TX
func NewDlcContractFundingSigsMsg(contract *DlcContract,
	signedTx *wire.MsgTx) DlcContractFundingSigsMsg {

	msg := new(DlcContractFundingSigsMsg)
	msg.PeerIdx = contract.PeerIdx
	msg.Idx = contract.TheirIdx
	msg.SignedFundingTx = signedTx
	return *msg
}

// NewDlcContractFundingSigsMsgFromBytes deserializes a byte array into a
// DlcContractFundingSigsMsg
func NewDlcContractFundingSigsMsgFromBytes(b []byte,
	peerIdx uint32) (DlcContractFundingSigsMsg, error) {

	msg := new(DlcContractFundingSigsMsg)
	msg.PeerIdx = peerIdx

	// TODO
	if len(b) < 34 {
		return *msg, fmt.Errorf("DlcContractFundingSigsMsg %d bytes, expect"+
			"at least 34", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	msg.Idx, _ = wire.ReadVarInt(buf, 0)

	msg.SignedFundingTx = wire.NewMsgTx()
	msg.SignedFundingTx.Deserialize(buf)

	return *msg, nil
}

// Bytes serializes a DlcContractFundingSigsMsg into a byte array
func (msg DlcContractFundingSigsMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	wire.WriteVarInt(&buf, 0, msg.Idx)

	writer := bufio.NewWriter(&buf)
	msg.SignedFundingTx.Serialize(writer)
	writer.Flush()
	return buf.Bytes()
}

// Peer returns the peer index this message was received from/sent to
func (msg DlcContractFundingSigsMsg) Peer() uint32 {
	return msg.PeerIdx
}

// MsgType returns the type of this message
func (msg DlcContractFundingSigsMsg) MsgType() uint8 {
	return MSGID_DLC_CONTRACTFUNDINGSIGS
}

// DlcContractSigProofMsg acknowledges the funding of the contract to a peer.
// It contains the fully signed funding transaction that has already been
// published to the blockchain
type DlcContractSigProofMsg struct {
	// The index of the peer we're communicating with
	PeerIdx uint32
	// The contract we're communicating about
	Idx uint64
	// The fully signed funding transaction
	SignedFundingTx *wire.MsgTx
}

// NewDlcContractSigProofMsg creates a new DlcContractSigProofMsg based on the
// passed contract and signed funding TX
func NewDlcContractSigProofMsg(contract *DlcContract,
	signedTx *wire.MsgTx) DlcContractSigProofMsg {

	msg := new(DlcContractSigProofMsg)
	msg.PeerIdx = contract.PeerIdx
	msg.Idx = contract.TheirIdx
	msg.SignedFundingTx = signedTx
	return *msg
}

// NewDlcContractSigProofMsgFromBytes deserializes a byte array into a
// DlcContractSigProofMsg
func NewDlcContractSigProofMsgFromBytes(b []byte,
	peerIdx uint32) (DlcContractSigProofMsg, error) {

	msg := new(DlcContractSigProofMsg)
	msg.PeerIdx = peerIdx

	// TODO
	if len(b) < 34 {
		return *msg, fmt.Errorf("DlcContractSigProofMsg %d bytes, expect"+
			" at least 34", len(b))
	}

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	msg.Idx, _ = wire.ReadVarInt(buf, 0)

	msg.SignedFundingTx = wire.NewMsgTx()
	msg.SignedFundingTx.Deserialize(buf)

	return *msg, nil
}

// Bytes serializes a DlcContractSigProofMsg into a byte array
func (msg DlcContractSigProofMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	wire.WriteVarInt(&buf, 0, msg.Idx)

	writer := bufio.NewWriter(&buf)
	msg.SignedFundingTx.Serialize(writer)
	writer.Flush()
	return buf.Bytes()
}

// Peer returns the peer index this message was received from/sent to
func (msg DlcContractSigProofMsg) Peer() uint32 {
	return msg.PeerIdx
}

// MsgType returns the type of this message
func (msg DlcContractSigProofMsg) MsgType() uint8 {
	return MSGID_DLC_SIGPROOF
}

// MultihopPaymentRequestMsg initiates a new multihop payment. It is sent to
// the peer that will ultimately receive the payment.
type MultihopPaymentRequestMsg struct {
	// The index of the peer we're communicating with
	PeerIdx uint32
	// The type of coin we're requesting to send
	Cointype uint32
}

func NewMultihopPaymentRequestMsg(peerIdx uint32, cointype uint32) MultihopPaymentRequestMsg {
	msg := new(MultihopPaymentRequestMsg)
	msg.PeerIdx = peerIdx
	msg.Cointype = cointype
	return *msg
}

func NewMultihopPaymentRequestMsgFromBytes(b []byte,
	peerIdx uint32) (MultihopPaymentRequestMsg, error) {

	msg := new(MultihopPaymentRequestMsg)
	msg.PeerIdx = peerIdx

	buf := bytes.NewBuffer(b[1:])

	err := binary.Read(buf, binary.BigEndian, &msg.Cointype)
	if err != nil {
		return *msg, err
	}

	return *msg, nil
}

// Bytes serializes a MultihopPaymentRequestMsg into a byte array
func (msg MultihopPaymentRequestMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	binary.Write(&buf, binary.BigEndian, msg.Cointype)
	return buf.Bytes()
}

// Peer returns the peer index this message was received from/sent to
func (msg MultihopPaymentRequestMsg) Peer() uint32 {
	return msg.PeerIdx
}

// MsgType returns the type of this message
func (msg MultihopPaymentRequestMsg) MsgType() uint8 {
	return MSGID_PAY_REQ
}

// MultihopPaymentRequestMsg initiates a new multihop payment. It is sent to
// the peer that will ultimately send the payment.
type MultihopPaymentAckMsg struct {
	// The index of the peer we're communicating with
	PeerIdx uint32
	// The hash to the preimage we use to clear out the HTLCs
	HHash [32]byte
}

func NewMultihopPaymentAckMsg(peerIdx uint32, hHash [32]byte) MultihopPaymentAckMsg {
	msg := new(MultihopPaymentAckMsg)
	msg.PeerIdx = peerIdx
	msg.HHash = hHash
	return *msg
}

func NewMultihopPaymentAckMsgFromBytes(b []byte,
	peerIdx uint32) (MultihopPaymentAckMsg, error) {

	msg := new(MultihopPaymentAckMsg)
	msg.PeerIdx = peerIdx
	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	copy(msg.HHash[:], buf.Next(32))
	return *msg, nil
}

// Bytes serializes a MultihopPaymentAckMsg into a byte array
func (msg MultihopPaymentAckMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	buf.Write(msg.HHash[:])

	return buf.Bytes()
}

func (msg MultihopPaymentAckMsg) Peer() uint32 {
	return msg.PeerIdx
}

// MsgType returns the type of this message
func (msg MultihopPaymentAckMsg) MsgType() uint8 {
	return MSGID_PAY_ACK
}

// RemoteControlRpcRequestMsg contains a request to be executed by the local
// lit node, when the remote node has been authorized to do so.
type RemoteControlRpcRequestMsg struct {
	PeerIdx uint32
	// The pubkey of the node remote controlling. Can be null, in which case
	// the pubkey of the peer sending the message is used to determine the
	// authorization
	PubKey [33]byte

	// The method being called, for example "LitRPC.Send"
	Method string

	// A unique nonce that will be used to match the response that is sent
	// back in reply to this request.
	Idx uint64

	// The JSON serialized arguments to the RPC method
	Args []byte

	// If PubKey is passed, this should contain a signature made with the
	// corresponding private key of the Bytes() method of this message type,
	// containing a zero Sig
	Sig [64]byte

	// The digest used for the signature. Can be one of
	// DIGEST_TYPE_SHA256    = 0x00 (Default)
	// DIGEST_TYPE_RIPEMD160 = 0x01
	// Different digest is supported to allow the use of embedded devices
	// such as smart cards that do not support signing SHA256 digests.
	// They exist, really.
	DigestType uint8
}

func NewRemoteControlRpcRequestMsgFromBytes(b []byte,
	peerIdx uint32) (RemoteControlRpcRequestMsg, error) {

	msg := new(RemoteControlRpcRequestMsg)
	msg.PeerIdx = peerIdx

	buf := bytes.NewBuffer(b[1:])
	copy(msg.PubKey[:], buf.Next(33))
	copy(msg.Sig[:], buf.Next(64))
	binary.Read(buf, binary.BigEndian, &msg.DigestType)
	binary.Read(buf, binary.BigEndian, &msg.Idx)

	methodLength, _ := wire.ReadVarInt(buf, 0)
	msg.Method = string(buf.Next(int(methodLength)))

	argsLength, _ := wire.ReadVarInt(buf, 0)
	msg.Args = buf.Next(int(argsLength))
	return *msg, nil
}

// Bytes serializes a RemoteControlRpcRequestMsg into a byte array
func (msg RemoteControlRpcRequestMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	buf.Write(msg.PubKey[:])
	buf.Write(msg.Sig[:])
	binary.Write(&buf, binary.BigEndian, msg.DigestType)
	binary.Write(&buf, binary.BigEndian, msg.Idx)

	methodBytes := []byte(msg.Method)
	wire.WriteVarInt(&buf, 0, uint64(len(methodBytes)))
	buf.Write(methodBytes)

	wire.WriteVarInt(&buf, 0, uint64(len(msg.Args)))
	buf.Write(msg.Args)

	return buf.Bytes()
}

// Peer returns the peer index this message was received from/sent to
func (msg RemoteControlRpcRequestMsg) Peer() uint32 {
	return msg.PeerIdx
}

func (msg RemoteControlRpcRequestMsg) MsgType() uint8 {
	return MSGID_REMOTE_RPCREQUEST
}

type RouteHop struct {
	Node     [20]byte
	CoinType uint32
}

func (rh *RouteHop) Bytes() []byte {
	var buf bytes.Buffer

	buf.Write(rh.Node[:])
	binary.Write(&buf, binary.BigEndian, rh.CoinType)

	return buf.Bytes()
}

func NewRouteHopFromBytes(b []byte) (*RouteHop, error) {
	buf := bytes.NewBuffer(b)

	if buf.Len() < 24 {
		return nil, fmt.Errorf("not enough bytes for RouteHop")
	}

	rh := new(RouteHop)

	copy(rh.Node[:], buf.Next(20))

	err := binary.Read(buf, binary.BigEndian, &rh.CoinType)
	if err != nil {
		return nil, err
	}

	return rh, nil
}

// MultihopPaymentSetupMsg forms a new multihop payment. It is sent to
// the next-in-line peer, which will forward it to the next hop until
// the target is reached
type MultihopPaymentSetupMsg struct {
	// The index of the peer we're communicating with
	PeerIdx uint32
	// The hash to the preimage we use to clear out the HTLCs
	HHash [32]byte
	// The PKHs (in order) of the nodes we're using.
	NodeRoute []RouteHop
	// Data associated with the payment
	Data [32]byte
}

// NewMultihopPaymentSetupMsg does...
func NewMultihopPaymentSetupMsg(peerIdx uint32, hHash [32]byte, nodeRoute []RouteHop, data [32]byte) MultihopPaymentSetupMsg {
	msg := new(MultihopPaymentSetupMsg)
	msg.PeerIdx = peerIdx
	msg.HHash = hHash
	msg.NodeRoute = nodeRoute
	msg.Data = data
	return *msg
}

func NewMultihopPaymentSetupMsgFromBytes(b []byte,
	peerIdx uint32) (MultihopPaymentSetupMsg, error) {

	msg := new(MultihopPaymentSetupMsg)
	msg.PeerIdx = peerIdx
	buf := bytes.NewBuffer(b[1:]) // get rid of messageType
	copy(msg.HHash[:], buf.Next(32))

	hops, _ := wire.ReadVarInt(buf, 0)
	for i := uint64(0); i < hops; i++ {
		rh, err := NewRouteHopFromBytes(buf.Next(24))
		if err != nil {
			return *msg, err
		}

		msg.NodeRoute = append(msg.NodeRoute, *rh)
	}

	copy(msg.Data[:], buf.Next(32))

	return *msg, nil
}

// Bytes serializes a MultihopPaymentSetupMsg into a byte array
func (msg MultihopPaymentSetupMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	buf.Write(msg.HHash[:])
	wire.WriteVarInt(&buf, 0, uint64(len(msg.NodeRoute)))
	for _, nd := range msg.NodeRoute {
		buf.Write(nd.Bytes())
	}

	buf.Write(msg.Data[:])

	return buf.Bytes()
}

func (msg MultihopPaymentSetupMsg) Peer() uint32 {
	return msg.PeerIdx
}

// MsgType returns the type of this message
func (msg MultihopPaymentSetupMsg) MsgType() uint8 {
	return MSGID_PAY_SETUP
}

// RemoteControlRpcResponseMsg is sent in response to a request message
// and contains the output of the command that was executed
type RemoteControlRpcResponseMsg struct {
	PeerIdx uint32
	Idx     uint64 // Unique nonce that was sent in the request
	Error   bool   // Indicates that the reply is an error
	Result  []byte // JSON serialization of the reply object
}

func NewRemoteControlRpcResponseMsg(peerIdx uint32, msgIdx uint64, isError bool, json []byte) RemoteControlRpcResponseMsg {

	msg := new(RemoteControlRpcResponseMsg)
	msg.PeerIdx = peerIdx
	msg.Idx = msgIdx
	msg.Error = isError
	msg.Result = json
	return *msg
}

func NewRemoteControlRpcResponseMsgFromBytes(b []byte,
	peerIdx uint32) (RemoteControlRpcResponseMsg, error) {

	msg := new(RemoteControlRpcResponseMsg)
	buf := bytes.NewBuffer(b[1:])

	msg.PeerIdx = peerIdx
	binary.Read(buf, binary.BigEndian, &msg.Idx)
	binary.Read(buf, binary.BigEndian, &msg.Error)

	resultLength, _ := wire.ReadVarInt(buf, 0)
	msg.Result = buf.Next(int(resultLength))

	return *msg, nil
}

// Bytes serializes a RemoteControlRpcRequestMsg into a byte array
func (msg RemoteControlRpcResponseMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	binary.Write(&buf, binary.BigEndian, msg.Idx)
	binary.Write(&buf, binary.BigEndian, msg.Error)

	wire.WriteVarInt(&buf, 0, uint64(len(msg.Result)))
	buf.Write(msg.Result)
	return buf.Bytes()
}

// Peer returns the peer index this message was received from/sent to
func (msg RemoteControlRpcResponseMsg) Peer() uint32 {
	return msg.PeerIdx
}

// MsgType returns the type of this message
func (msg RemoteControlRpcResponseMsg) MsgType() uint8 {
	return MSGID_REMOTE_RPCRESPONSE
}


// For chunked messages

type BeginChunksMsg struct {
	PeerIdx uint32
	TimeStamp int64
}

func NewChunksBeginMsgFromBytes(b []byte, peerIdx uint32) (BeginChunksMsg, error) {

	msg := new(BeginChunksMsg)
	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	msg.PeerIdx = peerIdx
	binary.Read(buf, binary.BigEndian, &msg.TimeStamp)

	return *msg, nil
}

func (msg BeginChunksMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	binary.Write(&buf, binary.BigEndian, msg.TimeStamp)

	return buf.Bytes()
}

func (msg BeginChunksMsg) Peer() uint32 {
	return msg.PeerIdx
}

func (msg BeginChunksMsg) MsgType() uint8 {
	return MSGID_CHUNKS_BEGIN
}


type ChunkMsg struct {
	PeerIdx uint32
	TimeStamp int64
	ChunkSize int32
	Data []byte
}


func NewChunkMsgFromBytes(b []byte, peerIdx uint32) (ChunkMsg, error) {

	msg := new(ChunkMsg)

	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	msg.PeerIdx = peerIdx
	binary.Read(buf, binary.BigEndian, &msg.TimeStamp)
	binary.Read(buf, binary.BigEndian, &msg.ChunkSize)

	msg.Data = make([]byte, msg.ChunkSize)
	binary.Read(buf, binary.BigEndian, msg.Data)

	return *msg, nil

}


func (msg ChunkMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	binary.Write(&buf, binary.BigEndian, msg.TimeStamp)
	binary.Write(&buf, binary.BigEndian, msg.ChunkSize)
	buf.Write(msg.Data)

	return buf.Bytes()
} 

func (msg ChunkMsg) Peer() uint32 {
	return msg.PeerIdx
}

func (msg ChunkMsg) MsgType() uint8 {
	return MSGID_CHUNK_BODY
}



type EndChunksMsg struct {
	PeerIdx uint32
	TimeStamp int64
}

func NewChunksEndMsgFromBytes(b []byte, peerIdx uint32) (EndChunksMsg, error) {


	msg := new(EndChunksMsg)
	buf := bytes.NewBuffer(b[1:]) // get rid of messageType

	msg.PeerIdx = peerIdx
	binary.Read(buf, binary.BigEndian, &msg.TimeStamp)

	return *msg, nil
}

func (msg EndChunksMsg) Bytes() []byte {
	var buf bytes.Buffer

	buf.WriteByte(msg.MsgType())
	binary.Write(&buf, binary.BigEndian, msg.TimeStamp)

	return buf.Bytes()
}

func (msg EndChunksMsg) Peer() uint32 {
	return msg.PeerIdx
}

func (msg EndChunksMsg) MsgType() uint8 {
	return MSGID_CHUNKS_END
}