package qln

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mit-dci/lit/btcutil/btcec"
	"golang.org/x/net/proxy"
)

type announcement struct {
	ipv4 string
	ipv6 string
	addr string
	sig  string
	pbk  string
}

type nodeinfo struct {
	Success bool
	Node    struct {
		IPv4 string
		IPv6 string
		Addr string
	}
}

func Announce(priv *btcec.PrivateKey, litport string, litadr string, trackerURL string) error {
	client := &http.Client{
		Timeout: time.Second * 4, // 4+4 to accomodate the 10s RPC timeout
	}
	resp, err := client.Get("https://ipv4.myexternalip.com/raw")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	liturlIPv4 := strings.TrimSpace(buf.String()) + litport

	var liturlIPv6 string

	/* TODO: Find a better way to get this information. Their
	 * SSL cert doesn't work for IPv6.
	 */
	resp, err = client.Get("http://ipv6.myexternalip.com/raw")
	if err != nil {
		log.Printf("%v", err)
	} else {
		defer resp.Body.Close()
		buf = new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		liturlIPv6 = strings.TrimSpace(buf.String()) + litport
	}

	urlBytes := []byte(liturlIPv4 + liturlIPv6)
	urlHash := sha256.Sum256(urlBytes)
	urlSig, err := priv.Sign(urlHash[:])
	if err != nil {
		return err
	}

	var ann announcement

	ann.ipv4 = liturlIPv4
	ann.ipv6 = liturlIPv6
	ann.addr = litadr
	ann.sig = hex.EncodeToString(urlSig.Serialize())
	ann.pbk = hex.EncodeToString(priv.PubKey().SerializeCompressed())

	_, err = http.PostForm(trackerURL+"/announce",
		url.Values{"ipv4": {ann.ipv4},
			"ipv6": {ann.ipv6},
			"addr": {ann.addr},
			"sig":  {ann.sig},
			"pbk":  {ann.pbk}})

	if err != nil {
		return err
	}

	return nil
}

func Lookup(litadr string, trackerURL string, proxyURL string) (string, string, error) {
	var client http.Client

	if proxyURL != "" {
		dialer, err := proxy.SOCKS5("tcp", proxyURL, nil, proxy.Direct)
		if err != nil {
			return "", "", err
		}

		client.Transport = &http.Transport{
			Dial: dialer.Dial,
		}
	}

	resp, err := client.Get(trackerURL + "/" + litadr)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var node nodeinfo
	err = decoder.Decode(&node)
	if err != nil {
		return "", "", err
	}

	if !node.Success {
		return "", "", errors.New("Node not found")
	}

	return node.Node.IPv4, node.Node.IPv6, nil
}
