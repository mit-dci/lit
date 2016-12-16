package litrpc

import (
	"fmt"

	"github.com/btcsuite/btcutil"
	"github.com/mit-dci/lit/lndc"
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
		args.Port, adr.String())
	return nil
}

// ------------------------- connect
type ConnectArgs struct {
	LNAddr string
}

func (r *LitRPC) Connect(args ConnectArgs, reply *StatusReply) error {

	connectNode, err := lndc.LnAddrFromString(args.LNAddr)
	if err != nil {
		return err
	}

	// get my private ID key
	idPriv := r.Node.IdKey()

	// Assign remote connection
	r.Node.RemoteCon = new(lndc.LNDConn)

	err = r.Node.RemoteCon.Dial(idPriv,
		connectNode.NetAddr.String(), connectNode.Base58Adr.ScriptAddress())
	if err != nil {
		return err
	}

	// store this peer in the db
	_, err = r.Node.NewPeer(r.Node.RemoteCon.RemotePub)
	if err != nil {
		return err
	}

	idslice := btcutil.Hash160(r.Node.RemoteCon.RemotePub.SerializeCompressed())
	var newId [16]byte
	copy(newId[:], idslice[:16])
	go r.Node.LNDCReceiver(r.Node.RemoteCon, newId)
	reply.Status = fmt.Sprintf("connected to %x",
		r.Node.RemoteCon.RemotePub.SerializeCompressed())
	return nil
}

type SayArgs struct {
	Message string
}

func (r *LitRPC) Say(args SayArgs, reply *StatusReply) error {

	if r.Node.RemoteCon == nil || r.Node.RemoteCon.RemotePub == nil {
		return fmt.Errorf("Not connected to anyone\n")
	}

	msg := append([]byte{qln.MSGID_TEXTCHAT}, []byte(args.Message)...)

	_, err := r.Node.RemoteCon.Write(msg)
	return err
}

func (r *LitRPC) Stop(args NoArgs, reply *StatusReply) error {
	reply.Status = "Stopping lit node"
	r.OffButton <- true
	return nil
}
