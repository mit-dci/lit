package lncore;

import (
	"fmt"
	
	"github.com/mit-dci/lit/bech32"
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

// LnDefaultPort is from BOLT1.
const LnDefaultPort = 9735

// PkhBech32Prefix if for addresses lit already uses.
const PkhBech32Prefix = "ln"

// PubkeyBech32Prefix is for encoding the full pubkey in bech32.
const PubkeyBech32Prefix = "lnpk"

// LnAddressData is all of the data that can be encoded in a parsed address.
type LnAddressData struct {
	Pkh    *string
	Pubkey []byte
	IPAddr *string
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
func DumpAddressFormats(data LnAddressData) (map[string]string, error) {

	ret := make(map[string]string)

	if data.Port != LnDefaultPort && data.IPAddr != nil {
		return ret, fmt.Errorf("nondefault port specified but IP not specified")
	}

	// Full.
	if data.Pkh != nil && data.Pubkey != nil && data.IPAddr != nil {
		ret[AddrFmtFull] = fmt.Sprintf("%s:%x@%s:%d", *data.Pkh, data.Pubkey, *data.IPAddr, data.Port)
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
		ret[AddrFmtPubkey] = fmt.Sprintf("%x", data.Pubkey)
		if data.IPAddr != nil {
			if data.Port == LnDefaultPort {
				ret[AddrFmtPubkeyIP] = fmt.Sprintf("%x@%s", data.Pubkey, *data.IPAddr)
			}
			ret[AddrFmtPubkeyIPPort] = fmt.Sprintf("%x@%s:%d", data.Pubkey, *data.IPAddr, data.Port)
		}
	}
	
	// TODO more
	
	return ret, nil
	
}

/*
+0      q 	p 	z 	r 	y 	9 	x 	8
+8 	g 	f 	2 	t 	v 	d 	w 	0
+16 	s 	3 	j 	n 	5 	4 	k 	h
+24 	c 	e 	6 	m 	u 	a 	7 	l
*/

const bech32bodyregex = `[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{4,}`
const ipaddrregex = `[0-9]{1,3}(\.[0-9]{1,3}){3}`
const fulladdrregex = `ln1` + bech32bodyregex + `(:([0-9a-fA-F]{64}|lnpk1` + bech32bodyregex + `))?(@` + ipaddrregex + `(:[0-9]{,5})?)?`

// ParseLnAddrData takes a LnAddr and parses the internal data.
func ParseLnAddrData(addr LnAddr) (*LnAddressData, error) {
	return nil, nil // TODO
}
