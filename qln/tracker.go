package qln

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/adiabat/btcd/btcec"
)

type announcement struct {
	url  string
	addr string
	sig  string
	pbk  string
}

type nodeinfo struct {
	success bool
	node    struct {
		url  string
		addr string
	}
}

func Announce(priv *btcec.PrivateKey, litport string, litadr string) error {
	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	liturl := strings.TrimSpace(buf.String()) + litport

	urlBytes := []byte(liturl)

	urlHash := sha256.Sum256(urlBytes)

	urlSig, err := priv.Sign(urlHash[:])
	if err != nil {
		return err
	}

	var ann announcement

	ann.url = liturl
	ann.addr = litadr
	ann.sig = hex.EncodeToString(urlSig.Serialize())
	ann.pbk = hex.EncodeToString(priv.PubKey().SerializeCompressed())

	_, err = http.PostForm("http://localhost:46580/announce",
		url.Values{"url": {ann.url},
			"addr": {ann.addr},
			"sig":  {ann.sig},
			"pbk":  {ann.pbk}})

	if err != nil {
		return err
	}

	return nil
}

func Lookup(litadr string) (string, error) {
	resp, err := http.Get("http://localhost:46580/" + litadr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var node nodeinfo
	err = decoder.Decode(&node)
	if err != nil {
		return "", err
	}

	if !node.success {
		return "", errors.New("Node not found")
	}

	return node.node.url, nil
}
