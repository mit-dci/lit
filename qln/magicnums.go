package qln

import "github.com/btcsuite/btcutil/hdkeychain"

const (
	UseWallet             = 0 | hdkeychain.HardenedKeyStart
	UseChannelFund        = 20 | hdkeychain.HardenedKeyStart
	UseChannelRefund      = 30 | hdkeychain.HardenedKeyStart
	UseChannelWatchRefund = 31 | hdkeychain.HardenedKeyStart
	UseChannelHAKDBase    = 40 | hdkeychain.HardenedKeyStart
	UseChannelElkrem      = 8888 | hdkeychain.HardenedKeyStart
	// links Id and channel. replaces UseChannelFund

	UseIdKey = 111 | hdkeychain.HardenedKeyStart

	// high 3 bytes are in sequence, low 3 bytes are in time
	seqMask  = 0xff000000 // assert high byte
	timeMask = 0x21000000 // 1987 to 1988
)

var (
	BKTPeers   = []byte("pir") // all peer data is in this bucket.
	BKTMap     = []byte("map") // map of peer index to pubkey
	BKTWatch   = []byte("wch") // txids & signatures for export to watchtowers
	KEYIdx     = []byte("idx") // index for key derivation
	KEYutxo    = []byte("utx") // serialized utxo for the channel
	KEYUnsig   = []byte("usg") // unsigned fund tx
	KEYCladr   = []byte("cdr") // coop close address (Don't make fun of my lisp)
	KEYState   = []byte("ima") // channel state
	KEYElkRecv = []byte("elk") // elkrem receiver
	KEYqclose  = []byte("qcl") // channel close outpoint & height
)
