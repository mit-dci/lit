package litrpc

import (
	"fmt"

	"github.com/adiabat/bech32"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
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
	ChanTotal   int64 // total balance in channels
	TxoTotal    int64 // all utxos
	MatureWitty int64 // confirmed, spendable and witness
}

func (r *LitRPC) Bal(args *NoArgs, reply *BalReply) error {
	var err error
	var allTxos portxo.TxoSliceByAmt

	nowHeight := r.Node.SubWallet.CurrentHeight()

	allTxos, err = r.Node.SubWallet.UtxoDump()
	if err != nil {
		return err
	}

	// ask sub-wallet for balance
	reply.TxoTotal = allTxos.Sum()
	reply.MatureWitty = allTxos.SumWitness(nowHeight)

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
	Delay    int32
	Witty    bool

	KeyPath string
}
type TxoListReply struct {
	Txos []TxoInfo
}

// TxoList sends back a list of all non-channel utxos
func (r *LitRPC) TxoList(args *NoArgs, reply *TxoListReply) error {
	allTxos, err := r.Node.SubWallet.UtxoDump()
	if err != nil {
		return err
	}

	syncHeight := r.Node.SubWallet.CurrentHeight()

	reply.Txos = make([]TxoInfo, len(allTxos))
	for i, u := range allTxos {
		reply.Txos[i].OutPoint = u.Op.String()
		reply.Txos[i].Amt = u.Value
		reply.Txos[i].Height = u.Height
		// show delay before utxo can be spent
		if u.Seq != 0 {
			reply.Txos[i].Delay = u.Height + int32(u.Seq) - syncHeight
		}
		reply.Txos[i].Witty = u.Mode&portxo.FlagTxoWitness != 0
		reply.Txos[i].KeyPath = u.KeyGen.String()
	}
	return nil
}

type SyncHeightReply struct {
	SyncHeight   int32
	HeaderHeight int32
}

func (r *LitRPC) SyncHeight(args *NoArgs, reply *SyncHeightReply) error {
	reply.SyncHeight = r.Node.SubWallet.CurrentHeight()
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

	txOuts := make([]*wire.TxOut, nOutputs)
	for i, s := range args.DestAddrs {
		if args.Amts[i] < 10000 {
			return fmt.Errorf("Amt %d less than min 10000", args.Amts[i])
		}

		outScript, err := AdrStringToOutscript(s, r.Node.SubWallet.Params())
		if err != nil {
			return err
		}

		txOuts[i] = wire.NewTxOut(args.Amts[i], outScript)
	}

	// we don't care if it's witness or not
	ops, err := r.Node.SubWallet.MaybeSend(txOuts, false)
	if err != nil {
		return err
	}

	err = r.Node.SubWallet.ReallySend(&ops[0].Hash)
	if err != nil {
		return err
	}

	reply.Txids = append(reply.Txids, ops[0].Hash.String())
	return nil
}

// ------------------------- sweep
type SweepArgs struct {
	DestAdr string
	NumTx   uint32
	Drop    bool
}

// AdrStringToOutscript converts an address string into an output script byte slice
func AdrStringToOutscript(adr string, p *chaincfg.Params) ([]byte, error) {
	var err error
	var outScript []byte
	if adr[:3] == "tb1" || adr[:3] == "bc1" {
		// try bech32 address
		outScript, err = bech32.SegWitAddressDecode(adr)
		if err != nil {
			return nil, err
		}
	} else {
		// try for base58 address
		adr, err := btcutil.DecodeAddress(adr, p)
		if err != nil {
			return nil, err
		}
		outScript, err = txscript.PayToAddrScript(adr)
		if err != nil {
			return nil, err
		}
	}
	return outScript, nil
}

func (r *LitRPC) Sweep(args SweepArgs, reply *TxidsReply) error {

	outScript, err := AdrStringToOutscript(args.DestAdr, r.Node.SubWallet.Params())
	if err != nil {
		return err
	}

	fmt.Printf("numtx: %d\n", args.NumTx)
	if args.NumTx < 1 {
		return fmt.Errorf("can't send %d txs", args.NumTx)
	}

	txids, err := r.Node.SubWallet.Sweep(outScript, args.NumTx)
	if err != nil {
		return err
	}

	for _, txid := range txids {
		reply.Txids = append(reply.Txids, txid.String())
	}

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
	outScript, err := AdrStringToOutscript(args.DestAdr, r.Node.SubWallet.Params())
	if err != nil {
		return err
	}

	txos := make([]*wire.TxOut, args.NumOutputs)

	for i, _ := range txos {
		txos[i] = new(wire.TxOut)
		txos[i].Value = args.AmtPerOutput + int64(i)
		txos[i].PkScript = outScript
	}

	// don't care if inputs are witty or not
	ops, err := r.Node.SubWallet.MaybeSend(txos, false)
	if err != nil {
		return err
	}
	err = r.Node.SubWallet.ReallySend(&ops[0].Hash)
	if err != nil {
		return err
	}

	reply.Txids = append(reply.Txids, ops[0].String())
	return nil
}

// ------------------------- address
type AddressArgs struct {
	NumToMake uint32
}
type AddressReply struct {
	WitAddresses    []string
	LegacyAddresses []string
}

func (r *LitRPC) Address(args *AddressArgs, reply *AddressReply) error {

	// If you tell it to make 0 new addresses, it sends a list of all the old ones
	if args.NumToMake == 0 {

		// this gets old p2pkh addresses; need to convert them to bech32
		allAdr, err := r.Node.SubWallet.AdrDump()
		if err != nil {
			return err
		}

		reply.WitAddresses = make([]string, len(allAdr))
		reply.LegacyAddresses = make([]string, len(allAdr))
		for i, a := range allAdr {
			// add old address
			reply.LegacyAddresses[i] = a.String()
			// take 20-byte PKH out and convert to a bech32 segwit v0 address
			bech32adr, err := bech32.Tb1AdrFromPKH(a.ScriptAddress())
			if err != nil {
				return err
			}
			reply.WitAddresses[i] = bech32adr
		}

		return nil
	}

	reply.WitAddresses = make([]string, args.NumToMake)
	reply.LegacyAddresses = make([]string, args.NumToMake)

	remaining := args.NumToMake
	for remaining > 0 {
		adr := r.Node.SubWallet.NewAdr()

		reply.LegacyAddresses[remaining-1] = adr.String()

		// take 20-byte PKH out and convert to bech32 address
		bech32adr, err := bech32.Tb1AdrFromPKH(adr.ScriptAddress())
		if err != nil {
			return err
		}

		reply.WitAddresses[remaining-1] = bech32adr
		remaining--
	}

	return nil
}
