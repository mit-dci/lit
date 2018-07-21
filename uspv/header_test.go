package uspv

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/wire"
	"github.com/mit-dci/lit/coinparam"
	."github.com/mit-dci/lit/logs"
	"testing"
	"time"
)

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

func TestMoreWork(t *testing.T) {
	var a []*wire.BlockHeader
	var b []*wire.BlockHeader
	p := &coinparam.BitcoinParams
	// 0000000000dd4dd73d78e8fd29ba2fd2ed618bd6fa2ee92559f5 42fdb26e7c1d
	for j := 0; j < 2016; j++ {
		header := "0000000000dd4dd73d78e8fd29ba2fd2ed618bd6fa2ee92559f5" + randomHex(6)
		testHash, _ := chainhash.NewHashFromStr(header)
		temp := wire.BlockHeader{
			Version:    32,
			PrevBlock:  *testHash,
			MerkleRoot: *testHash,
			Timestamp:  time.Unix(0x495fab29, 0),
			Bits:       uint32(0x1d00ffff), // let just make up some stuff over here
			Nonce:      uint32(0x1d00aaaa), // all invalid
		}
		a = append(a, &temp)
		b = append(b, &temp)
	}
	header := "0000000000dd4dd73d78e8fd29ba2fd2ed618bd6fa2ee92559f5" + randomHex(6)
	// Chain A -> 2016 -> header1 -> header3 -> header4
	// Chain B -> 2016 -> header2
	testHash, _ := chainhash.NewHashFromStr(header)

	temp1 := wire.BlockHeader{
		Version:    32,
		PrevBlock:  *testHash,
		MerkleRoot: *testHash,
		Timestamp:  time.Unix(0x495fab29, 0),
		Bits:       uint32(0x1d000ff1), // let just make up some stuff over here
		Nonce:      uint32(0x1d00aaaa), // all invalid
	}
	a = append(a, &temp1)

	temp2 := wire.BlockHeader{
		Version:    32,
		PrevBlock:  *testHash,
		MerkleRoot: *testHash,
		Timestamp:  time.Unix(0x495fab29, 0),
		Bits:       uint32(0x1d000ff2), // let just make up some stuff over here
		Nonce:      uint32(0x1d00aaaa), // all invalid
	}
	a = append(a, &temp2)

	temp3 := wire.BlockHeader{
		Version:    32,
		PrevBlock:  *testHash,
		MerkleRoot: *testHash,
		Timestamp:  time.Unix(0x495fab29, 0),
		Bits:       uint32(0x1d000ff3), // let just make up some stuff over here
		Nonce:      uint32(0x1d00aaaa), // all invalid
	}
	a = append(a, &temp3)

	temp4 := wire.BlockHeader{
		Version:    32,
		PrevBlock:  *testHash,
		MerkleRoot: *testHash,
		Timestamp:  time.Unix(0x495fab29, 0),
		Bits:       uint32(0x1d0000f1), // let just make up some stuff over here
		Nonce:      uint32(0x1d00aaaa), // all invalid
	}

	b = append(b, &temp4)
	// Work of A: 206916179889
	// WOrk of B: 1167945961455

	if moreWork(a, b, p) {
		Log.Error("Test failed!!")
		t.Fatal()
	} else {
		Log.Info("Test Passed!")
	}
}
