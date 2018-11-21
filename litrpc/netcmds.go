package litrpc

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lnp2p"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/qln"
)

// ------------------------- testlog

func (r *LitRPC) TestLog(arg string, reply *string) error {
	logging.Info(arg)
	*reply = arg
	return nil
}

// ------------------------- listen

type ListenArgs struct {
	Port int
}

type ListeningPortsReply struct {
	LisIpPorts []string
	Adr        string
}

func (r *LitRPC) Listen(args ListenArgs, reply *ListeningPortsReply) error {
	// this case is covered in lit-af, but in case other clients want to call this
	// RPC with an empty argument, ints default to 0 but there might be people who
	// want the computer to do the port assignment. To do it the proper way, they
	// can call the RPC with port set to -1
	if args.Port == -1 {
		args.Port = 2448
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

type ConnectReply struct {
	Status  string
	PeerIdx uint32
}

func (r *LitRPC) Connect(args ConnectArgs, reply *ConnectReply) error {

	// first, see if the peer to connect to is referenced by peer index.
	var connectAdr string
	// check if a peer number was supplied instead of a pubkeyhash
	// accept either an string or a pubkey (raw)
	// so args.LNAddr passed here contains blah@host:ip
	fmt.Println("PASSED STUFF:", args.LNAddr)
	peerIdxint, err := strconv.Atoi(args.LNAddr)
	// number will mean no error
	if err == nil {
		// get peer from address book
		pubArr, host := r.Node.GetPubHostFromPeerIdx(uint32(peerIdxint))

		connectAdr = lnutil.LitAdrFromPubkey(pubArr)
		if host != "" {
			connectAdr += "@" + host
		}
		logging.Infof("try string %s\n", connectAdr)

	} else {
		// use string as is, try to convert to ln address
		connectAdr = args.LNAddr
	}

	err = r.Node.PeerMan.TryConnectAddress(connectAdr, nil)
	if err != nil {
		return err
	}

	// Extract the plain lit addr since we don't always have it.
	// TODO Make this more "correct" since it's using the old system a lot.
	paddr := connectAdr
	if strings.Contains(paddr, "@") {
		paddr = strings.SplitN(paddr, "@", 2)[0]
	}

	var pm *lnp2p.PeerManager = r.Node.PeerMan
	p := pm.GetPeer(lncore.LnAddr(paddr))
	if p == nil {
		return fmt.Errorf("couldn't find peer in manager after connecting")
	}

	reply.Status = fmt.Sprintf("connected to peer %s", connectAdr)
	reply.PeerIdx = p.GetIdx()
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
	Connections []qln.SimplePeerInfo
	MyPKH       string
}
type ConInfo struct {
	PeerNumber uint32
	RemoteHost string
}

func (r *LitRPC) ListConnections(args NoArgs, reply *ListConnectionsReply) error {
	reply.Connections = r.Node.GetConnectedPeerList()
	reply.MyPKH = r.Node.GetLnAddr()
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

// ------------ Show multihop payments
type MultihopPaymentInfo struct {
	RHash     [32]byte
	R         [16]byte
	Amt       int64
	Path      []string
	Succeeded bool
}

type MultihopPaymentsReply struct {
	Payments []MultihopPaymentInfo
}

func (r *LitRPC) ListMultihopPayments(args NoArgs, reply *MultihopPaymentsReply) error {
	r.Node.MultihopMutex.Lock()
	defer r.Node.MultihopMutex.Unlock()
	for _, p := range r.Node.InProgMultihop {
		var path []string
		for _, hop := range p.Path {
			path = append(path, fmt.Sprintf("%s:%d", bech32.Encode("ln", hop.Node[:]), hop.CoinType))
		}

		i := MultihopPaymentInfo{
			p.HHash,
			p.PreImage,
			p.Amt,
			path,
			p.Succeeded,
		}

		reply.Payments = append(reply.Payments, i)
	}

	return nil
}

type RCAuthArgs struct {
	PubKey        []byte
	Authorization *qln.RemoteControlAuthorization
}

func (r *LitRPC) RemoteControlAuth(args RCAuthArgs, reply *StatusReply) error {

	pub, err := koblitz.ParsePubKey(args.PubKey, koblitz.S256())
	if err != nil {
		reply.Status = fmt.Sprintf("Error deserializing pubkey: %s", err.Error())
		return err
	}
	compressedPubKey := pub.SerializeCompressed()
	var pubKey [33]byte
	copy(pubKey[:], compressedPubKey[:])

	args.Authorization.UnansweredRequest = false

	err = r.Node.SaveRemoteControlAuthorization(pubKey, args.Authorization)
	if err != nil {
		logging.Errorf("Error saving auth: %s", err.Error())
		return err
	}

	action := "Granted"
	if !args.Authorization.Allowed {
		action = "Denied / revoked"
	}
	reply.Status = fmt.Sprintf("%s remote control access for pubkey [%x]", action, pubKey)
	return nil
}

type RCRequestAuthArgs struct {
	PubKey [33]byte
}

func (r *LitRPC) RequestRemoteControlAuthorization(args RCRequestAuthArgs, reply *StatusReply) error {
	auth := new(qln.RemoteControlAuthorization)
	auth.Allowed = false
	auth.UnansweredRequest = true

	err := r.Node.SaveRemoteControlAuthorization(args.PubKey, auth)
	if err != nil {
		logging.Errorf("Error saving auth request: %s", err.Error())
		return err
	}

	reply.Status = fmt.Sprintf("Access requested for pubkey [%x]", args.PubKey)
	return nil
}

type RCPendingAuthRequestsReply struct {
	PubKeys [][33]byte
}

func (r *LitRPC) ListPendingRemoteControlAuthRequests(args NoArgs, reply *RCPendingAuthRequestsReply) error {
	auth := new(qln.RemoteControlAuthorization)
	auth.Allowed = false
	auth.UnansweredRequest = true

	requests, err := r.Node.GetPendingRemoteControlRequests()
	if err != nil {
		logging.Errorf("Error saving auth request: %s", err.Error())
		return err
	}

	reply.PubKeys = make([][33]byte, len(requests))
	for i, r := range requests {
		reply.PubKeys[i] = r.PubKey
	}

	return nil

}
