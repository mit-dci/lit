package litrpc

import (
	"fmt"
	"strings"

	"github.com/adiabat/bech32"
	"github.com/adiabat/btcd/chaincfg"
	"github.com/adiabat/btcd/txscript"
	"github.com/adiabat/btcutil"
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
	_, adrData, err := bech32.Decode(adr)
	if err == nil { // valid bech32 string
		if len(adrData) != 20 {
			return nil, fmt.Errorf("Address %s has %d byte payload, expect 20",
				adr, len(adrData))
		}
		var adr160 [20]byte
		copy(adr160[:], adrData)

		outScript = lnutil.DirectWPKHScriptFromPKH(adr160)
	} else {
		// try for base58 address
		// btcutil addresses don't really work as they won't tell you the
		// network; you have to tell THEM the network, which defeats the point
		// of having an address.  default to testnet only here

		// could work on adding more old-style addresses; for now use new bech32
		// addresses for multi-wallet / segwit sends.
		adr, err := btcutil.DecodeAddress(adr, &chaincfg.TestNet3Params)
		if err != nil {
			return nil, err
		}

		outScript, err = txscript.PayToAddrScript(adr)
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

	if len(adr) < 5 {
		// well that's not even an address
		return 12345
	}
	if strings.HasPrefix(adr, "m") || strings.HasPrefix(adr, "n") {
		// guess testnet; could be regtest
		return 1
	}
	// add other prefixes here...
	return 1
}

// Gives the cointype from an address string (if known)
func CoinTypeFromBechAdr(adr string) (uint32, error) {
	hrp, _, err := bech32.Decode(adr)
	if err != nil {
		return 0, err
	}
	return chaincfg.PrefixToCoinType(hrp)
}
