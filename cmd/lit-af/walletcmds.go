package main

import (
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var invoiceCommand = &Command{
	Format: fmt.Sprintf(
		"%s%s%s%s\n", lnutil.White("invoice"), lnutil.OptColor("pay"), lnutil.OptColor("gen"), lnutil.OptColor("ls")),
	Description:      "Pay, Generate or view invoices\n",
	ShortDescription: "Pay, Generate or view invoices\n",
}

var payCommand = &Command{
	Format: fmt.Sprintf(
		"%s%s\n", lnutil.White("pay"), lnutil.ReqColor("invoice")),
	Description:      "Pay the given amount in satoshis to the given address defined by the invoice.\n",
	ShortDescription: "Pay invoice for the given amount.\n",
}

var genCommand = &Command{
	Format: fmt.Sprintf(
		"%s%s%s\n", lnutil.White("gen"), lnutil.ReqColor("cointype"), lnutil.ReqColor("amount")),
	Description:      "Request the passed amount in satoshis / equivalent units as an invoice.\n",
	ShortDescription: "Generate invoice for the given amount and cointype\n",
}

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

func (lc *litAfClient) Invoice(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, invoiceCommand.Format)
		fmt.Fprintf(color.Output, invoiceCommand.Description)
		return nil
	}

	if len(textArgs) == 0 {
		fmt.Println("pick a command: pay, gen, ls")
		return nil
	}

	cmd := textArgs[0]
	if cmd == "gen" {
		return lc.GenInvoice(textArgs[1:])
	}
	if cmd == "pay" {
		return lc.PayInvoice(textArgs[1:])
	}
	if cmd == "ls" {
		return lc.ListInvoices()
	}
	return fmt.Errorf("Invalid command passed along with invoice")
}

func (lc *litAfClient) GenInvoice(textArgs []string) error {
	stopEx, err := CheckHelpCommand(genCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.GenInvoiceArgs)
	reply := new(litrpc.GenInvoiceReply)

	args.CoinType = textArgs[0]
	amount, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	args.Amount = uint64(amount)

	err = lc.Call("LitRPC.GenInvoice", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "generated invoice %s worth %d %s coins", reply.Invoice, amount, args.CoinType)
	return nil
}

func (lc *litAfClient) PayInvoice(textArgs []string) error {
	stopEx, err := CheckHelpCommand(payCommand, textArgs, 1)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.PayInvoiceArgs)
	reply := new(litrpc.PayInvoiceReply)

	args.Invoice = textArgs[0]
	err = lc.Call("LitRPC.PayInvoice", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "Paid invoice: %s", args.Invoice)
	return nil
}

func (lc *litAfClient) ListInvoices() error {
	// so we need to pritn a list of all invoices here. For this, we need
	// to receive all the invoices from all the databases and parse them in a nice format
	// so that we can view stuff clearly. Somewhat similar to channel stuff, but
	// less dense
	reply := new(litrpc.LsInvoiceReplyMsg)
	replydup := new(litrpc.LsInvoiceMsg)
	args := new(litrpc.NoArgs)

	fmt.Fprintf(color.Output, "%s\n", lnutil.Green("Our Invoices"))
	err := lc.Call("LitRPC.ListAllGeneratedInvoices", args, reply)
	if err != nil {
		return err
	}
	if len(reply.Invoices) > 0 {
		fmt.Fprintf(color.Output, "%s\n", lnutil.Header("Active"))
		for _, invoice := range reply.Invoices {
			// its is an InvoiceReplyMsg
			fmt.Fprintf(color.Output, "%s: %s, %s: %d, %s: %s\n\n",
				lnutil.OutPoint("InvoiceID"), invoice.Id, lnutil.OutPoint("Amount"),
					invoice.Amount, lnutil.OutPoint("CoinType"), invoice.CoinType)
		}
	}
	err = lc.Call("LitRPC.ListAllRepliedInvoices", args, replydup)
	if err != nil {
		return err
	}
	if len(replydup.Invoices) > 0 {
		fmt.Fprintf(color.Output, "%s\n", lnutil.Header("Replied"))
		for _, invoice := range replydup.Invoices {
			// its is an InvoiceReplyMsg
			fmt.Fprintf(color.Output, "%s: %d, %s: %s\n\n",
				lnutil.OutPoint("Peer"), invoice.PeerIdx, lnutil.OutPoint("InvoiceID"), invoice.Id)
		}
	}
	err = lc.Call("LitRPC.ListAllGotPaidInvoices", args, reply)
	if err != nil {
		return err
	}
	if len(reply.Invoices) > 0 {
		fmt.Fprintf(color.Output, "%s\n", lnutil.Header("Got Paid"))
		for _, invoice := range reply.Invoices {
			// its is an InvoiceReplyMsg
			fmt.Fprintf(color.Output, "%s: %d, %s: %s, %s: %d, %s: %s\n\n",
				lnutil.OutPoint("Peer"), invoice.PeerIdx, lnutil.OutPoint("InvoiceID"), invoice.Id,
				lnutil.OutPoint("Amount"), invoice.Amount, lnutil.OutPoint("CoinType"), invoice.CoinType)
		}
	}
	fmt.Fprintf(color.Output, "%s\n", lnutil.Green("Remote Peers' Invoices"))
	// these are invoices that we pay for
	err = lc.Call("LitRPC.ListAllRequestedInvoices", args, replydup)
	if err != nil {
		return err
	}
	if len(replydup.Invoices) > 0 {
		fmt.Fprintf(color.Output, "%s\n", lnutil.Header("Requested"))
		for _, invoice := range replydup.Invoices {
			// its is an InvoiceReplyMsg
			fmt.Fprintf(color.Output, "%s: %d, %s: %s\n\n",
				lnutil.OutPoint("Peer"), invoice.PeerIdx, lnutil.OutPoint("InvoiceID"), invoice.Id)
		}
	}
	err = lc.Call("LitRPC.ListAllPendingInvoices", args, reply)
	if err != nil {
		return err
	}
	if len(reply.Invoices) > 0 {
		fmt.Fprintf(color.Output, "%s\n", lnutil.Header("Pending"))
		for _, invoice := range reply.Invoices {
			// its is an InvoiceReplyMsg
			fmt.Fprintf(color.Output, "%s: %d, %s: %s, %s: %d, %s: %s\n\n",
				lnutil.OutPoint("Peer"), invoice.PeerIdx, lnutil.OutPoint("InvoiceID"), invoice.Id,
				lnutil.OutPoint("Amount"), invoice.Amount, lnutil.OutPoint("CoinType"), invoice.CoinType)
		}
	}
	err = lc.Call("LitRPC.ListAllPaidInvoices", args, reply)
	if err != nil {
		return err
	}
	if len(reply.Invoices) > 0 {
		fmt.Fprintf(color.Output, "%s\n", lnutil.Header("Paid"))
		for _, invoice := range reply.Invoices {
			// its is an InvoiceReplyMsg
			fmt.Fprintf(color.Output, "%s: %d, %s: %s, %s: %d, %s: %s\n\n",
				lnutil.OutPoint("Peer"), invoice.PeerIdx, lnutil.OutPoint("InvoiceID"), invoice.Id,
				lnutil.OutPoint("Amount"), invoice.Amount, lnutil.OutPoint("CoinType"), invoice.CoinType)
		}
	}
	return nil
}

// Send sends coins somewhere
func (lc *litAfClient) Send(textArgs []string) error {
	stopEx, err := CheckHelpCommand(sendCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.SendArgs)
	reply := new(litrpc.TxidsReply)

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

	err = lc.Call("LitRPC.Send", args, reply)
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
	stopEx, err := CheckHelpCommand(sweepCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.SweepArgs)
	reply := new(litrpc.TxidsReply)

	args.DestAdr = textArgs[0]
	numTxs, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	args.NumTx = uint32(numTxs)
	if len(textArgs) > 2 {
		args.Drop = true
	}

	err = lc.Call("LitRPC.Sweep", args, reply)
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
	stopEx, err := CheckHelpCommand(fanCommand, textArgs, 3)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.FanArgs)
	reply := new(litrpc.TxidsReply)
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

	err = lc.Call("LitRPC.Fanout", args, reply)
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
		err := lc.Call("LitRPC.SetFee", SetArgs, reply)
		if err != nil {
			return err
		}
	} else {
		err := lc.Call("LitRPC.GetFee", GetArgs, reply)
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
	stopEx, err := CheckHelpCommand(addressCommand, textArgs, 0)
	if err != nil || stopEx {
		return err
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
	err = lc.Call("LitRPC.Address", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "new adr(s): %s\nold: %s\n",
		lnutil.Address(reply.WitAddresses), lnutil.Address(reply.LegacyAddresses))
	return nil

}
