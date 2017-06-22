package litrpc

import (
	"fmt"

	"github.com/adiabat/btcutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/qln"
)

type ChannelInfo struct {
	OutPoint      string
	CoinType      uint32
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

	if args.ChanIdx == 0 {
		qcs, err = r.Node.GetAllQchans()
		if err != nil {
			return err
		}
	} else {
		qc, err := r.Node.GetQchanByIdx(args.ChanIdx)
		if err != nil {
			return err
		}
		qcs = append(qcs, qc)
	}

	reply.Channels = make([]ChannelInfo, len(qcs))

	for i, q := range qcs {
		reply.Channels[i].OutPoint = q.Op.String()
		reply.Channels[i].CoinType = q.Coin()
		reply.Channels[i].Closed = q.CloseData.Closed
		reply.Channels[i].Capacity = q.Value
		reply.Channels[i].MyBalance = q.State.MyAmt
		reply.Channels[i].Height = q.Height
		reply.Channels[i].StateNum = q.State.StateIdx
		reply.Channels[i].PeerIdx = q.KeyGen.Step[3] & 0x7fffffff
		reply.Channels[i].CIdx = q.KeyGen.Step[4] & 0x7fffffff
	}
	return nil
}

// ------------------------- fund
type FundArgs struct {
	Peer        uint32 // who to make the channel with
	CoinType    uint32 // what coin to use
	Capacity    int64  // later can be minimum capacity
	Roundup     int64  // ignore for now; can be used to round-up capacity
	InitialSend int64  // Initial send of -1 means "ALL"
}

func (r *LitRPC) FundChannel(args FundArgs, reply *StatusReply) error {
	var err error
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

	wal := r.Node.SubWallet[args.CoinType]
	if wal == nil {
		return fmt.Errorf("No wallet of cointype %d linked", args.CoinType)
	}

	nowHeight := wal.CurrentHeight()

	// see if we have enough money before calling the funding function.  Not
	// strictly required but it's better to fail here instead of after net traffic.
	// also assume a fee of like 50K sat just to be safe
	var allPorTxos portxo.TxoSliceByAmt
	allPorTxos, err = wal.UtxoDump()
	if err != nil {
		return err
	}

	spendable := allPorTxos.SumWitness(nowHeight)

	if args.Capacity > spendable-50000 {
		return fmt.Errorf("Wanted %d but %d available for channel creation",
			args.Capacity, spendable-50000)
	}

	idx, err := r.Node.FundChannel(
		args.Peer, args.CoinType, args.Capacity, args.InitialSend)
	if err != nil {
		return err
	}

	reply.Status = fmt.Sprintf("funded channel %d", idx)

	return nil
}

// ------------------------- push
type PushArgs struct {
	ChanIdx uint32
	Amt     int64
}
type PushReply struct {
	StateIndex uint64
}

// Push is the command to push miney to the other side of the channel.
// Currently waits for the process to complete before returning.
// Will change to .. tries to send, but may not complete.

func (r *LitRPC) Push(args PushArgs, reply *PushReply) error {

	if args.Amt > 100000000 || args.Amt < 1 {
		return fmt.Errorf(
			"can't push %d max is 1 coin (100000000), min is 1", args.Amt)
	}

	fmt.Printf("push %d to chan %d\n", args.Amt, args.ChanIdx)

	// load the whole channel from disk just to see who the peer is
	// (pretty inefficient)
	dummyqc, err := r.Node.GetQchanByIdx(args.ChanIdx)
	if err != nil {
		return err
	}
	// see if channel is closed and error early
	if dummyqc.CloseData.Closed {
		return fmt.Errorf("Can't push; channel %d closed", args.ChanIdx)
	}

	// but we want to reference the qc that's already in ram
	// first see if we're connected to that peer

	// map read, need mutex...?
	r.Node.RemoteMtx.Lock()
	peer, ok := r.Node.RemoteCons[dummyqc.Peer()]
	r.Node.RemoteMtx.Unlock()
	if !ok {
		return fmt.Errorf("not connected to peer %d for channel %d",
			dummyqc.Peer(), dummyqc.Idx())
	}
	qc, ok := peer.QCs[dummyqc.Idx()]
	if !ok {
		return fmt.Errorf("peer %d doesn't have channel %d",
			dummyqc.Peer(), dummyqc.Idx())
	}

	fmt.Printf("channel %s\n", qc.Op.String())

	if qc.CloseData.Closed {
		return fmt.Errorf("Channel %d already closed by tx %s",
			args.ChanIdx, qc.CloseData.CloseTxid.String())
	}

	err = r.Node.PushChannel(qc, uint32(args.Amt))
	if err != nil {
		return err
	}

	reply.StateIndex = qc.State.StateIdx
	return nil
}

// ------------------------- cclose
type ChanArgs struct {
	ChanIdx uint32
}

// reply with status string
// CloseChannel is a cooperative closing of a channel to a specified address.
func (r *LitRPC) CloseChannel(args ChanArgs, reply *StatusReply) error {

	qc, err := r.Node.GetQchanByIdx(args.ChanIdx)
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

	qc, err := r.Node.GetQchanByIdx(args.ChanIdx)
	if err != nil {
		return err
	}
	return r.Node.BreakChannel(qc)
}

// ------------------------- dumpPriv
type PrivInfo struct {
	OutPoint string
	Amt      int64
	Height   int32
	Delay    int32
	CoinType string
	Witty    bool
	PairKey  string

	WIF string
}

type DumpReply struct {
	Privs []PrivInfo
}

// DumpPrivs returns WIF private keys for every utxo and channel
func (r *LitRPC) DumpPrivs(args NoArgs, reply *DumpReply) error {
	// get wifs for all channels
	qcs, err := r.Node.GetAllQchans()
	if err != nil {
		return err
	}

	for _, qc := range qcs {
		wal, ok := r.Node.SubWallet[qc.Coin()]
		if !ok {
			return fmt.Errorf("Coin %d not connected; can't show keys", qc.Coin())
		}
		var thisTxo PrivInfo
		thisTxo.OutPoint = qc.Op.String()
		thisTxo.Amt = qc.Value
		thisTxo.Height = qc.Height
		thisTxo.CoinType = wal.Params().Name
		thisTxo.Witty = true
		thisTxo.PairKey = fmt.Sprintf("%x", qc.TheirPub)

		priv := wal.GetPriv(qc.KeyGen)
		wif := btcutil.WIF{priv, true, wal.Params().PrivateKeyID}
		thisTxo.WIF = wif.String()

		reply.Privs = append(reply.Privs, thisTxo)
	}

	// get WIFs for all utxos in the wallets
	for _, wal := range r.Node.SubWallet {
		walTxos, err := wal.UtxoDump()
		if err != nil {
			return err
		}

		syncHeight := wal.CurrentHeight()

		theseTxos := make([]PrivInfo, len(walTxos))
		for i, u := range walTxos {
			theseTxos[i].OutPoint = u.Op.String()
			theseTxos[i].Amt = u.Value
			theseTxos[i].Height = u.Height
			theseTxos[i].CoinType = wal.Params().Name
			// show delay before utxo can be spent
			if u.Seq != 0 {
				theseTxos[i].Delay = u.Height + int32(u.Seq) - syncHeight
			}
			theseTxos[i].Witty = u.Mode&portxo.FlagTxoWitness != 0
			priv := wal.GetPriv(u.KeyGen)
			wif := btcutil.WIF{priv, true, wal.Params().PrivateKeyID}

			theseTxos[i].WIF = wif.String()
		}

		reply.Privs = append(reply.Privs, theseTxos...)
	}

	return nil
}
