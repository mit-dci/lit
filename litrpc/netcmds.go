package litrpc

import (
	"fmt"
	"log"
	"strconv"

	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"
)

// ------------------------- testlog

func (r *LitRPC) TestLog(arg string, reply *string) error {
	log.Print(arg)
	*reply = arg
	return nil
}

// ------------------------- listen

type ListenArgs struct {
	Port string
}

type ListeningPortsReply struct {
	LisIpPorts []string
	Adr        string
}

func (r *LitRPC) Listen(args ListenArgs, reply *ListeningPortsReply) error {
	if args.Port == "" {
		args.Port = ":2448"
	}
	_, err := r.Node.TCPListener(args.Port)
	if err != nil {
		return err
	}

	reply.Adr, reply.LisIpPorts = r.Node.GetLisAddressAndPorts()

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
		log.Printf("try string %s\n", connectAdr)

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

// ------------------------- name a connection
type AssignNicknameArgs struct {
	Peer     uint32
	Nickname string
}

func (r *LitRPC) AssignNickname(args AssignNicknameArgs, reply *StatusReply) error {
	// attempt to save nickname to the database, this process also checks if the peer exists
	err := r.Node.SaveNicknameForPeerIdx(args.Nickname, args.Peer)
	if err != nil {
		return err
	}

	// it's okay if we aren't connected to this peer right now, but if we are
	// then their nickname needs to be updated in the remote connections list
	// otherwise this doesn't get updated til after a restart
	if peer, ok := r.Node.RemoteCons[args.Peer]; ok {
		peer.Nickname = args.Nickname
	}

	reply.Status = fmt.Sprintf("changed nickname of peer %d to %s",
		args.Peer, args.Nickname)
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

// ------------ Dump channel map
type ChannelGraphReply struct {
	Graph string
}

func (r *LitRPC) GetChannelMap(args NoArgs, reply *ChannelGraphReply) error {
	reply.Graph = r.Node.VisualiseGraph()
	return nil
}
