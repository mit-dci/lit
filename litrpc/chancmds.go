package litrpc

import (
	"fmt"

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
	Peer        uint32 // who to make the channel with
	Capacity    int64  // later can be minimum capacity
	Roundup     int64  // ignore for now; can be used to round-up capacity
	InitialSend int64  // Initial send of -1 means "ALL"
}

func (r *LitRPC) FundChannel(args FundArgs, reply *StatusReply) error {

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

	// see if we have enough money before calling the funding function.  Not
	// strictly required but it's better to fail here instead of after net traffic.
	// The litNode doesn't actually know how much money the basewallet has.
	// So it could potentially try to start a channel and fail because it doesn't have
	// the money.  Safe enough as all it's done is requested a point, which is
	// idempotent on both sides.
	_, _, err := r.SCon.TS.PickUtxos(args.Capacity, true)
	if err != nil {
		return err
	}

	idx, err := r.Node.FundChannel(args.Peer, args.Capacity, args.InitialSend)
	if err != nil {
		return err
	}

	reply.Status = fmt.Sprintf("funded channel %d", idx)

	return nil
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

	if args.Amt > 100000000 || args.Amt < 1 {
		return fmt.Errorf("push %d, max push is 1 coin / 100000000", args.Amt)
	}

	fmt.Printf("push %d to (%d,%d)\n", args.Amt, args.PeerIdx, args.ChanIdx)

	r.Node.RemoteMtx.Lock()
	_, ok := r.Node.RemoteCons[args.PeerIdx]
	r.Node.RemoteMtx.Unlock()
	if !ok {
		return fmt.Errorf("not connected to specified peer %d ",
			args.PeerIdx)
	}

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

	qc, err := r.Node.GetQchanByIdx(args.PeerIdx, args.ChanIdx)
	if err != nil {
		return err
	}

	err = r.Node.CoopClose(qc)
	if err != nil {
		return err
	}
	reply.Status = "OK closed"

	return nil
}

// ------------------------- break
func (r *LitRPC) BreakChannel(args ChanArgs, reply *StatusReply) error {

	r.Node.RemoteMtx.Lock()
	_, ok := r.Node.RemoteCons[args.PeerIdx]
	r.Node.RemoteMtx.Unlock()
	if !ok {
		return fmt.Errorf("not connected to specified peer %d ",
			args.PeerIdx)
	}

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

}
