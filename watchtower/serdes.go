package watchtower

import (
	"fmt"
	"log"

	"github.com/mit-dci/lit/lnutil"
)

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
	log.Println("OUTPUT from IDSig", &s, b)
	return &s, nil
}

//type IdxSig struct {
//	PKHIdx   uint32
//	StateIdx uint64
//	Sig      [64]byte
//}
