package lnutil

// all the messages to and from peers look like this internally
type Msg interface {
	setData(...[]byte)
}
type LitMsg struct { // RETRACTED -> remove then fix errors caused
	PeerIdx uint32
	ChanIdx uint32 // optional, may be 0
	MsgType uint8
	Data    []byte
}

type NewLitMsg interface {
	PeerIdx() uint32
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
	peerIdx   uint32
	outpoint  [36]byte
	delta     [4]byte
	signature [64]byte
}

func NewDeltaSigMsg(peerid uint32, OP []byte, DELTA []byte, SIG []byte) (*DeltaSigMsg, error) {
	d := new(DeltaSigMsg)
	d.peerIdx = peerid
	err := d.setData(OP, DELTA, SIG)
	return d, err
}
func (self *DeltaSigMsg) setData(OP []byte, DELTA []byte, SIG []byte) error {
	copy(self.outpoint[:], OP[:])
	copy(self.delta[:], DELTA[:])
	copy(self.signature[:], SIG[:])
	return nil
}

func (self *DeltaSigMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.outpoint[:]...)
	msg = append(msg, self.delta[:]...)
	msg = append(msg, self.signature[:]...)
	return msg
}

func (self *DeltaSigMsg) PeerIdx() uint32 { return self.peerIdx }
func (self *DeltaSigMsg) MsgType() uint32 { return MSGID_DELTASIG }

//----------

type SigRevMsg struct {
	peerIdx    uint32
	outpoint   [36]byte
	signature  [64]byte
	elk        [32]byte
	n2ElkPoint [33]byte
}

func NewSigRev(peerid uint32, OP [36]byte, SIG [64]byte, ELK []byte, N2ELK [33]byte) (*SigRevMsg, error) {
	s := new(SigRevMsg)
	s.peerIdx = peerid
	err := s.setData(OP, SIG, ELK, N2ELK)
	return s, err
}

func (self *SigRevMsg) setData(OP [36]byte, SIG [64]byte, ELK []byte, N2ELK [33]byte) error {
	copy(self.outpoint[:], OP[:])
	copy(self.signature[:], SIG[:])
	copy(self.elk[:], ELK[:])
	copy(self.n2ElkPoint[:], N2ELK[:])
	// add some way of checking lengths
	return nil
}

func (self *SigRevMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.outpoint[:]...)
	msg = append(msg, self.signature[:]...)
	msg = append(msg, self.elk[:]...)
	msg = append(msg, self.n2ElkPoint[:]...)
	return msg
}

func (self *SigRevMsg) PeerIdx() uint32 { return self.peerIdx }
func (self *SigRevMsg) MsgType() uint32 { return MSGID_SIGREV }

//----------

type GapSigRevMsg struct {
	peerIdx    uint32
	outpoint   [36]byte
	signature  [64]byte
	elk        [32]byte
	n2ElkPoint [33]byte
}

func NewGapSigRev(peerid uint32, OP [36]byte, SIG [64]byte, ELK []byte, N2ELK [33]byte) (*GapSigRevMsg, error) {
	g := new(GapSigRevMsg)
	g.peerIdx = peerid
	err := g.setData(OP, SIG, ELK, N2ELK)
	return g, err
}

func (self *GapSigRevMsg) setData(OP [36]byte, SIG [64]byte, ELK []byte, N2ELK [33]byte) error {
	copy(self.outpoint[:], OP[:])
	copy(self.signature[:], SIG[:])
	copy(self.elk[:], ELK[:])
	copy(self.n2ElkPoint[:], N2ELK[:])
	// add some way of checking lengths
	return nil
}

func (self *GapSigRevMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.outpoint[:]...)
	msg = append(msg, self.signature[:]...)
	msg = append(msg, self.elk[:]...)
	msg = append(msg, self.n2ElkPoint[:]...)
	return msg
}

func (self *GapSigRevMsg) PeerIdx() uint32 { return self.peerIdx }
func (self *GapSigRevMsg) MsgType() uint32 { return MSGID_GAPSIGREV }

//----------

type RevMsg struct {
	peerIdx    uint32
	outpoint   [36]byte
	elk        [32]byte
	n2ElkPoint [33]byte
}

func NewRevMsg(peerid uint32, OP [36]byte, ELK []byte, N2ELK [33]byte) (*RevMsg, error) {
	r := new(RevMsg)
	r.peerIdx = peerid
	err := r.setData(OP, ELK, N2ELK)
	return r, err
}

func (self *RevMsg) setData(OP [36]byte, ELK []byte, N2ELK [33]byte) error {
	copy(self.outpoint[:], OP[:])
	copy(self.elk[:], ELK[:])
	copy(self.n2ElkPoint[:], N2ELK[:])
	return nil
}

func (self *RevMsg) Bytes() []byte {
	var msg []byte
	msg = append(msg, self.outpoint[:]...)
	msg = append(msg, self.elk[:]...)
	msg = append(msg, self.n2ElkPoint[:]...)
	return msg
}

func (self *RevMsg) PeerIdx() uint32 { return self.peerIdx }
func (self *RevMsg) MsgType() uint32 { return MSGID_REV } // maybe switch to use strings instead, probably better
//----------
