package main

import (
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var sendCommand = &Command{
	Format: fmt.Sprintf(
		"%s%s\n", lnutil.White("send"), lnutil.ReqColor("address", "amount")),
	Description:      "Send the given amount of satoshis to the given address.\n",
	ShortDescription: "Send the given amount of satoshis to the given address.\n",
}

var addressCommand = &Command{
	Format: fmt.Sprintf(
		"%s%s\n", lnutil.White("adr"), lnutil.ReqColor("?amount", "?cointype")),
	Description:      "Makes new addresses in a specified wallet.\n",
	ShortDescription: "Makes new addresses.\n",
}

var fanCommand = &Command{
	Format: fmt.Sprintf(
		"%s%s\n", lnutil.White("fan"), lnutil.ReqColor("addr", "howmany", "howmuch")),
	Description:      "\n",
	ShortDescription: "\n",
	// TODO: Add description.
}

var sweepCommand = &Command{
	Format: fmt.Sprintf(
		"%s%s%s\n", lnutil.White("sweep"),
		lnutil.ReqColor("addr", "howmany"), lnutil.OptColor("drop")),
	Description: "Move UTXOs with many 1-in-1-out txs.\n",
	// TODO: Make this more clear.
	ShortDescription: "Move UTXOs with many 1-in-1-out txs.\n",
}

// Send sends coins somewhere
func (lc *litAfClient) Send(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, sendCommand.Format)
		fmt.Fprintf(color.Output, sendCommand.Description)
		return nil
	}

	args := new(litrpc.SendArgs)
	reply := new(litrpc.TxidsReply)

	// need args, fail
	if len(textArgs) < 2 {
		return fmt.Errorf(sendCommand.Description)
	}
	/*
		adr, err := btcutil.DecodeAddress(args[0], lc.Param)
		if err != nil {
			fmt.Fprintf(color.Output,"error parsing %s as address\t", args[0])
			return err
		}
	*/
	amt, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "send %d to address: %s \n", amt, textArgs[0])

	args.DestAddrs = []string{textArgs[0]}
	args.Amts = []int64{int64(amt)}

	err = lc.rpccon.Call("LitRPC.Send", args, reply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, "sent txid(s):\n")
	for i, t := range reply.Txids {
		fmt.Fprintf(color.Output, "\t%d %s\n", i, t)
	}
	return nil
}

// Sweep moves utxos with many 1-in-1-out txs
func (lc *litAfClient) Sweep(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, sweepCommand.Format)
		fmt.Fprintf(color.Output, sweepCommand.Description)
		return nil
	}

	args := new(litrpc.SweepArgs)
	reply := new(litrpc.TxidsReply)

	var err error

	if len(textArgs) < 2 {
		return fmt.Errorf(sweepCommand.Format)
	}

	args.DestAdr = textArgs[0]
	numTxs, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	args.NumTx = uint32(numTxs)
	if len(textArgs) > 2 {
		args.Drop = true
	}

	err = lc.rpccon.Call("LitRPC.Sweep", args, reply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, "Swept\n")
	for i, t := range reply.Txids {
		fmt.Fprintf(color.Output, "%d %s\n", i, t)
	}

	return nil
}

//// ------------------------- fanout
//type FanArgs struct {
//	DestAdr      string
//	NumOutputs   uint32
//	AmtPerOutput int64
//}

func (lc *litAfClient) Fan(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, fanCommand.Format)
		fmt.Fprintf(color.Output, fanCommand.Description)
		return nil
	}

	args := new(litrpc.FanArgs)
	reply := new(litrpc.TxidsReply)
	if len(textArgs) < 3 {
		return fmt.Errorf(fanCommand.Format)
	}
	var err error
	args.DestAdr = textArgs[0]

	outputs, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	args.NumOutputs = uint32(outputs)

	amt, err := strconv.Atoi(textArgs[2])
	if err != nil {
		return err
	}
	args.AmtPerOutput = int64(amt)

	err = lc.rpccon.Call("LitRPC.Fanout", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "Fanout:\n")
	for i, t := range reply.Txids {
		fmt.Fprintf(color.Output, "\t%d %s\n", i, t)
	}
	return nil
}

// ------------------ get / set fee

func (lc *litAfClient) Fee(textArgs []string) error {

	reply := new(litrpc.FeeReply)

	SetArgs := new(litrpc.SetFeeArgs)
	GetArgs := new(litrpc.FeeArgs)

	// whether we are setting the fee or not
	var set bool
	// There is an argument. That's the coin type. (coin type 0 means default)
	if len(textArgs) > 0 {
		feeint, err := strconv.Atoi(textArgs[0])
		if err != nil {
			fmt.Printf("Can't set fee to %s, querying current fee instead\n", textArgs[0])
		} else {
			set = true
			SetArgs.Fee = int64(feeint)
		}
	}

	if len(textArgs) > 1 {
		coinint, err := strconv.Atoi(textArgs[1])
		if err != nil {
			return err
		}
		SetArgs.CoinType = uint32(coinint)
		GetArgs.CoinType = uint32(coinint)
	}

	if set {
		err := lc.rpccon.Call("LitRPC.SetFee", SetArgs, reply)
		if err != nil {
			return err
		}
	} else {
		err := lc.rpccon.Call("LitRPC.GetFee", GetArgs, reply)
		if err != nil {
			return err
		}

	}

	fmt.Printf("Current fee rate %d sat / byte\n", reply.CurrentFee)

	return nil
}

// ------------------ set fee

func (lc *litAfClient) SetFee(textArgs []string) error {

	args := new(litrpc.SetFeeArgs)
	reply := new(litrpc.FeeReply)

	// there is at least 1 argument; that should be the new fee rate
	if len(textArgs) > 0 {
		feeint, err := strconv.Atoi(textArgs[0])
		if err != nil {
			return err
		}
		args.Fee = int64(feeint)
	}
	// there is another argument. That's the coin type. (coin type 0 means default)
	if len(textArgs) > 1 {
		coinint, err := strconv.Atoi(textArgs[1])
		if err != nil {
			return err
		}
		args.CoinType = uint32(coinint)
	}

	fmt.Printf("Current fee rate %d sat / byte\n", reply.CurrentFee)

	return nil
}

// Address makes new addresses
func (lc *litAfClient) Address(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, addressCommand.Format)
		fmt.Fprintf(color.Output, addressCommand.Description)
		return nil
	}

	var cointype, numadrs uint32

	// if no arguments given, generate 1 new address.
	// if no cointype given, assume type 1 (testnet)
	switch len(textArgs) {
	default: // meaning 2 or more args.  args 3+ are ignored
		cnum, err := strconv.Atoi(textArgs[1])
		if err != nil {
			return err
		}
		cointype = uint32(cnum)
		fallthrough
	case 1:
		num, err := strconv.Atoi(textArgs[0])
		if err != nil {
			return err
		}
		numadrs = uint32(num)
	case 0:
		// default one new address
		numadrs = 1
	}
	// cointype of 0 means default, not mainnet.
	// this is ugly but does prevent mainnet use for now.

	reply := new(litrpc.AddressReply)

	args := new(litrpc.AddressArgs)
	args.CoinType = cointype
	args.NumToMake = numadrs

	fmt.Printf("args: %v\n", args)
	err := lc.rpccon.Call("LitRPC.Address", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "new adr(s): %s\nold: %s\n",
		lnutil.Address(reply.WitAddresses), lnutil.Address(reply.LegacyAddresses))
	return nil

}

// ------------------ send raw tx
func (lc *litAfClient) RawTx(textArgs []string) error {
	args := new(litrpc.RawArgs)
	reply := new(litrpc.TxidsReply)

	// there is at least 1 argument; that should be the new fee rate
	if len(textArgs) == 0 {
		return fmt.Errorf("raw needs a hex string")
	}

	args.TxHex = textArgs[0]

	err := lc.rpccon.Call("LitRPC.PushRawTx", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "sent txid(s):\n")
	for i, t := range reply.Txids {
		fmt.Fprintf(color.Output, "\t%d %s\n", i, t)
	}
	return nil
}

// ------------------ imporTxo
func (lc *litAfClient) ImporTxo(textArgs []string) error {
	args := new(litrpc.RawArgs)
	reply := new(litrpc.StatusReply)

	// there is at least 1 argument; that should be the new fee rate
	if len(textArgs) == 0 {
		return fmt.Errorf("ImporTxo needs a hex string")
	}

	args.TxHex = textArgs[0]

	err := lc.rpccon.Call("LitRPC.ImporTxo", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "imported utxo:\n\t%s\n", reply.Status)
	return nil
}
