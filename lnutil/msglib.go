package lnutil

// all the messages to and from peers look like this internally
type LitMsg struct {
	PeerIdx uint32
	MsgType uint8
	Data    []byte
}

const (
	MSGID_POINTREQ  = 0x30
	MSGID_POINTRESP = 0x31
	MSGID_CHANDESC  = 0x32
	MSGID_CHANACK   = 0x33
	MSGID_SIGPROOF  = 0x34

	MSGID_CLOSEREQ  = 0x40 // close channel
	MSGID_CLOSERESP = 0x41

	MSGID_TEXTCHAT = 0x70 // send a text message

	MSGID_DELTASIG = 0x80 // pushing funds in channel; request to send
	//	MSGID_ACKSIG = 0x81 // pulling funds in channel; acknowledge update and sign
	MSGID_SIGREV = 0x81 // pulling funds; signing new state and revoking old
	MSGID_REV    = 0x82 // pushing funds; revoking previous channel state

	MSGID_GAPSIGREV = 0x83 // resolving collision

	MSGID_FWDMSG     = 0x20
	MSGID_FWDAUTHREQ = 0x21
)
