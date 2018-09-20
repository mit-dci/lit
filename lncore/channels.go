package lncore

// LitChannelStorage .
type LitChannelStorage interface {
	GetChannel(handle ChannelHandle) (*ChannelInfo, error)
	GetChannelHandles() ([]ChannelHandle, error)
	GetChannels() ([]ChannelInfo, error)

	AddChannel(handle ChannelHandle, info ChannelInfo) error
	UpdateChannel(handle ChannelHandle, info ChannelInfo) error
	ArchiveChannel(handle ChannelHandle) error

	GetArchivedChannelHandles() ([]ChannelHandle, error)
}

// ChannelHandle is "something" to concisely uniquely identify a channel.  Usually just a txid.
type ChannelHandle [64]byte

// ChannelState .
type ChannelState uint8

const (
	// CstateInit means it hasn't been broadcast yet.
	CstateInit = 0

	// CstateUnconfirmed means it's been broadcast but not included yet.
	CstateUnconfirmed = 1

	// CstateOK means it's been included and the channel is active.
	CstateOK = 2

	// CstateClosing means the close tx has been broadcast but not included yet.
	CstateClosing = 3

	// CstateBreaking means it's the break tx has been broadcast but not included and spendable yet.
	CstateBreaking = 4

	// CstateClosed means nothing else should be done with the channel anymore.
	CstateClosed = 5

	// CstateError means something went horribly wrong and we may need extra intervention.
	CstateError = 255
)

// ChannelInfo .
type ChannelInfo struct {
	PeerAddr   string       `json:"peeraddr"`
	CoinType   int32        `json:"cointype"`
	State      ChannelState `json:"state"`
	OpenTx     []byte       `json:"opentx"`     // should this be here?
	OpenHeight int32        `json:"openheight"` // -1 if unconfirmed
	// TODO More.
}
