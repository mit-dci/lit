package coinparam

import (
	"time"

	"github.com/adiabat/btcd/chaincfg/chainhash"
	"github.com/adiabat/btcd/wire"
)

// BC2NetParams are the parameters for the BC2 test network.
var BC2NetParams = Params{
	Name:          "bc2",
	NetMagicBytes: 0xcaa5afea,
	DefaultPort:   "8444",
	DNSSeeds:      []string{},

	// Chain parameters
	GenesisBlock:             &bc2GenesisBlock,
	GenesisHash:              &bc2GenesisHash,
	PoWFunction:              chainhash.DoubleHashH,
	DiffCalcFunction:         diffBTC,
	FeePerByte:               80,
	PowLimit:                 bc2NetPowLimit,
	PowLimitBits:             0x1d7fffff,
	CoinbaseMaturity:         10,
	SubsidyReductionInterval: 210000,
	TargetTimespan:           time.Hour * 1,   // 1 hour
	TargetTimePerBlock:       time.Minute * 1, // 1 minute
	RetargetAdjustmentFactor: 4,               // 25% less, 400% more
	ReduceMinDifficulty:      false,
	MinDiffReductionTime:     time.Minute * 20, // TargetTimePerBlock * 2
	GenerateSupported:        false,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{},

	// Enforce current block version once majority of the network has
	// upgraded.
	// 51% (51 / 100)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 75% (75 / 100)
	BlockEnforceNumRequired: 51,
	BlockRejectNumRequired:  75,
	BlockUpgradeNumToCheck:  100,

	// Mempool parameters
	RelayNonStdTxs: true,

	// Address encoding magics
	PubKeyHashAddrID: 0x19, // starts with B
	ScriptHashAddrID: 0x1c, // starts with ?
	Bech32Prefix:     "bc2",
	PrivateKeyID:     0xef, // starts with 9 7(uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 2,

	TestCoin: true,
}

// bc2GenesisHash is the hash of the first block in the block chain for the
// test network (version 3).
var bc2GenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x71, 0xed, 0xa3, 0xc2, 0xe3, 0x36, 0x73, 0x3d, 0x45, 0x03, 0x88, 0x90,
	0xd8, 0xae, 0x54, 0x11, 0x87, 0x92, 0x1c, 0x49, 0xb8, 0x7f, 0x41, 0xd6,
	0x99, 0xf6, 0xf3, 0xae, 0x0a, 0x00, 0x00, 0x00,
})

// bc2GenesisMerkleRoot is the same on bc2
var bc2GenesisMerkleRoot = genesisMerkleRoot

// bc2GenesisBlock has a different time stamp and difficulty
var bc2GenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: bc2GenesisMerkleRoot,     // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		Timestamp:  time.Unix(1483467305, 0), // later
		Bits:       0x1d7fffff,               // 486604799 [00000000ffff0000000000000000000000000000000000000000000000000000]
		Nonce:      0x334188d,                // 53745805
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}
