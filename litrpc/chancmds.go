package litrpc

import (
	"fmt"

	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"
)

type ChannelInfo struct {
	OutPoint      string
	Closed        bool
	Capacity      int64
	MyBalance     int64
	Height        int32  // block height of channel fund confirmation
	StateNum      uint64 // Most recent commit number
	PeerIdx, CIdx uint32
	PeerID        string
}
type ChannelListReply struct {
	Channels []ChannelInfo
}

// ChannelList sends back a list of every (open?) channel with some
// info for each.
func (r *LitRPC) ChannelList(args ChanArgs, reply *ChannelListReply) error {
	var err error
	var qcs []*qln.Qchan

	if args.PeerIdx == 0 && args.ChanIdx == 0 {
		qcs, err = r.Node.GetAllQchans()
		if err != nil {
			return err
		}
	} else {
		qc, err := r.Node.GetQchanByIdx(args.PeerIdx, args.ChanIdx)
		if err != nil {
			return err
		}
		qcs = append(qcs, qc)
	}

	reply.Channels = make([]ChannelInfo, len(qcs))

	for i, q := range qcs {
		reply.Channels[i].OutPoint = q.Op.String()
		reply.Channels[i].Closed = q.CloseData.Closed
		reply.Channels[i].Capacity = q.Value
		reply.Channels[i].MyBalance = q.State.MyAmt
		reply.Channels[i].Height = q.Height
		reply.Channels[i].StateNum = q.State.StateIdx
		reply.Channels[i].PeerIdx = q.KeyGen.Step[3] & 0x7fffffff
		reply.Channels[i].CIdx = q.KeyGen.Step[4] & 0x7fffffff
		reply.Channels[i].PeerID = fmt.Sprintf("%x", q.PeerId)
	}
	return nil
}

// ------------------------- fund
type FundArgs struct {
	LNAddr      string
	Capacity    int64 // later can be minimum capacity
	Roundup     int64 // ignore for now; can be used to round-up capacity
	InitialSend int64 // Initial send of -1 means "ALL"
}

func (r *LitRPC) FundChannel(args FundArgs, reply *StatusReply) error {

	if r.Node.RemoteCon == nil || r.Node.RemoteCon.RemotePub == nil {
		return fmt.Errorf("Not connected to anyone")
	}

	if r.Node.InProg != nil && r.Node.InProg.PeerIdx != 0 {
		return fmt.Errorf("channel with peer %d not done yet", r.Node.InProg.PeerIdx)
	}

	if args.InitialSend < 0 || args.Capacity < 0 {
		return fmt.Errorf("Can't have negative send or capacity")
	}
	if args.Capacity < 1000000 { // limit for now
		return fmt.Errorf("Min channel capacity 1M sat")
	}
	if args.InitialSend > args.Capacity {
		return fmt.Errorf("Cant send %d in %d capacity channel",
			args.InitialSend, args.Capacity)
	}

	// see if we have enough money.  Doesn't freeze here though, just
	// checks for ability to fund.  Freeze happens when we receive the response.
	// Could fail if we run out of money before calling MaybeSend()
	_, _, err := r.SCon.TS.PickUtxos(args.Capacity, true)
	if err != nil {
		return err
	}

	var peerArr [33]byte
	copy(peerArr[:], r.Node.RemoteCon.RemotePub.SerializeCompressed())

	peerIdx, cIdx, err := r.Node.NextIdxForPeer(peerArr)
	if err != nil {
		return err
	}

	r.Node.InProg = new(qln.InFlightFund)

	r.Node.InProg.ChanIdx = cIdx
	r.Node.InProg.PeerIdx = peerIdx
	r.Node.InProg.Amt = args.Capacity
	r.Node.InProg.InitSend = args.InitialSend

	msg := []byte{qln.MSGID_POINTREQ}
	_, err = r.Node.RemoteCon.Write(msg)
	return err
}

// ------------------------- push
type PushArgs struct {
	PeerIdx, ChanIdx uint32
	Amt              int64
}
type PushReply struct {
	MyAmt      int64
	StateIndex uint64
}

func (r *LitRPC) Push(args PushArgs, reply *PushReply) error {
	if r.Node.RemoteCon == nil || r.Node.RemoteCon.RemotePub == nil {
		return fmt.Errorf("Not connected to anyone, can't push\n")
	}
	if args.Amt > 100000000 || args.Amt < 1 {
		return fmt.Errorf("push %d, max push is 1 coin / 100000000", args.Amt)
	}

	// find the peer index of who we're connected to
	currentPeerIdx, err := r.Node.GetPeerIdx(r.Node.RemoteCon.RemotePub)
	if err != nil {
		return err
	}
	if uint32(args.PeerIdx) != currentPeerIdx {
		return fmt.Errorf("Want to close with peer %d but connected to %d",
			args.PeerIdx, currentPeerIdx)
	}
	fmt.Printf("push %d to (%d,%d) %d times\n",
		args.Amt, args.PeerIdx, args.ChanIdx)

	qc, err := r.Node.GetQchanByIdx(args.PeerIdx, args.ChanIdx)
	if err != nil {
		return err
	}

	if qc.CloseData.Closed {
		return fmt.Errorf("Channel (%d,%d) already closed by tx %s",
			args.PeerIdx, args.ChanIdx, qc.CloseData.CloseTxid.String())
	}

	err = r.Node.PushChannel(qc, uint32(args.Amt))
	if err != nil {
		return err
	}
	reply.MyAmt = qc.State.MyAmt
	reply.StateIndex = qc.State.StateIdx
	return nil
}

// ------------------------- cclose
type ChanArgs struct {
	PeerIdx, ChanIdx uint32
}

// reply with status string
// CloseChannel is a cooperative closing of a channel to a specified address.
func (r *LitRPC) CloseChannel(args ChanArgs, reply *StatusReply) error {

	if r.Node.RemoteCon == nil || r.Node.RemoteCon.RemotePub == nil {
		return fmt.Errorf("Not connected to anyone\n")
	}

	// find the peer index of who we're connected to
	currentPeerIdx, err := r.Node.GetPeerIdx(r.Node.RemoteCon.RemotePub)
	if err != nil {
		return err
	}
	if args.PeerIdx != currentPeerIdx {
		return fmt.Errorf("Want to close with peer %d but connected to %d	",
			args.PeerIdx, currentPeerIdx)
	}

	qc, err := r.Node.GetQchanByIdx(args.PeerIdx, args.ChanIdx)
	if err != nil {
		return err
	}

	if qc.CloseData.Closed {
		return fmt.Errorf("can't close (%d,%d): already closed",
			qc.KeyGen.Step[3]&0x7fffffff, qc.KeyGen.Step[4]&0x7fffffff)
	}

	tx, err := qc.SimpleCloseTx()
	if err != nil {
		return err
	}

	sig, err := r.Node.SignSimpleClose(qc, tx)
	if err != nil {
		return err
	}

	// Save something to db... TODO
	// Should save something, just so the UI marks it as closed, and
	// we don't accept payments on this channel anymore.

	opArr := lnutil.OutPointToBytes(qc.Op)
	// close request is just the op, sig
	msg := []byte{qln.MSGID_CLOSEREQ}
	msg = append(msg, opArr[:]...)
	msg = append(msg, sig...)

	_, err = r.Node.RemoteCon.Write(msg)

	return err
}

// ------------------------- break
func (r *LitRPC) BreakChannel(args ChanArgs, reply *StatusReply) error {

	qc, err := r.Node.GetQchanByIdx(args.PeerIdx, args.ChanIdx)
	if err != nil {
		return err
	}

	if qc.CloseData.Closed {
		return fmt.Errorf("Can't break (%d,%d), already closed\n",
			qc.KeyGen.Step[3]&0x7fffffff, qc.KeyGen.Step[4]&0x7fffffff)
	}

	fmt.Printf("breaking (%d,%d)\n",
		qc.KeyGen.Step[3]&0x7fffffff, qc.KeyGen.Step[4]&0x7fffffff)
	z, err := qc.ElkSnd.AtIndex(0)
	if err != nil {
		return err
	}
	fmt.Printf("elk send 0: %s\n", z.String())
	z, err = qc.ElkRcv.AtIndex(0)
	if err != nil {
		return err
	}
	fmt.Printf("elk recv 0: %s\n", z.String())
	// set delta to 0...
	qc.State.Delta = 0
	tx, err := r.Node.SignBreakTx(qc)
	if err != nil {
		return err
	}

	// broadcast
	return r.Node.BaseWallet.PushTx(tx)

	return nil
}
