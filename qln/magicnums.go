package qln

import "github.com/mit-dci/lit/btcutil/hdkeychain"

const (
	UseWallet             = 0 | hdkeychain.HardenedKeyStart
	UseChannelFund        = 20 | hdkeychain.HardenedKeyStart
	UseChannelRefund      = 30 | hdkeychain.HardenedKeyStart
	UseChannelWatchRefund = 31 | hdkeychain.HardenedKeyStart
	UseChannelHAKDBase    = 40 | hdkeychain.HardenedKeyStart
	UseChannelElkrem      = 8888 | hdkeychain.HardenedKeyStart

	// key derivation path for contract payout keys (the keys the contract pays
	// TO combined with the oracle's signature)
	UseContractPayoutBase = 50 | hdkeychain.HardenedKeyStart

	// key derivation path for contract spending keys (the keys the contract
	// pays FROM, and the funding TX pays TO)
	UseContractFundMultisig = 51 | hdkeychain.HardenedKeyStart

	// key derivation path for contract payout PKH (the hash the contract
	// pays TO)
	UseContractPayoutPKH = 52 | hdkeychain.HardenedKeyStart

	// key derivation path for HTLC pubkeys
	UseHTLCBase = 60 | hdkeychain.HardenedKeyStart

	// links Id and channel. replaces UseChannelFund
	UseIdKey = 111 | hdkeychain.HardenedKeyStart

	// high 3 bytes are in sequence, low 3 bytes are in time
	seqMask  = 0xff000000 // assert high byte
	timeMask = 0x21000000 // 1987 to 1988
)

var (
	BKTChannel = []byte("chn") // all channel data is in this bucket.
	BKTPeers   = []byte("pir") // all peer data is in this bucket.
	BKTPeerMap = []byte("pmp") // map of peer index to pubkey
	BKTChanMap = []byte("cmp") // map of channel index to outpoint
	BKTWatch   = []byte("wch") // txids & signatures for export to watchtowers
	BKTHTLCOPs = []byte("hlo") // htlc outpoints to watch

	KEYIdx      = []byte("idx")  // index for key derivation
	KEYhost     = []byte("hst")  // hostname where peer lives
	KEYnickname = []byte("nick") // nickname where peer lives

	KEYutxo    = []byte("utx") // serialized utxo for the channel
	KEYState   = []byte("now") // channel state
	KEYElkRecv = []byte("elk") // elkrem receiver
	KEYqclose  = []byte("cls") // channel close outpoint & height
)
