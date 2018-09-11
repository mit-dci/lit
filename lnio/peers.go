package lnio

// LnAddr is just a bech32-encoded pubkey.
// TODO Move this to another package so it's more obviously not *just* IO-related.
type LnAddr string

// LitPeerStorage is storage for peer data.
type LitPeerStorage interface {
	GetPeerAddrs() ([]LnAddr, error)
	GetPeerInfo(addr LnAddr) (*PeerInfo, error)
	GetPeerInfos() (map[LnAddr]PeerInfo, error)
	AddPeer(addr LnAddr, pi PeerInfo) error
	DeletePeer(addr LnAddr) error
}

// PeerInfo .
type PeerInfo struct {
	LnAddr   *LnAddr `json:"lnaddr"`
	Nickname *string `json:"name"`
	NetAddr  string  `json:"netaddr"` // ip address, port, I guess
}
