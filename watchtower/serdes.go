package watchtower

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	// desc describes a new channel
	MSGID_WATCH_DESC = 0xA0
	// commsg is a single state in the channel
	MSGID_WATCH_COMMSG = 0xA1
)

// Descriptors are 128 bytes
// PKH 20
// Delay 2
// Fee 8
// HAKDbase 33
// Timebase 33
// Elk0 32

// ToBytes turns a WatchannelDescriptor into 100 bytes
func (sd *WatchannelDescriptor) ToBytes() []byte {
	var buf bytes.Buffer
	buf.Write(sd.DestPKHScript[:])
	binary.Write(&buf, binary.BigEndian, sd.Delay)
	binary.Write(&buf, binary.BigEndian, sd.Fee)
	buf.Write(sd.HAKDBasePoint[:])
	buf.Write(sd.TimeBasePoint[:])
	buf.Write(sd.ElkZero.CloneBytes())
	return buf.Bytes()
}

// WatchannelDescriptorFromBytes turns 96 or 128 bytes into a WatchannelDescriptor
func WatchannelDescriptorFromBytes(b []byte) (WatchannelDescriptor, error) {
	var sd WatchannelDescriptor
	if len(b) != 128 && len(b) != 96 {
		return sd, fmt.Errorf("WatchannelDescriptor %d bytes, expect 128", len(b))
	}
	buf := bytes.NewBuffer(b)

	copy(sd.DestPKHScript[:], buf.Next(20))
	err := binary.Read(buf, binary.BigEndian, &sd.Delay)
	if err != nil {
		return sd, err
	}
	err = binary.Read(buf, binary.BigEndian, &sd.Fee)
	if err != nil {
		return sd, err
	}

	copy(sd.HAKDBasePoint[:], buf.Next(33))
	copy(sd.TimeBasePoint[:], buf.Next(33))
	// might not be anything left, which is OK, elk0 will just be blank
	copy(sd.ElkZero[:], buf.Next(32))

	return sd, nil
}

// ComMsg are 132 bytes.
// PKH 20
// txid 16
// elk 32
// sig 64
// ToBytes turns a ComMsg into 132 bytes
func (sm *ComMsg) ToBytes() (b [132]byte) {
	var buf bytes.Buffer
	buf.Write(sm.DestPKHScript[:])
	buf.Write(sm.Txid[:])
	buf.Write(sm.Elk.CloneBytes())
	buf.Write(sm.Sig[:])
	copy(b[:], buf.Bytes())
	return
}

// ComMsgFromBytes turns 112 bytes into a SorceMsg
func ComMsgFromBytes(b [128]byte) ComMsg {
	var sm ComMsg
	copy(sm.Txid[:], b[:16])
	copy(sm.Elk[:], b[16:48])
	copy(sm.Sig[:], b[48:])
	return sm
}

// IdxSigs are 74 bytes
// PKHIdx 4
// StateIdx 6
// Sig 64

// no idxSig to bytes function -- done inline in the addMsg db call

//type IdxSig struct {
//	PKHIdx   uint32
//	StateIdx uint64
//	Sig      [64]byte
//}
