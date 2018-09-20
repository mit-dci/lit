package coinparam

import (
	"time"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/wire"
)

// DummyUsdNetParams for DummyUsd fork defines the network parameters for the DummyUsd network.
var DummyUsdNetParams = Params{
	Name:          "dollar",
	NetMagicBytes: 0x44535564,
	DefaultPort:   "26999",
	DNSSeeds:      []string{},

	// Chain parameters
	GenesisBlock: &DummyUsdGenesisBlock,
	GenesisHash:  &DummyUsdGenesisHash,
	PoWFunction: func(b []byte, height int32) chainhash.Hash {
		return chainhash.DoubleHashH(b)
	},
	DiffCalcFunction: diffBitcoin,
	//	func(r io.ReadSeeker, height, startheight int32, p *Params) (uint32, error) {
	//		return diffBTC(r, height, startheight, p, false)
	//	},
	FeePerByte:               80,
	PowLimit:                 regressionPowLimit,
	PowLimitBits:             0x207fffff,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 150,
	TargetTimespan:           time.Hour * 24 * 14, // 14 days
	TargetTimePerBlock:       time.Minute * 10,    // 10 minutes
	RetargetAdjustmentFactor: 4,                   // 25% less, 400% more
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Minute * 20, // TargetTimePerBlock * 2
	GenerateSupported:        true,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: nil,

	// Enforce current block version once majority of the network has
	// upgraded.
	// 75% (750 / 1000)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 95% (950 / 1000)
	BlockEnforceNumRequired: 750,
	BlockRejectNumRequired:  950,
	BlockUpgradeNumToCheck:  1000,

	// Mempool parameters
	RelayNonStdTxs: true,

	// Address encoding magics
	PubKeyHashAddrID: 0x1e, // starts with D
	ScriptHashAddrID: 0x5a, // starts with d
	PrivateKeyID:     0x83, // starts with u
	Bech32Prefix:     "dusd",

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0xA5, 0xB3, 0xF4}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0xA5, 0xB7, 0x8F}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 262,
	TestCoin:   true,
}

var DummyUsdGenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xd8, 0x69, 0x52, 0xad, 0x56, 0xac, 0xda, 0xe2,
	0x32, 0x33, 0xa9, 0xb3, 0x24, 0x66, 0x27, 0x89,
	0xf7, 0x39, 0x60, 0xa3, 0x7a, 0x17, 0xae, 0xf7,
	0x69, 0x2e, 0xf0, 0x1b, 0x6d, 0x33, 0x4d, 0x19,
})

var DummyUsdGenesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x7e, 0xb6, 0x28, 0xda, 0xa0, 0xd2, 0xef, 0x48,
	0x1f, 0x45, 0x9f, 0x5b, 0x35, 0x47, 0x18, 0x41,
	0xb6, 0x69, 0x20, 0xdc, 0x12, 0xa3, 0xbf, 0x94,
	0x92, 0xca, 0xfe, 0xfc, 0x4e, 0xab, 0xb0, 0x72,
})

var DummyUsdGenesisCoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: []byte{
				0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04, 0x41,
				0x49, 0x20, 0x77, 0x6f, 0x6e, 0x64, 0x65, 0x72,
				0x20, 0x69, 0x66, 0x20, 0x70, 0x65, 0x6f, 0x70,
				0x6c, 0x65, 0x20, 0x77, 0x69, 0x6c, 0x6c, 0x20,
				0x6e, 0x6f, 0x74, 0x69, 0x63, 0x65, 0x20, 0x74,
				0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x6a,
				0x75, 0x73, 0x74, 0x20, 0x62, 0x69, 0x74, 0x63,
				0x6f, 0x69, 0x6e, 0x20, 0x77, 0x69, 0x74, 0x68,
				0x20, 0x61, 0x20, 0x66, 0x6c, 0x61, 0x76, 0x6f,
				0x72,
			},
			Sequence: 0xffffffff,
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value: 0x12a05f200,
			PkScript: []byte{
				0x41, 0x04, 0x67, 0x8a, 0xfd, 0xb0, 0xfe, 0x55, /* |A.g....U| */
				0x48, 0x27, 0x19, 0x67, 0xf1, 0xa6, 0x71, 0x30, /* |H'.g..q0| */
				0xb7, 0x10, 0x5c, 0xd6, 0xa8, 0x28, 0xe0, 0x39, /* |..\..(.9| */
				0x09, 0xa6, 0x79, 0x62, 0xe0, 0xea, 0x1f, 0x61, /* |..yb...a| */
				0xde, 0xb6, 0x49, 0xf6, 0xbc, 0x3f, 0x4c, 0xef, /* |..I..?L.| */
				0x38, 0xc4, 0xf3, 0x55, 0x04, 0xe5, 0x1e, 0xc1, /* |8..U....| */
				0x12, 0xde, 0x5c, 0x38, 0x4d, 0xf7, 0xba, 0x0b, /* |..\8M...| */
				0x8d, 0x57, 0x8a, 0x4c, 0x70, 0x2b, 0x6b, 0xf1, /* |.W.Lp+k.| */
				0x1d, 0x5f, 0xac, /* |._.| */
			},
		},
	},
	LockTime: 0,
}

// regTestGenesisBlock defines the genesis block of the block chain which serves
// as the public transaction ledger for the regression test network.
var DummyUsdGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{}, // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: DummyUsdGenesisMerkleRoot,
		Timestamp:  time.Unix(1537252543, 0),
		Bits:       0x207fffff,
		Nonce:      0,
	},
	Transactions: []*wire.MsgTx{&DummyUsdGenesisCoinbaseTx},
}
