package lncore

// LitPeerStorage is storage for peer data.
type LitPeerStorage interface {
	GetPeerAddrs() ([]string, error)
	GetPeerInfo(addr string) (PeerInfo, error)
	GetPeerInfos() (map[string]PeerInfo, error)
	AddPeer(addr string, pi PeerInfo) error
	UpdatePeer(addr string, pi PeerInfo) error
	DeletePeer(addr string) error

	// TEMP Until we figure this all out.
	GetUniquePeerIdx() (uint32, error)
}

// PeerInfo stores peer related data
type PeerInfo struct {
	Addr   string `json:"string"`
	Nickname string `json:"name"`
	NetAddr  string `json:"netaddr"` // ip address, port, I guess

	// TEMP This is again, for adapting to the old system.
	PeerIdx uint32 `json:"hint_peeridx"`
}
