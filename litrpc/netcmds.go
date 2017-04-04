package litrpc

import (
	"fmt"
	"strconv"

	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"
)

// ------------------------- listen

type ListenArgs struct {
	Port string
}

func (r *LitRPC) Listen(args ListenArgs, reply *StatusReply) error {
	if args.Port == "" {
		args.Port = ":2448"
	}
	adr, err := r.Node.TCPListener(args.Port)
	if err != nil {
		return err
	}
	// todo: say what port and what pubkey in status message
	reply.Status = fmt.Sprintf("listening on %s with key %s",
		args.Port, adr)
	return nil
}

// ------------------------- connect
type ConnectArgs struct {
	LNAddr string
}

func (r *LitRPC) Connect(args ConnectArgs, reply *StatusReply) error {

	// first, see if the peer to connect to is referenced by peer index.
	var connectAdr string
	// check if a peer number was supplied instead of a pubkeyhash
	peerIdxint, err := strconv.Atoi(args.LNAddr)
	// number will mean no error
	if err == nil {
		// get peer from address book
		pubArr, host := r.Node.GetPubHostFromPeerIdx(uint32(peerIdxint))

		connectAdr = lnutil.LitAdrFromPubkey(pubArr)
		if host != "" {
			connectAdr += "@" + host
		}
		fmt.Printf("try string %s\n", connectAdr)

	} else {
		// use string as is, try to convert to ln address
		connectAdr = args.LNAddr
	}

	err = r.Node.DialPeer(connectAdr)
	if err != nil {
		return err
	}

	reply.Status = fmt.Sprintf("connected to peer %s", connectAdr)
	return nil
}

// ------------------------- ShowConnections

type ListConnectionsReply struct {
	Connections []qln.PeerInfo
	MyPKH       string
}
type ConInfo struct {
	PeerNumber uint32
	RemoteHost string
}

func (r *LitRPC) ListConnections(args NoArgs, reply *ListConnectionsReply) error {
	reply.Connections = r.Node.GetConnectedPeerList()
	return nil
}

type ListeningPortsReply struct {
	LisIpPorts []string
	Adr        string
}

func (r *LitRPC) GetListeningPorts(args NoArgs, reply *ListeningPortsReply) error {
	reply.Adr, reply.LisIpPorts = r.Node.GetLisAddressAndPorts()

	return nil
}

// ------- receive chat
func (r *LitRPC) GetMessages(args NoArgs, reply *StatusReply) error {
	reply.Status = <-r.Node.UserMessageBox
	return nil
}

type SayArgs struct {
	Peer    uint32
	Message string
}

func (r *LitRPC) Say(args SayArgs, reply *StatusReply) error {
	return r.Node.SendChat(args.Peer, args.Message)
}

func (r *LitRPC) Stop(args NoArgs, reply *StatusReply) error {
	reply.Status = "Stopping lit node"
	r.OffButton <- true
	return nil
}
