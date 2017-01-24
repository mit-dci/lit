package litrpc

import (
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
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
	Mature      int64 // confirmed and spendable
	MatureWitty int64 // confirmed, spendable and witness
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
		if u.Height > 0 && u.Height+int32(u.Seq) <= curHeight {
			reply.Mature += u.Value
			if u.Mode&portxo.FlagTxoWitness != 0 {
				reply.MatureWitty += u.Value
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
	Delay    int32
	Witty    bool

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
	syncHeight, err := r.SCon.TS.GetDBSyncHeight()
	if err != nil {
		return err
	}

	reply.Txos = make([]TxoInfo, len(allTxos))
	for i, u := range allTxos {
		reply.Txos[i].OutPoint = u.Op.String()
		reply.Txos[i].Amt = u.Value
		reply.Txos[i].Height = u.Height
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
	var err error
	reply.SyncHeight, err = r.SCon.TS.GetDBSyncHeight()
	return err
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

// ------------------------- address
type AdrArgs struct {
	NumToMake uint32
}
type AdrReply struct {
	WitAddresses    []string
	LegacyAddresses []string
}

func (r *LitRPC) Address(args *AdrArgs, reply *AdrReply) error {

	// If you tell it to make 0 new addresses, it sends a list of all the old ones
	if args.NumToMake == 0 {

		allAdr, err := r.SCon.TS.GetAllAddresses()
		if err != nil {
			return err
		}

		reply.WitAddresses = make([]string, len(allAdr))
		reply.LegacyAddresses = make([]string, len(allAdr))
		for i, a := range allAdr {
			// add witness address
			reply.WitAddresses[i] = a.String()

			// take 20-byte PKH out and convert to old address
			oldAdr, err := btcutil.NewAddressPubKeyHash(
				a.ScriptAddress(), r.SCon.Param)
			if err != nil {
				return err
			}
			reply.LegacyAddresses[i] = oldAdr.String()
		}

		return nil
	}

	reply.WitAddresses = make([]string, args.NumToMake)
	reply.LegacyAddresses = make([]string, args.NumToMake)

	remaining := args.NumToMake
	for remaining > 0 {
		a160, err := r.SCon.TS.NewAdr160()
		if err != nil {
			return err
		}

		wa, err := btcutil.NewAddressWitnessPubKeyHash(a160, r.SCon.Param)
		if err != nil {
			return err
		}
		reply.WitAddresses[remaining-1] = wa.String()

		// take 20-byte PKH out and convert to old address
		oldAdr, err := btcutil.NewAddressPubKeyHash(
			wa.ScriptAddress(), r.SCon.Param)
		if err != nil {
			return err
		}

		reply.LegacyAddresses[remaining-1] = oldAdr.String()
		remaining--
	}

	return nil
}
