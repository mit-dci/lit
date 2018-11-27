package lncore

import (
	"fmt"
	"regexp"
	"strings"
	"strconv"
	"encoding/hex"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/crypto/fastsha256"
)

/*
 * Ok so there's a lot of stuff going on here.  Lit natively supports addresses
 * that are just the pkh of the pubkey encoded in bech32.  And that's pretty
 * cool, you look up the IP+port in a tracker (or DHT) and connected with
 * Noise_XX, then verify that the pubkey they provide you matches the pkh you
 * already have.  Also you can provide IP+port if you want.
 *
 * But in mainstream BOLT nodes they use the full pubkey hex-encoded.  They have
 * a DHT where you look up the IP+port (if not provided) and connect with
 * Noise_XK, which assumes you already have the pubkey (since you do) and that's
 * all fine.
 *
 * This tries to unify all of them with formats like this under 1 parser:
 *
 *      <bech32 pkh>[@<ip>[:<port>]
 * or
 *
 *      <hex of bech32 pubkey>[@<ip>[:<port>]
 *
 * or fully
 *
 *      <bech32 pkh>:<pubkey>[@<ip>:<port>]
 */

// LnAddr is just a bech32-encoded pubkey.
// TODO Move this to another package so it's more obviously not *just* IO-related.
type LnAddr string

// ToString returns the LnAddr as a string.  Right now it just unwraps it but it
// might do something more eventually.
func (lnaddr LnAddr) ToString() string {
	return string(lnaddr)
}

// LnDefaultPort is from BOLT1.
const LnDefaultPort = 9735

// PkhBech32Prefix if for addresses lit already uses.
const PkhBech32Prefix = "ln"

// PubkeyBech32Prefix is for encoding the full pubkey in bech32.
const PubkeyBech32Prefix = "lnpk"

// LnAddressData is all of the data that can be encoded in a parsed address.
type LnAddressData struct {
	// Pkh is the pubkey hash as bech32.
	Pkh    *string
	
	// Pubkey is the pubkey, as bytes, decoded from source format.
	Pubkey []byte

	// IPAddr is the IP address, if present.
	IPAddr *string

	// Port should always be defined, default is LnDefaultPort.
	Port   uint16
}

// HasFullAddress returns whether or not we need to do a tracker/DHT lookup in
// order to resolve the connection address for this address data.  Basically, it
// checks if there's an IP address defined.
func (d *LnAddressData) HasFullAddress() bool {
	return d.IPAddr != nil
}

// AddrFmtFull is <bech32 pkh>:<hex pk>@<ip>:<port>
const AddrFmtFull = "full"

// AddrFmtFullBech32Pk is <bech32 pkh>:<bech32 pk>@<ip>:<port>
const AddrFmtFullBech32Pk = "full_bech32_pk"

// AddrFmtFullNoPort is <bech32 pkh>:<hex pk>@<ip>
const AddrFmtFullNoPort = "full_no_port"

// AddrFmtFullBech32PkNoPort is <bech32 pkh>:<bech32 pk>@<ip>
const AddrFmtFullBech32PkNoPort = "full_bech32_pk_no_port"

// AddrFmtBech32 is just <bech32 pkh>
const AddrFmtBech32 = "bech32_pkh"

// AddrFmtLit is just AddrFmtBech32
const AddrFmtLit = "lit"

// AddrFmtPubkey is <hex pubkey>
const AddrFmtPubkey = "hex_pk"

// AddrFmtBech32Pubkey is <bech32 pk>
const AddrFmtBech32Pubkey = "bech32_pk"

// AddrFmtPubkeyIP is <hex pk>@<ip>
const AddrFmtPubkeyIP = "hex_pk_ip"

// AddrFmtPubkeyIPPort is <hex pk>@<ip>:<port>
const AddrFmtPubkeyIPPort = "hex_pk_ip_port"

// AddrFmtLitFull is <bech32 pkh>@<ip>:<port>
const AddrFmtLitFull = "lit_full"

// AddrFmtLitIP is <bech32 pkh>@<ip>
const AddrFmtLitIP = "lit_ip"

// DumpAddressFormats returns the addresses in all of the formats that fully represent it.
func DumpAddressFormats(data *LnAddressData) (map[string]string, error) {

	ret := make(map[string]string)

	if data.Port != LnDefaultPort && data.IPAddr == nil {
		return ret, fmt.Errorf("nondefault port specified but IP not specified")
	}

 	// Full.
	if data.Pkh != nil && data.Pubkey != nil && data.IPAddr != nil {
		ret[AddrFmtFull] = fmt.Sprintf("%s:%X@%s:%d", *data.Pkh, data.Pubkey, *data.IPAddr, data.Port)
		ret[AddrFmtFullBech32Pk] = fmt.Sprintf("%s:%s@%s:%d", *data.Pkh, bech32.Encode(PubkeyBech32Prefix, data.Pubkey), *data.IPAddr, data.Port)
	}
	
	// Minimal Lit.
	if data.Pkh != nil {
		ret[AddrFmtLit] = *data.Pkh
	}

	// Long Lit.
	if data.Pkh != nil && data.IPAddr != nil {
		if data.Port == LnDefaultPort {
			ret[AddrFmtLitIP] = fmt.Sprintf("%s@%s", *data.Pkh, *data.IPAddr)
		}
		ret[AddrFmtLitFull] = fmt.Sprintf("%s@%s:%d", *data.Pkh, *data.IPAddr, data.Port)
	}

	// Hex pubkey stuff
	if data.Pubkey != nil {
		ret[AddrFmtPubkey] = fmt.Sprintf("%X", data.Pubkey)
		if data.IPAddr != nil {
			if data.Port == LnDefaultPort {
				ret[AddrFmtPubkeyIP] = fmt.Sprintf("%X@%s", data.Pubkey, *data.IPAddr)
			}
			ret[AddrFmtPubkeyIPPort] = fmt.Sprintf("%X@%s:%d", data.Pubkey, *data.IPAddr, data.Port)
		}
	}

	// TODO more

	return ret, nil

}

// ParseLnAddrData takes a LnAddr and parses the internal data.  Assumes it's
// not been force-casted as full checks are done in the ParseLnAddr function.
func ParseLnAddrData(addr LnAddr) (*LnAddressData, error) {

	addrdata := new(LnAddressData)
	
	parts := strings.SplitN(string(addr), "@", 2) // using cast here because reasons

	// First process the identify information.
	pkdata := strings.SplitN(parts[0], ":", 2)
	if len(pkdata) == 2 {

		// The pkh is already in the right format for internal storage.
		addrdata.Pkh = &pkdata[0]		
		
		// Now figure out how to parse the full pubkey info.
		if strings.HasPrefix(pkdata[1], PubkeyBech32Prefix + "1") {
			_, data, err := bech32.Decode(pkdata[1])
			if err != nil {
				return nil, err
			}
			addrdata.Pubkey = data
		} else {
			data, err := hex.DecodeString(pkdata[1])
			if err != nil {
				return nil, err
			}
			addrdata.Pubkey = data
		}
		
	} else {

		// If there's only 1 part then there's 3 mutually exclusive options.
		if strings.HasPrefix(pkdata[0], PkhBech32Prefix + "1") {
			addrdata.Pkh = &pkdata[0]
		} else if strings.HasPrefix(pkdata[0], PubkeyBech32Prefix + "1") {
			_, data, err := bech32.Decode(pkdata[0])
			if err != nil {
				return nil, err
			}
			addrdata.Pubkey = data
		} else {
			data, err := hex.DecodeString(pkdata[0])
			if err != nil {
				return nil, err
			}
			addrdata.Pubkey = data
		}

		// Now we figure out what the pkh should be if the full pubkey is specified.
		if addrdata.Pkh == nil && addrdata.Pubkey != nil {
			pkh := ConvertPubkeyToBech32Pkh(addrdata.Pubkey)
			addrdata.Pkh = &pkh
		}
		
	}

	// Now parse the location information if it's there.
	if len(parts) > 1 {
		ipdata := strings.SplitN(parts[1], ":", 2)

		// The IP address is already in the right format, probably.
		addrdata.IPAddr = &ipdata[0]

		// Parse the port if it's there.
		if len(ipdata) > 1 {
			port, err := strconv.Atoi(ipdata[1])
			if err != nil {
				return nil, err
			}
			if port > 65535 {
				return nil, fmt.Errorf("port number %d not in range", port)
			}
			addrdata.Port = uint16(port)
		} else {
			addrdata.Port = LnDefaultPort
		}
		
	}
	
	return addrdata, nil // TODO
}

/*
+0      q 	p 	z 	r 	y 	9 	x 	8
+8 	g 	f 	2 	t 	v 	d 	w 	0
+16 	s 	3 	j 	n 	5 	4 	k 	h
+24 	c 	e 	6 	m 	u 	a 	7 	l
*/

const bech32bodyregex = `[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{4,}`
const ipaddrregex = `[0-9]{1,3}(\.[0-9]{1,3}){3}`
const pkhregex = `ln1` + bech32bodyregex
const pubkeyregex = `(lnpk1` + bech32bodyregex + `|` + `[0-9a-fA-F]{64})`
const fulladdrregex = `(` + pkhregex + `|` + pubkeyregex + `|` + pkhregex + `:` + pubkeyregex + `)(@` + ipaddrregex + `(:[0-9]{1,5})?)?`

// ParseLnAddr will verify that the string passed is a valid LN address of any format.
func ParseLnAddr(m string) (LnAddr, error) {

	r := regexp.MustCompile(fulladdrregex)

	if !r.Match([]byte(m)) {
		return "", fmt.Errorf("address doesn't match overall format")
	}

	parts := strings.SplitN(m, "@", 2)
	pkdata := strings.SplitN(parts[0], ":", 2)
	if len(pkdata) == 2 {
		// This means there's both a pkh and a pubkey.
		
		pkhprefix, _, pkherr := bech32.Decode(pkdata[0])
		if pkherr != nil {
			return "", fmt.Errorf("invalid pkh bech32")
		}
		if pkhprefix != PkhBech32Prefix {
			return "", fmt.Errorf("pkh bech32 prefix incorrect")
		}
		
		pkprefix, _, pkerr := bech32.Decode(pkdata[1])
		if pkerr != nil {
			// Maybe it's hex.
			_, err := hex.DecodeString(pkdata[1])
			if err != nil {
				return "", fmt.Errorf("pubkey not valid bech32 or hex ('%s' and '%s')", pkerr.Error(), err.Error())
			}
		}
		if pkprefix != PubkeyBech32Prefix {
			return "", fmt.Errorf("pubkey bech32 prefix incorrect")
		}
		
	} else {
		// This means there's *either* a pubkey or pkh, in some format.

		prefix, _, err := bech32.Decode(pkdata[0])
		if err != nil {
			// Maybe it's hex.
			_, err2 := hex.DecodeString(pkdata[0])
			if err2 != nil {
				return "", fmt.Errorf("pubkey not valid bech32 or hex ('%s' and '%s')", err.Error(), err2.Error())
			}
		}

		if err == nil && prefix != PkhBech32Prefix && prefix != PubkeyBech32Prefix {
			return "", fmt.Errorf("pubkey (or phk) bech32 prefix incorrect")
		}
	}

	// Should we make sure that the pubkey and pkh match if both are present?
	
	return LnAddr(m), nil // should be the only place we cast to this type

}

// ConvertPubkeyToBech32Pkh converts the pubkey into the bech32 representation
// of the PKH.
func ConvertPubkeyToBech32Pkh(pubkey []byte) string {
	hash := fastsha256.Sum256(pubkey)
	enc := bech32.Encode("ln", hash[:20])
	return enc
}
