package lncore

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
