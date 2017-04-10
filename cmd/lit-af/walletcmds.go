package main

import (
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var sendCommand = &Command{
	Format:           fmt.Sprintf("%s%s\n", lnutil.White("send"), lnutil.ReqColor("address", "amount")),
	Description:      "Send the given amount of satoshis to the given address.\n",
	ShortDescription: "Send the given amount of satoshis to the given address.\n",
}

var addressCommand = &Command{
	Format:           fmt.Sprintf("%s%s\n", lnutil.White("address"), lnutil.ReqColor("?amount")),
	Description:      "Makes a new address.\n",
	ShortDescription: "Makes a new address.\n",
}

var fanCommand = &Command{
	Format:           fmt.Sprintf("%s%s\n", lnutil.White("fan"), lnutil.ReqColor("addr", "howmany", "howmuch")),
	Description:      "\n",
	ShortDescription: "\n",
	// TODO: Add description.
}

var sweepCommand = &Command{
	Format:      fmt.Sprintf("%s%s%s\n", lnutil.White("sweep"), lnutil.ReqColor("addr", "howmany"), lnutil.OptColor("drop")),
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

// Address makes new addresses
func (lc *litAfClient) Address(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, addressCommand.Format)
		fmt.Fprintf(color.Output, addressCommand.Description)
		return nil
	}

	args := new(litrpc.AddressArgs)

	// if no arguments given, generate 1 new address.
	if len(textArgs) < 1 {
		args.NumToMake = 1
	} else {
		num, _ := strconv.Atoi(textArgs[0])
		args.NumToMake = uint32(num)
	}

	reply := new(litrpc.AddressReply)
	err := lc.rpccon.Call("LitRPC.Address", args, reply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, "new adr(s): %s\nold: %s\n",
		lnutil.Address(reply.WitAddresses), lnutil.Address(reply.LegacyAddresses))
	return nil

}
