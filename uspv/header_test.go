package uspv

import (
	"testing"
	"github.com/adiabat/btcd/wire"
	"log"
	"github.com/mit-dci/lit/coinparam"
 	"github.com/adiabat/btcd/chaincfg/chainhash"
	"github.com/adiabat/btcd/blockchain"
	"time"
)

func TestMoreWork(t *testing.T) {
	var a []*wire.BlockHeader
	var b []*wire.BlockHeader
	p := &coinparam.BitcoinParams
	testHash, _ := chainhash.NewHashFromStr("0000000000dd4dd73d78e8fd29ba2fd2ed618bd6fa2ee92559f542fdb26e7c1d")
	var a1 = wire.BlockHeader{
		Version: 32,
		PrevBlock: *testHash,
		MerkleRoot: *testHash,
		Timestamp: time.Unix(0x495fab29, 0),
		Bits: uint32(0x1d00ffff), // let just make up some stuff over here
		Nonce: uint32(0x1d00aaaa),// all invalid
	};
	b1 := a1
	a = append(a, &a1)
	b = append(b, &b1)

	if !moreWork(a,b,p) {
		log.Println("All's cool, this basic test runs")
	} else {
		log.Println(blockchain.CalcWork(a1.Bits))
	}
}
