package consts

// commonly used constants that can be used anywhere, without ambiguity
const (
	ChannelTimeout         = 60               // channel operation timeout in seconds before failing the channel
	ChannelAdvTimeout      = 60               // max allowed seconds between channel advertisements before deletion
	MaxChanCapacity        = int64(100000000) // Maximum Channel Capacity (at 1 coin now)
	MinChanCapacity        = int64(1000000)   // Minimum Channel Capacity
	SafeFee                = int64(50000)     // safeFee while initializing a chan
	MaxKeys                = uint32(1 << 20)  // max number of keys lit can store (could be infinite, still)
	MaxTxCount             = int64(10000)     // max tx's associated with an address
	DustCutoff             = int64(20000)     // below this, give to miners
	MinOutput              = 100000           // minOutput is the minimum output amt, post fee. This (plus fees) is also the minimum channel balance
	MinSendAmt             = 10000            // minimum amount that can be sent through a chan
	MaxTxLen               = 100000           // maximum number of tx's that can be ingested at once
	JusticeFee             = int64(5000)      // if someone spends the com tx, pay a high justice fee
	BitcoinRegtestBHeight  = 120              // height at which you want regtest sync to start
	BitcoinTestnet3BHeight = 1256000          // height at which testnet3 sync starts
	VertcoinTestnetBHeight = 25000            // height at which vertcoin testnet sync starts
	DualFundFee            = 50
	MaxSendAmt             = 1 << 30 // maximum amount that can be sent through a chan
	MaxKeyLimit            = 1 << 30 // maximum number of keys that lit can store
	JusticeTxBump          = 100     // fix justicetx fee 10 times the normal fee
	QcStateFee             = 10      // fixqcstatefee
	DefaultLockTime        = 500     //default lock time
	DlcSettlementTxFee     = 1000
	MaxOraclesNumber	   = 8
)
