package watchtower

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/mit-dci/lit/lnutil"
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
	buf.Write(sd.CustomerBasePoint[:])
	buf.Write(sd.AdversaryBasePoint[:])
	buf.Write(sd.ElkZero.CloneBytes())
	return buf.Bytes()
}

// WatchannelDescriptorFromBytes turns 96 or 128 bytes into a WatchannelDescriptor
func WatchannelDescriptorFromBytes(b []byte) (WatchannelDescriptor, error) {
	var sd WatchannelDescriptor
	if len(b) != 128 && len(b) != 96 {
		return sd, fmt.Errorf(
			"WatchannelDescriptor %d bytes, expect 128 or 96", len(b))
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

	copy(sd.CustomerBasePoint[:], buf.Next(33))
	copy(sd.AdversaryBasePoint[:], buf.Next(33))
	// might not be anything left, which is OK, elk0 will just be blank
	copy(sd.ElkZero[:], buf.Next(32))

	return sd, nil
}

// ComMsg are 132 bytes.
// PKH 20
// txid 16
// sig 64
// elk 32
// ToBytes turns a ComMsg into 132 bytes
func (sm *ComMsg) ToBytes() (b [132]byte) {
	var buf bytes.Buffer
	buf.Write(sm.DestPKH[:])
	buf.Write(sm.ParTxid[:])
	buf.Write(sm.Sig[:])
	buf.Write(sm.Elk.CloneBytes())
	copy(b[:], buf.Bytes())
	return
}

// ComMsgFromBytes turns 112 bytes into a SorceMsg
func ComMsgFromBytes(b [128]byte) ComMsg {
	var sm ComMsg
	copy(sm.DestPKH[:], b[:20])
	copy(sm.ParTxid[:], b[20:36])
	copy(sm.Sig[:], b[36:100])
	copy(sm.Elk[:], b[100:])
	return sm
}

// IdxSigs are 74 bytes
// PKHIdx 4
// StateIdx 6
// Sig 64

// no idxSig to bytes function -- done inline in the addMsg db call

func IdxSigFromBytes(b []byte) (*IdxSig, error) {
	var s IdxSig
	if len(b) != 74 {
		return nil, fmt.Errorf("IdxSigFromBytes got %d bytes, expect 74", len(b))
	}
	s.PKHIdx = lnutil.BtU32(b[:4])
	s.StateIdx = uint64(lnutil.BtI64(b[4:12]))
	copy(s.Sig[:], b[12:])
	return &s, nil
}

//type IdxSig struct {
//	PKHIdx   uint32
//	StateIdx uint64
//	Sig      [64]byte
//}
