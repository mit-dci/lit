package lncore

import (
	"fmt"

	"github.com/mit-dci/lit/bech32"
)

// LnAddr is just a bech32-encoded pubkey.
// TODO Move this to another package so it's more obviously not *just* IO-related.
type LnAddr string

// ParseLnAddr will verify that the string passed is a valid LN address, as in
// ln1pmclh89haeswrw0unf8awuyqeu4t2uell58nea.
func ParseLnAddr(m string) (LnAddr, error) {

	prefix, raw, err := bech32.Decode(m)

	// Check it's valid bech32.
	if err != nil {
		return "", err
	}

	// Check it has the right prefix.
	if prefix != "ln" {
		return "", fmt.Errorf("prefix is not 'ln'")
	}

	// Check the length of the content bytes is right.
	if len(raw) > 20 {
		return "", fmt.Errorf("address too long to be pubkey")
	}

	return LnAddr(m), nil // should be the only place we cast to this type

}

// ToString returns the LnAddr as a string.  Right now it just unwraps it but it
// might do something more eventually.
func (lnaddr LnAddr) ToString() string {
	return string(lnaddr)
}

// LitPeerStorage is storage for peer data.
type LitPeerStorage interface {
	GetPeerAddrs() ([]LnAddr, error)
	GetPeerInfo(addr LnAddr) (*PeerInfo, error)
	GetPeerInfos() (map[LnAddr]PeerInfo, error)
	AddPeer(addr LnAddr, pi PeerInfo) error
	UpdatePeer(addr LnAddr, pi *PeerInfo) error
	DeletePeer(addr LnAddr) error

	// TEMP Until we figure this all out.
	GetUniquePeerIdx() (uint32, error)
}

// PeerInfo .
type PeerInfo struct {
	LnAddr   *LnAddr `json:"lnaddr"`
	Nickname *string `json:"name"`
	NetAddr  *string `json:"netaddr"` // ip address, port, I guess

	// TEMP This is again, for adapting to the old system.
	PeerIdx uint32 `json:"hint_peeridx"`
}
