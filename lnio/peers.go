package lnio

// LnAddr is just a bech32-encoded pubkey
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
	Nickname *string `json:"name"`
	NetAddr  string  `json:"netaddr"` // ip address, port, I guess
}
