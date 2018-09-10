package coinparam

import (
	"time"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/wire"

	"golang.org/x/crypto/scrypt"
)

// LiteCoinTestNet4Params are the parameters for the litecoin test network 4.
var LiteCoinTestNet4Params = Params{
	Name:          "litetest4",
	NetMagicBytes: 0xf1c8d2fd,
	DefaultPort:   "19335",
	DNSSeeds: []string{
		"testnet-seed.litecointools.com",
		"seed-b.litecoin.loshan.co.uk",
		"dnsseed-testnet.thrasher.io",
	},

	// Chain parameters
	GenesisBlock: &bc2GenesisBlock, // no it's not
	GenesisHash:  &liteCoinTestNet4GenesisHash,
	PoWFunction: func(b []byte, height int32) chainhash.Hash {
		scryptBytes, _ := scrypt.Key(b, b, 1024, 1, 1, 32)
		asChainHash, _ := chainhash.NewHash(scryptBytes)
		return *asChainHash
	},
	DiffCalcFunction: diffBitcoin,
	StartHeader: [80]byte{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xd9, 0xce, 0xd4, 0xed, 0x11, 0x30, 0xf7, 0xb7, 0xfa, 0xad, 0x9b, 0xe2,
		0x53, 0x23, 0xff, 0xaf, 0xa3, 0x32, 0x32, 0xa1, 0x7c, 0x3e, 0xdf, 0x6c,
		0xfd, 0x97, 0xbe, 0xe6, 0xba, 0xfb, 0xdd, 0x97, 0xf6, 0x0b, 0xa1, 0x58,
		0xf0, 0xff, 0x0f, 0x1e, 0xe1, 0x79, 0x04, 0x00,
	},
	StartHeight:              48384,
	AssumeDiffBefore:         50401,
	FeePerByte:               800,
	PowLimit:                 liteCoinTestNet4PowLimit,
	PowLimitBits:             0x1e0fffff,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 840000,
	TargetTimespan:           time.Hour * 84,    // 84 hours
	TargetTimePerBlock:       time.Second * 150, // 150 seconds
	RetargetAdjustmentFactor: 4,                 // 25% less, 400% more
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Minute * 10, // ?? unknown
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
	PubKeyHashAddrID: 0x6f, // starts with m or n
	ScriptHashAddrID: 0xc4, // starts with 2
	Bech32Prefix:     "tltc",
	PrivateKeyID:     0xef, // starts with 9 7(uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 65537, // i dunno, 0x010001 ?
}

// LiteCoinTestNet4Params are the parameters for the litecoin test network 4.
var LiteRegNetParams = Params{
	Name:          "litereg",
	NetMagicBytes: 0xdab5bffa,
	DefaultPort:   "19444",
	DNSSeeds:      []string{},

	// Chain parameters
	GenesisBlock: &liteCoinRegTestGenesisBlock, // no it's not
	GenesisHash:  &liteCoinRegTestGenesisHash,
	PoWFunction: func(b []byte, height int32) chainhash.Hash {
		scryptBytes, _ := scrypt.Key(b, b, 1024, 1, 1, 32)
		asChainHash, _ := chainhash.NewHash(scryptBytes)
		return *asChainHash
	},
	DiffCalcFunction:         diffBitcoin,
	FeePerByte:               800,
	PowLimit:                 regressionPowLimit,
	PowLimitBits:             0x207fffff,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 150,
	TargetTimespan:           time.Hour * 84,    // 84 hours (3.5 days)
	TargetTimePerBlock:       time.Second * 150, // 150 seconds (2.5 min)
	RetargetAdjustmentFactor: 4,                 // 25% less, 400% more
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Minute * 10, // ?? unknown
	GenerateSupported:        true,

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
	PubKeyHashAddrID: 0x6f, // starts with m or n
	ScriptHashAddrID: 0xc4, // starts with 2
	Bech32Prefix:     "rltc",
	PrivateKeyID:     0xef, // starts with 9 7(uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 258, // i dunno
	TestCoin: true,
}

// liteCoinTestNet4GenesisHash is the first hash in litecoin testnet4
var liteCoinTestNet4GenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xa0, 0x29, 0x3e, 0x4e, 0xeb, 0x3d, 0xa6, 0xe6, 0xf5, 0x6f, 0x81, 0xed,
	0x59, 0x5f, 0x57, 0x88, 0x0d, 0x1a, 0x21, 0x56, 0x9e, 0x13, 0xee, 0xfd,
	0xd9, 0x51, 0x28, 0x4b, 0x5a, 0x62, 0x66, 0x49,
})

var liteCoinTestNet4MerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xd9, 0xce, 0xd4, 0xed, 0x11, 0x30, 0xf7, 0xb7, 0xfa, 0xad, 0x9b, 0xe2,
	0x53, 0x23, 0xff, 0xaf, 0xa3, 0x32, 0x32, 0xa1, 0x7c, 0x3e, 0xdf, 0x6c,
	0xfd, 0x97, 0xbe, 0xe6, 0xba, 0xfb, 0xdd, 0x97,
})

// liteCoinTestNet4GenesisBlock has is like completely its own thing
var liteCoinTestNet4GenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{}, // empty
		MerkleRoot: liteCoinTestNet4MerkleRoot,
		Timestamp:  time.Unix(1486949366, 0), // later
		Bits:       0x1e0ffff0,
		Nonce:      293345,
	},
	//	Transactions: []*wire.MsgTx{&genesisCoinbaseTx}, // this is wrong... will it break?
}

// ==================== LiteRegNet

// liteCoinRegTestGenesisHash is the first hash in litecoin regtest
var liteCoinRegTestGenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xf9, 0x16, 0xc4, 0x56, 0xfc, 0x51, 0xdf, 0x62,
	0x78, 0x85, 0xd7, 0xd6, 0x74, 0xed, 0x02, 0xdc,
	0x88, 0xa2, 0x25, 0xad, 0xb3, 0xf0, 0x2a, 0xd1,
	0x3e, 0xb4, 0x93, 0x8f, 0xf3, 0x27, 0x08, 0x53,
})

// is this the same...?
var liteCoinRegTestMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xd9, 0xce, 0xd4, 0xed, 0x11, 0x30, 0xf7, 0xb7, 0xfa, 0xad, 0x9b, 0xe2,
	0x53, 0x23, 0xff, 0xaf, 0xa3, 0x32, 0x32, 0xa1, 0x7c, 0x3e, 0xdf, 0x6c,
	0xfd, 0x97, 0xbe, 0xe6, 0xba, 0xfb, 0xdd, 0x97,
})

// liteCoinTestNet4GenesisBlock has is like completely its own thing
var liteCoinRegTestGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{}, // empty
		MerkleRoot: liteCoinRegTestMerkleRoot,
		Timestamp:  time.Unix(1296688602, 0), // later
		Bits:       0x207fffff,
		Nonce:      0,
	},
	//	Transactions: []*wire.MsgTx{&genesisCoinbaseTx}, // this is wrong... will it break?
}
