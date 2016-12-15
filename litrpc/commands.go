package litrpc

import (
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/portxo"
)

type TxidsReply struct {
	Txids []string
}
type StatusReply struct {
	Status string
}

type NoArgs struct {
	// nothin
}

// ------------------------- balance
// BalReply is the reply when the user asks about their balance.
// This is a Non-Channel
type BalReply struct {
	ChanTotal         int64
	TxoTotal          int64
	SpendableNow      int64
	SpendableNowWitty int64
}

func (r *LitRPC) Bal(args *NoArgs, reply *BalReply) error {
	// check current chain height; needed for time-locked outputs
	curHeight, err := r.SCon.TS.GetDBSyncHeight()
	if err != nil {
		return err
	}
	allTxos, err := r.SCon.TS.GetAllUtxos()
	if err != nil {
		return err
	}
	// iterate through utxos to figure out how much we have
	for _, u := range allTxos {
		reply.TxoTotal += u.Value
		if u.Seq == 0 {
			reply.SpendableNow += u.Value
		} else {
			if u.Seq == 1 || u.Height+int32(u.Seq) > curHeight {
				reply.SpendableNow += u.Value
				reply.SpendableNowWitty += u.Value
			}
		}
	}

	// get all channel states
	qcs, err := r.Node.GetAllQchans()
	if err != nil {
		return err
	}
	// iterate through channels to figure out how much we have
	for _, q := range qcs {
		reply.ChanTotal += q.State.MyAmt
	}

	return nil
}

type TxoInfo struct {
	OutPoint string
	Amt      int64
	Height   int32

	KeyPath string
}
type TxoListReply struct {
	Txos []TxoInfo
}

// TxoList sends back a list of all non-channel utxos
func (r *LitRPC) TxoList(args *NoArgs, reply *TxoListReply) error {
	allTxos, err := r.SCon.TS.GetAllUtxos()
	if err != nil {
		return err
	}

	reply.Txos = make([]TxoInfo, len(allTxos))
	for i, u := range allTxos {
		reply.Txos[i].Amt = u.Value
		reply.Txos[i].Height = u.Height
		reply.Txos[i].OutPoint = u.Op.String()
		reply.Txos[i].KeyPath = u.KeyGen.String()
	}
	return nil
}

type ChannelInfo struct {
	OutPoint  string
	Capacity  int64
	MyBalance int64
	PeerID    string
}
type ChannelListReply struct {
	Channels []ChannelInfo
}

// ChannelList sends back a list of every (open?) channel with some
// info for each.
func (r *LitRPC) ChannelList(args *NoArgs, reply *ChannelListReply) error {
	qcs, err := r.Node.GetAllQchans()
	if err != nil {
		return err
	}
	reply.Channels = make([]ChannelInfo, len(qcs))

	for i, q := range qcs {
		reply.Channels[i].OutPoint = q.Op.String()
		reply.Channels[i].Capacity = q.Value
		reply.Channels[i].MyBalance = q.State.MyAmt
		reply.Channels[i].PeerID = fmt.Sprintf("%x", q.PeerId)
	}
	return nil
}

// ------------------------- send
type SendArgs struct {
	DestAddrs []string
	Amts      []int64
}

func (r *LitRPC) Send(args SendArgs, reply *TxidsReply) error {
	var err error

	nOutputs := len(args.DestAddrs)
	if nOutputs < 1 {
		return fmt.Errorf("No destination address specified")
	}
	if nOutputs != len(args.Amts) {
		return fmt.Errorf("%d addresses but %d amounts specified",
			nOutputs, len(args.Amts))
	}

	adrs := make([]btcutil.Address, nOutputs)

	for i, s := range args.DestAddrs {
		adrs[i], err = btcutil.DecodeAddress(s, r.SCon.TS.Param)
		if err != nil {
			return err
		}
		if args.Amts[i] < 10000 {
			return fmt.Errorf("Amt %d less than min 10000", args.Amts[i])
		}
	}
	tx, err := r.SCon.TS.SendCoins(adrs, args.Amts)
	if err != nil {
		return err
	}
	err = r.SCon.NewOutgoingTx(tx)
	if err != nil {
		return err
	}

	reply.Txids = append(reply.Txids, tx.TxHash().String())
	return nil
}

// ------------------------- sweep
type SweepArgs struct {
	DestAdr string
	NumTx   int
	Drop    bool
}

func (r *LitRPC) Sweep(args SweepArgs, reply *TxidsReply) error {
	adr, err := btcutil.DecodeAddress(args.DestAdr, r.SCon.TS.Param)
	if err != nil {
		fmt.Printf("error parsing %s as address\t", args.DestAdr)
		return err
	}
	fmt.Printf("numtx: %d\n", args.NumTx)
	if args.NumTx < 1 {
		return fmt.Errorf("can't send %d txs", args.NumTx)
	}
	nokori := args.NumTx

	var allUtxos portxo.TxoSliceByAmt
	allUtxos, err = r.SCon.TS.GetAllUtxos()
	if err != nil {
		return err
	}

	// smallest and unconfirmed last (because it's reversed)
	sort.Sort(sort.Reverse(allUtxos))

	for i, u := range allUtxos {
		if u.Height != 0 && u.Value > 10000 {
			var txid chainhash.Hash
			if args.Drop {
				//				intx, outtx, err := SCon.TS.SendDrop(*allUtxos[i], adr)
				//				if err != nil {
				//					return err
				//				}
				//				txid = outtx.TxSha()
				//				err = SCon.NewOutgoingTx(intx)
				//				if err != nil {
				//					return err
				//				}
				//				err = SCon.NewOutgoingTx(outtx)
				//				if err != nil {
				//					return err
				//				}
			} else {
				tx, err := r.SCon.TS.SendOne(*allUtxos[i], adr)
				if err != nil {
					return err
				}
				txid = tx.TxHash()
				err = r.SCon.NewOutgoingTx(tx)
				if err != nil {
					return err
				}
			}
			reply.Txids = append(reply.Txids, txid.String())
			nokori--
			if nokori == 0 {
				return nil
			}
		}
	}

	fmt.Printf("spent all confirmed utxos; not enough by %d\n", nokori)
	return nil
}

// ------------------------- fanout
type FanArgs struct {
	DestAdr      string
	NumOutputs   uint32
	AmtPerOutput int64
}

func (r *LitRPC) Fanout(args FanArgs, reply *TxidsReply) error {
	if args.NumOutputs < 1 {
		return fmt.Errorf("Must have at least 1 output")
	}
	if args.AmtPerOutput < 5000 {
		return fmt.Errorf("Minimum 5000 per output")
	}
	adr, err := btcutil.DecodeAddress(args.DestAdr, r.SCon.TS.Param)
	if err != nil {
		fmt.Printf("error parsing %s as address\t", args.DestAdr)
		return err
	}
	adrs := make([]btcutil.Address, args.NumOutputs)
	amts := make([]int64, args.NumOutputs)

	for i := int64(0); i < int64(args.NumOutputs); i++ {
		adrs[i] = adr
		amts[i] = args.AmtPerOutput + i
	}
	tx, err := r.SCon.TS.SendCoins(adrs, amts)
	if err != nil {
		return err
	}
	err = r.SCon.NewOutgoingTx(tx)
	if err != nil {
		return err
	}
	reply.Txids = append(reply.Txids, tx.TxHash().String())
	return nil
}

// ------------------------- listen
func (r *LitRPC) Listen(args NoArgs, reply *StatusReply) error {

	err := r.Node.TCPListener(":2448")
	if err != nil {
		return err
	}
	// todo: say what port and what pubkey in status message
	reply.Status = fmt.Sprintf("listening on %s", r.Node.RemoteCon.LocalAddr().String())
	return nil
}

// ------------------------- connect
type ConnectArgs struct {
	LNAddr string
}

// ------------------------- address
type AdrArgs struct {
	NumToMake uint32
}
type AdrReply struct {
	//	PreviousAddresses []string
	NewAddresses []string
}

func (r *LitRPC) Address(args *AdrArgs, reply *AdrReply) error {
	//	reply.PreviousAddresses = make([]string, len(SCon.TS.Adrs))
	reply.NewAddresses = make([]string, args.NumToMake)

	//	for i, a := range SCon.TS.Adrs {
	//		reply.PreviousAddresses[i] = a.PkhAdr.String()
	//	}

	nokori := args.NumToMake
	for nokori > 0 {
		a160, err := r.SCon.TS.NewAdr160()
		if err != nil {
			return err
		}

		wa, err := btcutil.NewAddressWitnessPubKeyHash(a160, r.SCon.Param)
		if err != nil {
			return err
		}
		reply.NewAddresses[nokori-1] = wa.String()
		nokori--
	}
	return nil
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

	return nil
}

// ------------------------- fund
type FundArgs struct {
	LNAddr      string
	Capacity    int64 // later can be minimum capacity
	Roundup     int64 // ignore for now; can be used to round-up capacity
	InitialSend int64 // Initial send of -1 means "ALL"
}

func (r *LitRPC) Fund(args FundArgs, reply *StatusReply) error {
	if args.InitialSend > args.Capacity {
		return fmt.Errorf("Initial send more than capacity")
	}

	return nil
}

// ------------------------- push
type PushArgs struct {
	PeerIdx, QChanIdx uint32
	Amt               int64
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
		args.Amt, args.PeerIdx, args.QChanIdx)

	qc, err := r.Node.GetQchanByIdx(args.PeerIdx, args.QChanIdx)
	if err != nil {
		return err
	}
	err = r.Node.PushChannel(qc, uint32(args.Amt))
	if err != nil {
		return err
	}
	reply.MyAmt = qc.State.MyAmt
	reply.StateIndex = qc.State.StateIdx
	return nil
}
