package litrpc

import (
	log "github.com/mit-dci/lit/logs"
	"strings"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/btcutil/base58"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
)

/*
Multi-wallet address functions that maybe should be in another package
(btcutil? chaincfg?) but are here for now since modifying btcd is hard.

*/

// AdrStringToOutscript converts an address string into an output script byte slice
// note that this ignores the prefix! Be careful not to mix networks.
// currently only works for testnet legacy addresses
func AdrStringToOutscript(adr string) ([]byte, error) {
	var err error
	var outScript []byte

	// use HRP to determine network / wallet to use
	outScript, err = bech32.SegWitAddressDecode(adr)
	if err != nil { // valid bech32 string
		// try for base58 address
		// btcutil addresses don't really work as they won't tell you the
		// network; you have to tell THEM the network, which defeats the point
		// of having an address.  default to testnet only here

		// could work on adding more old-style addresses; for now use new bech32
		// addresses for multi-wallet / segwit sends.

		// ignore netID here
		decoded, _, err := base58.CheckDecode(adr)
		if err != nil {
			return nil, err
		}

		outScript, err = lnutil.PayToPubKeyHashScript(decoded)
		if err != nil {
			return nil, err
		}
	}
	return outScript, nil
}

// Default to testnet for unknown / bad addrs.
func CoinTypeFromAdr(adr string) uint32 {
	ct, err := CoinTypeFromBechAdr(adr)
	if err == nil {
		return ct
	}
	log.Errorf("cointype from bech32 error: %s\n", err.Error())

	if len(adr) < 5 {
		// well that's not even an address
		return 12345
	}
	if strings.HasPrefix(adr, "m") || strings.HasPrefix(adr, "n") {
		// guess testnet; could be regtest
		return 1
	}
	if strings.HasPrefix(adr, "V") {
		return 28
	}
	if strings.HasPrefix(adr, "X") || strings.HasPrefix(adr, "W") {
		return 65536
	}
	// add other prefixes here...
	return 1
}

// Gives the cointype from an address string (if known)
func CoinTypeFromBechAdr(adr string) (uint32, error) {
	hrp, err := bech32.GetHRP(adr)
	if err != nil {
		return 0, err
	}
	return coinparam.PrefixToCoinType(hrp)
}
