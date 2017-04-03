package lnutil

import (
	"bytes"

	"github.com/adiabat/bech32"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Lit addresses use the bech32 format, but sometimes omit the checksum!
// You don't really *need* the checksum with LN identities because you
// can't lose money by sending it to the wrong place

/* types:
256 bit full pubkey? (58 char)

less than 256-bit is the truncated double-sha256 of the 33 byte pubkey.

160-bit with checksum (38 char)
LN1eyq3javh983xwlfvamh54r56v9ca6ck70dfrqz

95-bit without checksum (19 char)
ln1eyq3javh983xwlfvamh

tor uses 80-bit but that seems like it's cutting it close; 95 bit gives
a 32K-fold improvement and should be safe for a while.  Internal functions
use the full pubkey when known, and send the 160-bit checksummed string around,
with the last UI layer making the truncation to 19 chars if users want it.

It's also half the length (after the ln1, which users may not have to type?) of
the full address.

Non-UI code will need to deal with 95-bit truncated hashes for matching, which
is a little annoying, but can be stored in 12 bytes (96-bit) with the last bit
being wildcard.  That should be ok to deal with.

*/

func LitAdrFromPubkey(in [33]byte) string {
	doubleSha := chainhash.DoubleHashB(in[:])
	return bech32.Encode("ln", doubleSha[:20])
}

// tries to match a 95-bit truncated pkh string with a pubkey
// returns false if anything goes wrong.

func MatchPubkeyTruncAdr(adr string, pk [33]byte) bool {
	truncSquashed, err := bech32.StringToSquashedBytes(adr[3:])
	if err != nil {
		return false
	}

	truncPKH, err := bech32.Bytes5to8(truncSquashed)
	if err != nil {
		return false
	}

	if len(truncPKH) != 12 {
		return false
	}

	// double-hash the full pubkey
	fullPKH := chainhash.DoubleHashB(pk[:])

	// truncate to first 12 bytes of that
	fullPKH = fullPKH[:12]
	//de-assert lowest bit
	fullPKH[11] = fullPKH[11] & 0xfe

	// compare and return
	return bytes.Equal(truncPKH, fullPKH)
}

func MatchPubkeyFullAdr(adr string, pk [33]byte) bool {

	hrp, pkh, err := bech32.Decode(adr)
	if err != nil {
		return false
	}

	if hrp != "ln" || len(pkh) != 20 {
		return false
	}

	// double-hash the full pubkey
	fullPKH := chainhash.DoubleHashB(pk[:])

	// truncate to first 20 bytes of that
	fullPKH = fullPKH[:20]

	// compare and return
	return bytes.Equal(pkh, fullPKH)
}
