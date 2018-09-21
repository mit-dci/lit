package lnutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/btcutil/base58"
	"github.com/mit-dci/lit/crypto/fastsha256"
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

func LitFullKeyAdrEncode(in [33]byte) string {
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

// OldAddressFromPKH returns a base58 string from a 20 byte pubkey hash
func OldAddressFromPKH(pkHash [20]byte, netID byte) string {
	return base58.CheckEncode(pkHash[:], netID)
}

// ParseAdrString splits a string like
// "ln1yrvw48uc3atg8e2lzs43mh74m39vl785g4ehem@myhost.co:8191 into a separate
// pkh part and network part, adding the network part if needed
func ParseAdrString(adr string) (string, string) {
	id, host, port := ParseAdrStringWithPort(adr)
	if port == 0 {
		return id, ""
	}
	return id, fmt.Sprintf("%s:%d", host, port)
}

func ParseAdrStringWithPort(adr string) (string, string, uint32) {
	// Check if it's just a number - then it will be returned as the port
	if !strings.ContainsRune(adr, ':') && !strings.ContainsRune(adr, '@') {
		port, err := strconv.ParseUint(adr, 10, 32)
		if err == nil {
			return "", "127.0.0.1", uint32(port)
		}
	}

	// It's not a number, check if it starts with ln1,
	// if so, return the address and no host info (since it has no @)
	// Otherwise, assume it's a hostname and return empty adr and default port
	if !strings.ContainsRune(adr, ':') && !strings.ContainsRune(adr, '@') {
		if strings.HasPrefix(adr, "ln1") {
			return adr, "", 0
		} else {
			return "", adr, 2448
		}
	}

	lnAdr := ""
	hostSpec := "localhost:2448"

	// If it contains no @ but does contain a semicolon, expect this to be
	// an empty ln-address but a host with a port
	if strings.ContainsRune(adr, ':') && !strings.ContainsRune(adr, '@') {
		hostSpec = adr
	}

	// If it is formatted like adr@host, but without a semicolon, append
	// the default port.
	if !strings.ContainsRune(adr, ':') && strings.ContainsRune(adr, '@') {
		adr += ":2448"
	}

	if strings.ContainsRune(adr, '@') {
		idHost := strings.Split(adr, "@")
		lnAdr = idHost[0]
		hostSpec = idHost[1]
	}

	var host string
	var port uint32

	hostString := strings.Split(hostSpec, ":")
	host = hostString[0]
	if len(hostString) == 1 {
		// it is :, use default port
		port = 2448
	} else {
		port64, _ := strconv.ParseUint(hostString[1], 10, 32)
		port = uint32(port64)
	}
	if len(host) == 0 {
		host = "localhost"
	}
	return lnAdr, host, port
}
