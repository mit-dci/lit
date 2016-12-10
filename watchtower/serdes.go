package watchtower

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/mit-dci/lit/lnutil"
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
	return buf.Bytes()
}

// WatchannelDescriptorFromBytes turns 96 bytes into a WatchannelDescriptor
// Silently fails with incorrect size input, watch out.
func WatchannelDescriptorFromBytes(b []byte) WatchannelDescriptor {
	var sd WatchannelDescriptor
	if len(b) != 96 {
		return sd
		//		return sd, fmt.Errorf(
		//			"WatchannelDescriptor %d bytes, expect 128 or 96", len(b))
	}
	buf := bytes.NewBuffer(b)

	copy(sd.DestPKHScript[:], buf.Next(20))
	_ = binary.Read(buf, binary.BigEndian, &sd.Delay)

	_ = binary.Read(buf, binary.BigEndian, &sd.Fee)

	copy(sd.CustomerBasePoint[:], buf.Next(33))
	copy(sd.AdversaryBasePoint[:], buf.Next(33))

	return sd
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

// ComMsgFromBytes turns 132 bytes into a SorceMsg
// Silently fails with wrong size input.
func ComMsgFromBytes(b []byte) ComMsg {
	var sm ComMsg
	if len(b) != 132 {
		return sm
	}
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
	// kindof ugly but fast; need 8 bytes, so give invalid high 2 bytes
	// then set them to 0 after we've cast to uint64
	s.StateIdx = lnutil.BtU64(b[2:10])
	s.StateIdx &= 0x0000ffffffffffff
	copy(s.Sig[:], b[10:])
	return &s, nil
}

//type IdxSig struct {
//	PKHIdx   uint32
//	StateIdx uint64
//	Sig      [64]byte
//}
