package lnp2p

import (
	"net"
	"strings"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lncore"
	"golang.org/x/net/proxy"
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
func convertPubkeyToLitAddr(pk pubkey) lncore.LnAddr {
	b := (*koblitz.PublicKey)(pk).SerializeCompressed()
	doubleSha := fastsha256.Sum256(b[:])
	return lncore.LnAddr(bech32.Encode("ln", doubleSha[:20]))
}

func connectToProxyTCP(addr string, auth *string) (func(string, string) (net.Conn, error), error) {
	// Authentication is good.  Use it if it's there.
	var pAuth *proxy.Auth
	if auth != nil {
		parts := strings.SplitN(*auth, ":", 2)
		pAuth = &proxy.Auth{
			User:     parts[0],
			Password: parts[1],
		}
	}

	// Actually attempt to connect to the SOCKS Proxy.
	d, err := proxy.SOCKS5("tcp", addr, pAuth, proxy.Direct)
	if err != nil {
		return nil, err
	}

	return d.Dial, nil
}
