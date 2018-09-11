package lnp2p

import (
	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/lnio"
	"strings"
)

// ParseAdrString splits a string like
// "ln1yrvw48uc3atg8e2lzs43mh74m39vl785g4ehem@myhost.co:8191 into a separate
// pkh part and network part, adding the network part if needed
func splitAdrString(adr string) (string, string) {

	if !strings.ContainsRune(adr, ':') && strings.ContainsRune(adr, '@') {
		adr += ":2448"
	}

	idHost := strings.Split(adr, "@")

	if len(idHost) == 1 {
		return idHost[0], ""
	}

	return idHost[0], idHost[1]
}

// Given a raw pubkey, returns the lit addr.  Stolen from `lnutil/litadr.go`.
func convertPubkeyToLitAddr(pk pubkey) lnio.LnAddr {
	b := (*btcec.PublicKey)(pk).SerializeCompressed()
	doubleSha := fastsha256.Sum256(b[:])
	return lnio.LnAddr(bech32.Encode("ln", doubleSha[:20]))
}
