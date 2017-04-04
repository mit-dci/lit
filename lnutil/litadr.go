package lnutil

import (
	"fmt"

	"github.com/adiabat/bech32"
	"github.com/btcsuite/fastsha256"
)

// Lit addresses use the bech32 format, but sometimes omit the checksum!
// You don't really *need* the checksum with LN identities because you
// can't lose money by sending it to the wrong place

/* types:
256 bit full pubkey? (58 char)

less than 256-bit is the truncated sha256 of the 33 byte pubkey.
(probably the easiest way is to drop the first 'q' character

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

func LitFullAdrEncode(in [33]byte) string {
	withq := bech32.Encode("ln", in[:])
	// get rid of the q after the 1.  Pubkeys are always 0x02 or 0x03,
	// so the first 5 bits are always 0.
	withq = withq[:3] + withq[4:]
	return withq
}

func LitFullAdrDecode(in string) ([33]byte, error) {
	var pub [33]byte
	if len(in) != 61 {
		return pub, fmt.Errorf("Invalid length, got %d expect 33", len(in))
	}
	// add the q back in so it decodes
	in = in[:3] + "q" + in[3:]
	hrp, data, err := bech32.Decode(in)
	if err != nil {
		return pub, err
	}
	if hrp != "ln" {
		return pub, fmt.Errorf("Not a ln address, prefix %s", hrp)
	}
	copy(pub[:], data)
	return pub, nil
}

func LitAdrFromPubkey(in [33]byte) string {
	doubleSha := fastsha256.Sum256(in[:])
	return bech32.Encode("ln", doubleSha[:20])
}

// LitAdrOK make sure the address is OK.  Either it has a valid checksum, or
// it's shortened and doesn't.
func LitAdrOK(adr string) bool {
	hrp, _, err := bech32.Decode(adr)
	if hrp != "ln" {
		return false
	}
	if err == nil || len(adr) == 22 {
		return true
	}
	return false
}

// LitAdrBytes takes a lit address string and returns either 20 or 12 bytes.
// Or an error.
func LitAdrBytes(adr string) ([]byte, error) {
	if !LitAdrOK(adr) {
		return nil, fmt.Errorf("invalid ln address %s", adr)
	}

	_, pkh, err := bech32.Decode(adr)
	if err == nil {
		return pkh, nil
	}
	// add a q for padding
	adr = adr + "q"

	truncSquashed, err := bech32.StringToSquashedBytes(adr[3:])
	if err != nil {
		return nil, err
	}

	truncPKH, err := bech32.Bytes5to8(truncSquashed)
	if err != nil {
		return nil, err
	}
	return truncPKH, nil
}
