package coinparam

import (
	"time"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/wire"

	"github.com/bitgoin/lyra2rev2"
	"github.com/vertcoin/lyra2re"
	"golang.org/x/crypto/scrypt"
)

var VertcoinTestNetParams = Params{
	Name:          "vtctest",
	NetMagicBytes: 0x74726576,
	DefaultPort:   "15889",
	DNSSeeds: []string{
		"fr1.vtconline.org",
	},

	// Chain parameters
	DiffCalcFunction: diffVTCtest,
	MinHeaders:       4032,
	FeePerByte:       100,
	GenesisBlock:     &VertcoinTestnetGenesisBlock,
	GenesisHash:      &VertcoinTestnetGenesisHash,
	PowLimit:         liteCoinTestNet4PowLimit,
	PoWFunction: func(b []byte, height int32) chainhash.Hash {
		lyraBytes, _ := lyra2rev2.Sum(b)
		asChainHash, _ := chainhash.NewHash(lyraBytes)
		return *asChainHash
	},
	PowLimitBits:             0x1e0fffff,
	CoinbaseMaturity:         120,
	SubsidyReductionInterval: 840000,
	TargetTimespan:           time.Second * 302400, // 3.5 weeks
	TargetTimePerBlock:       time.Second * 150,    // 150 seconds
	RetargetAdjustmentFactor: 4,                    // 25% less, 400% more
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Second * 150 * 2, // ?? unknown
	GenerateSupported:        false,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{},

	BlockEnforceNumRequired: 26,
	BlockRejectNumRequired:  49,
	BlockUpgradeNumToCheck:  50,

	// Mempool parameters
	RelayNonStdTxs: true,

	// Address encoding magics
	PubKeyHashAddrID: 0x4a, // starts with X or W
	ScriptHashAddrID: 0xc4,
	Bech32Prefix:     "tvtc",
	PrivateKeyID:     0xef,

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 65536,
	TestCoin:   true,
}

var VertcoinRegTestParams = Params{
	Name:          "vtcreg",
	NetMagicBytes: 0xdab5bffa,
	DefaultPort:   "18444",
	DNSSeeds:      []string{},

	// Chain parameters
	DiffCalcFunction: diffVTCregtest,
	MinHeaders:       4032,
	FeePerByte:       100,
	GenesisBlock:     &VertcoinRegTestnetGenesisBlock,
	GenesisHash:      &VertcoinRegTestnetGenesisHash,
	PowLimit:         regressionPowLimit,
	PoWFunction: func(b []byte, height int32) chainhash.Hash {
		var hashBytes []byte

		if height >= 347000 {
			hashBytes, _ = lyra2rev2.Sum(b)
		} else if height >= 208301 {
			hashBytes, _ = lyra2re.Sum(b)
		} else {
			hashBytes, _ = scrypt.Key(b, b, 2048, 1, 1, 32)
		}

		asChainHash, _ := chainhash.NewHash(hashBytes)
		return *asChainHash
	},
	PowLimitBits:             0x207fffff,
	CoinbaseMaturity:         120,
	SubsidyReductionInterval: 150,
	TargetTimespan:           time.Second * 302400, // 3.5 weeks
	TargetTimePerBlock:       time.Second * 150,    // 150 seconds
	RetargetAdjustmentFactor: 4,                    // 25% less, 400% more
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Second * 150 * 2, // ?? unknown
	GenerateSupported:        false,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{},

	BlockEnforceNumRequired: 26,
	BlockRejectNumRequired:  49,
	BlockUpgradeNumToCheck:  50,

	// Mempool parameters
	RelayNonStdTxs: true,

	// Address encoding magics
	PubKeyHashAddrID: 0x6f,
	ScriptHashAddrID: 0xc4,
	Bech32Prefix:     "rvtc",
	PrivateKeyID:     0xef,

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 261,
	TestCoin:   true,
}

var VertcoinParams = Params{
	Name:          "vtc",
	NetMagicBytes: 0xdab5bffa,
	DefaultPort:   "5889",
	DNSSeeds: []string{
		"fr1.vtconline.org",
		"uk1.vtconline.org",
		"useast1.vtconline.org",
		"vtc.alwayshashing.com",
		"crypto.office-on-the.net",
		"p2pool.kosmoplovci.org",
	},

	// Chain parameters
	StartHeader: [80]byte{
		0x02, 0x00, 0x00, 0x00, 0x36, 0xdc, 0x16, 0xc7, 0x71, 0x63,
		0x1c, 0x52, 0xa4, 0x3d, 0xb7, 0xb0, 0xa9, 0x86, 0x95, 0x95,
		0xed, 0x7d, 0xc1, 0x68, 0xe7, 0x2e, 0xaf, 0x0f, 0x55, 0x08,
		0x02, 0x98, 0x9f, 0x5c, 0x7b, 0xe4, 0x37, 0xa6, 0x90, 0x76,
		0x66, 0xa7, 0xba, 0x55, 0x75, 0xd8, 0x8a, 0xc5, 0x14, 0x01,
		0x86, 0x11, 0x8e, 0x34, 0xe2, 0x4a, 0x04, 0x7b, 0x9d, 0x6e,
		0x96, 0x41, 0xbb, 0x29, 0xe2, 0x04, 0xcb, 0x49, 0x3c, 0x53,
		0x08, 0x58, 0x3f, 0xf4, 0x4d, 0x1b, 0x42, 0x22, 0x6e, 0x8a,
	},
	StartHeight:      598752,
	AssumeDiffBefore: 602784,
	DiffCalcFunction: diffVTC,
	MinHeaders:       4032,
	FeePerByte:       100,
	GenesisBlock:     &VertcoinGenesisBlock,
	GenesisHash:      &VertcoinGenesisHash,
	PowLimit:         liteCoinTestNet4PowLimit,
	PoWFunction: func(b []byte, height int32) chainhash.Hash {
		var hashBytes []byte

		if height >= 347000 {
			hashBytes, _ = lyra2rev2.Sum(b)
		} else if height >= 208301 {
			hashBytes, _ = lyra2re.Sum(b)
		} else {
			hashBytes, _ = scrypt.Key(b, b, 2048, 1, 1, 32)
		}

		asChainHash, _ := chainhash.NewHash(hashBytes)
		return *asChainHash
	},
	PowLimitBits:             0x1e0fffff,
	CoinbaseMaturity:         120,
	SubsidyReductionInterval: 840000,
	TargetTimespan:           time.Second * 302400, // 3.5 weeks
	TargetTimePerBlock:       time.Second * 150,    // 150 seconds
	RetargetAdjustmentFactor: 4,                    // 25% less, 400% more
	ReduceMinDifficulty:      false,
	MinDiffReductionTime:     time.Second * 150 * 2, // ?? unknown
	GenerateSupported:        false,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{
		{0, newHashFromStr("4d96a915f49d40b1e5c2844d1ee2dccb90013a990ccea12c492d22110489f0c4")},
		{24200, newHashFromStr("d7ed819858011474c8b0cae4ad0b9bdbb745becc4c386bc22d1220cc5a4d1787")},
		{65000, newHashFromStr("9e673a69c35a423f736ab66f9a195d7c42f979847a729c0f3cef2c0b8b9d0289")},
		{84065, newHashFromStr("a904170a5a98109b2909379d9bc03ef97a6b44d5dafbc9084b8699b0cba5aa98")},
		{228023, newHashFromStr("15c94667a9e941359d2ee6527e2876db1b5e7510a5ded3885ca02e7e0f516b51")},
		{346992, newHashFromStr("f1714fa4c7990f4b3d472eb22132891ccd3c7ad7208e2d1ab15bde68854fb0ee")},
		{347269, newHashFromStr("fa1e592b7ea2aa97c5f20ccd7c40f3aaaeb31d1232c978847a79f28f83b6c22a")},
		{430000, newHashFromStr("2f5703cf7b6f956b84fd49948cbf49dc164cfcb5a7b55903b1c4f53bc7851611")},
		{516999, newHashFromStr("572ed47da461743bcae526542053e7bc532de299345e4f51d77786f2870b7b28")},
		{627610, newHashFromStr("6000a787f2d8bb77d4f491a423241a4cc8439d862ca6cec6851aba4c79ccfedc")},
	},

	BlockEnforceNumRequired: 1512,
	BlockRejectNumRequired:  1915,
	BlockUpgradeNumToCheck:  2016,

	// Mempool parameters
	RelayNonStdTxs: true,

	// Address encoding magics
	PubKeyHashAddrID: 0x47, // starts with V
	ScriptHashAddrID: 0x05, // starts with 3
	Bech32Prefix:     "vtc",
	PrivateKeyID:     0x80,

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 28,
}

// ==================== VertcoinTestnet

// VertcoinTestNetGenesisHash
var VertcoinTestnetGenesisHash = chainhash.Hash([chainhash.HashSize]byte{
	0xc9, 0xd2, 0x7a, 0x49, 0x47, 0x27, 0x2e, 0xe3, 0xc2,
	0xe8, 0x1a, 0x74, 0xb6, 0x79, 0xac, 0xec, 0x5d, 0x85,
	0xa4, 0x6a, 0x97, 0x16, 0x79, 0xf0, 0xc8, 0x64, 0x7a,
	0xeb, 0x4f, 0xf2, 0xe8, 0xce,
})

var VertcoinTestnetMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{
	0xe7, 0x23, 0x01, 0xfc, 0x49, 0x32, 0x3e, 0xe1, 0x51,
	0xcf, 0x10, 0x48, 0x23, 0x0f, 0x03, 0x2c, 0xa5, 0x89,
	0x75, 0x3b, 0xa7, 0x08, 0x62, 0x22, 0xa5, 0xc0, 0x23,
	0xe3, 0xa0, 0x8c, 0xf3, 0x4a,
})

var VertcoinTestnetGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{}, // empty
		MerkleRoot: VertcoinTestnetMerkleRoot,
		Timestamp:  time.Unix(1481291250, 0), // later
		Bits:       0x1e0ffff0,
		Nonce:      915027,
	},
}

// ==================== Vertcoin

// VertcoinNetGenesisHash
var VertcoinGenesisHash = chainhash.Hash([chainhash.HashSize]byte{
	0xc4, 0xf0, 0x89, 0x04, 0x11, 0x22, 0x2d, 0x49, 0x2c, 0xa1,
	0xce, 0x0c, 0x99, 0x3a, 0x01, 0x90, 0xcb, 0xdc, 0xe2, 0x1e,
	0x4d, 0x84, 0xc2, 0xe5, 0xb1, 0x40, 0x9d, 0xf4, 0x15, 0xa9,
	0x96, 0x4d,
})

var VertcoinMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{
	0xe7, 0x23, 0x01, 0xfc, 0x49, 0x32, 0x3e, 0xe1, 0x51, 0xcf, 0x10, 0x48, 0x23, 0x0f,
	0x03, 0x2c, 0xa5, 0x89, 0x75, 0x3b, 0xa7, 0x08, 0x62, 0x22, 0xa5, 0xc0, 0x23, 0xe3,
	0xa0, 0x8c, 0xf3, 0x4a,
})

var VertcoinGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{}, // empty
		MerkleRoot: VertcoinMerkleRoot,
		Timestamp:  time.Unix(1389311371, 0), // later
		Bits:       0x1e0ffff0,
		Nonce:      5749262,
	},
}

// ==================== VertcoinRegTestnet

//  VertcoinRegTestnetGenesisHash
var VertcoinRegTestnetGenesisHash = chainhash.Hash([chainhash.HashSize]byte{
	0xce, 0x85, 0x4a, 0xdc, 0x33, 0xe8, 0x7c, 0xc1, 0x6f,
	0xbc, 0x32, 0x19, 0x1a, 0x7b, 0x02, 0x17, 0x73, 0xc9,
	0x06, 0x72, 0x86, 0x66, 0x0d, 0x65, 0xd1, 0xbb, 0xeb,
	0x47, 0xb0, 0xc0, 0x99, 0x23,
})

var VertcoinRegTestnetMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{
	0xe7, 0x23, 0x01, 0xfc, 0x49, 0x32, 0x3e, 0xe1, 0x51,
	0xcf, 0x10, 0x48, 0x23, 0x0f, 0x03, 0x2c, 0xa5, 0x89,
	0x75, 0x3b, 0xa7, 0x08, 0x62, 0x22, 0xa5, 0xc0, 0x23,
	0xe3, 0xa0, 0x8c, 0xf3, 0x4a,
})

var VertcoinRegTestnetGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{}, // empty
		MerkleRoot: VertcoinRegTestnetMerkleRoot,
		Timestamp:  time.Unix(1296688602, 0), // later
		Bits:       0x207fffff,
		Nonce:      2,
	},
}
