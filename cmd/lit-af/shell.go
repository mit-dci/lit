package main

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"io/ioutil"
	"net/http"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var lsCommand = &Command{
	Format:           lnutil.White("ls\n"),
	Description:      "Show various information about our current state, such as connections, addresses, UTXO's, balances, etc.\n",
	ShortDescription: "Show various information about our current state\n",
}

var exitCommand = &Command{
	Format:           lnutil.White("exit\n"),
	Description:      fmt.Sprintf("Alias: %s\nExit the interactive shell.\n", lnutil.White("quit")),
	ShortDescription: fmt.Sprintf("Alias: %s\nExit the interactive shell.\n", lnutil.White("quit")),
}

var helpCommand = &Command{
	Format:           fmt.Sprintf("%s%s\n", lnutil.White("help"), lnutil.OptColor("command")),
	Description:      "Show information about a given command\n",
	ShortDescription: "Show information about a given command\n",
}

var offCommand = &Command{
	Format:           lnutil.White("off"),
	Description:      "Shut down the lit node.\n",
	ShortDescription: "Shut down the lit node.\n",
}

func parseErr(err error, str string) error {
	if err != nil {
		fmt.Fprintf(color.Output, "%s error: %s\n", str, err)
	}
	return nil
}

// Shellparse parses user input and hands it to command functions if matching
func (lc *litAfClient) Shellparse(cmdslice []string) error {
	var err error
	var args []string
	cmd := cmdslice[0]
	if len(cmdslice) > 1 {
		args = cmdslice[1:]
	}

	if cmd == "exit" || cmd == "quit" {
		return lc.Exit(args)
	}
	// help gives you really terse help.  Just a list of commands.
	if cmd == "help" {
		err = lc.Help(args)
		return parseErr(err, "help")
	}

	if cmd == "watch" {
		err = lc.Watch(args)
		return parseErr(err, "watch")
	}

	// address a new address and displays it
	if cmd == "adr" {
		err = lc.Address(args)
		return parseErr(err, "adr")
	}

	// ls shows the current set of utxos, addresses and score
	if cmd == "ls" {
		err = lc.Ls(args)
		return parseErr(err, "ls")
	}

	// send sends coins to the address specified
	if cmd == "send" {
		err = lc.Send(args)
		return parseErr(err, "send")
	}

	if cmd == "lis" { // listen for lnd peers
		err = lc.Lis(args)
		return parseErr(err, "lis")
	}

	if cmd == "off" { // stop remote node
		// actually returns an error
		return lc.Stop(args)
	}

	if cmd == "sweep" { // make lots of 1-in 1-out txs
		err = lc.Sweep(args)
		return parseErr(err, "sweep")
	}

	// push money in a channel away from you
	if cmd == "push" {
		err = lc.Push(args)
		return parseErr(err, "push")
	}

	if cmd == "add" {
		err = lc.AddHTLC(args)
		return parseErr(err, "add")
	}

	if cmd == "clear" {
		err = lc.ClearHTLC(args)
		return parseErr(err, "clear")
	}

	if cmd == "claim" {
		err = lc.ClaimHTLC(args)
		return parseErr(err, "claim")
	}

	if cmd == "con" { // connect to lnd host
		err = lc.Connect(args)
		return parseErr(err, "con")
	}

	if cmd == "dlc" { // the root command for Discreet log contracts
		err = lc.Dlc(args)
		return parseErr(err, "dlc")
	}

	// fund and create a new channel
	if cmd == "fund" {
		err = lc.FundChannel(args)
		return parseErr(err, "fund")
	}

	// mutually fund and create a new channel
	if cmd == "dualfund" {
		if (len(args) > 0 && args[0] == "-h") || len(args) == 0 {
			err = lc.DualFund(args)
		} else {
			if args[0] == "start" {
				err = lc.DualFundChannel(args[1:])
			} else if args[0] == "decline" {
				err = lc.DualFundDecline(args[1:])
			} else if args[0] == "accept" {
				err = lc.DualFundAccept(args[1:])
			} else {
				fmt.Fprintf(color.Output, "dualfund error - unrecognized subcommand %s\n", args[0])
			}
		}

		if err != nil {
			fmt.Fprintf(color.Output, "dualfund error: %s\n", err)
		}
		return nil
	}
	// cooperative close of a channel
	if cmd == "close" {
		err = lc.CloseChannel(args)
		return parseErr(err, "close")
	}
	if cmd == "break" {
		err = lc.BreakChannel(args)
		return parseErr(err, "break")
	}
	if cmd == "say" {
		err = lc.Say(args)
		return parseErr(err, "say")
	}

	if cmd == "fan" { // fan-out tx
		err = lc.Fan(args)
		return parseErr(err, "fan")
	}
	if cmd == "fee" { // get fee rate for a wallet
		err = lc.Fee(args)
		return parseErr(err, "fee")
	}
	if cmd == "dump" { // dump all private keys
		err = lc.Dump(args)
		return parseErr(err, "dump")
	}
	if cmd == "history" { // dump justice tx history
		err = lc.History(args)
		return parseErr(err, "history")
	}
	if cmd == "graph" { // dump graphviz for channels
		err = lc.Graph(args)
		return parseErr(err, "grpah")
	}

	fmt.Fprintf(color.Output, "Command not recognized. type help for command list.\n")
	return nil
}

func (lc *litAfClient) Exit(textArgs []string) error {
	if len(textArgs) > 0 {
		if len(textArgs) == 1 && textArgs[0] == "-h" {
			fmt.Fprintf(color.Output, exitCommand.Format)
			fmt.Fprintf(color.Output, exitCommand.Description)
			return nil
		}
		fmt.Fprintf(color.Output, "Unexpected argument: "+textArgs[0])
		return nil
	}
	return fmt.Errorf("User exit")
}

func (lc *litAfClient) Ls2(textArgs []string) error {
	resp, err := http.Post("http://127.0.0.1:9750/lit",
		"application/json",
		bytes.NewBufferString(
			`{"jsonrpc":"2.0","id":1,"method":"LitRPC.TxoList","params":[]}`))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("JSON over HTTP response: %s\n", string(b))
	return nil
}

func (lc *litAfClient) Ls(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, lsCommand.Format)
		fmt.Fprintf(color.Output, lsCommand.Description)
		return nil
	}

	pReply := new(litrpc.ListConnectionsReply)
	cReply := new(litrpc.ChannelListReply)
	aReply := new(litrpc.AddressReply)
	tReply := new(litrpc.TxoListReply)
	bReply := new(litrpc.BalanceReply)
	lReply := new(litrpc.ListeningPortsReply)
	dfReply := new(litrpc.PendingDualFundReply)

	err := lc.Call("LitRPC.ListConnections", nil, pReply)
	if err != nil {
		return err
	}
	if len(pReply.Connections) > 0 {
		fmt.Fprintf(color.Output, "\t%s\n", lnutil.Header("Peers:"))
		for _, peer := range pReply.Connections {
			fmt.Fprintf(color.Output, "%s %s (%s)\n",
				lnutil.White(peer.PeerNumber), peer.RemoteHost, peer.LitAdr)
		}
	}

	err = lc.Call("LitRPC.ChannelList", nil, cReply)
	if err != nil {
		return err
	}
	if len(cReply.Channels) > 0 {
		fmt.Fprintf(color.Output, "\t%s\n", lnutil.Header("Channels:"))
	}

	sort.Slice(cReply.Channels, func(i, j int) bool {
		return cReply.Channels[i].Height < cReply.Channels[j].Height
	})

	var closedChannels []litrpc.ChannelInfo
	var openChannels []litrpc.ChannelInfo
	for _, c := range cReply.Channels {
		if c.Closed {
			closedChannels = append(closedChannels, c)
		} else {
			openChannels = append(openChannels, c)
		}
	}

	for _, c := range closedChannels {
		fmt.Fprintf(color.Output, lnutil.Red("Closed:       "))
		fmt.Fprintf(
			color.Output,
			"%s (peer %d) type %d %s\n\t cap: %s bal: %s h: %d state: %d data: %x pkh: %x\n",
			lnutil.White(c.CIdx), c.PeerIdx, c.CoinType,
			lnutil.OutPoint(c.OutPoint),
			lnutil.SatoshiColor(c.Capacity), lnutil.SatoshiColor(c.MyBalance),
			c.Height, c.StateNum, c.Data, c.Pkh)
	}

	for _, c := range openChannels {
		if c.Height <= 0 {
			c := color.New(color.FgGreen).Add(color.Underline)
			c.Printf("Unconfirmed:")
			fmt.Fprintf(color.Output, lnutil.Green("  "))
			//needed for preventing the underline from extending
		} else {
			fmt.Fprintf(color.Output, lnutil.Green("Open:         "))
		}
		fmt.Fprintf(
			color.Output,
			"%s (peer %d) type %d %s\n\t cap: %s bal: %s h: %d state: %d data: %x pkh: %x\n",
			lnutil.White(c.CIdx), c.PeerIdx, c.CoinType,
			lnutil.OutPoint(c.OutPoint),
			lnutil.SatoshiColor(c.Capacity), lnutil.SatoshiColor(c.MyBalance),
			c.Height, c.StateNum, c.Data, c.Pkh)
	}

	err = lc.Call("LitRPC.PendingDualFund", nil, dfReply)
	if err != nil {
		return err
	}
	if dfReply.Pending {
		fmt.Fprintf(color.Output, "\t%s\n", lnutil.Header("Pending Dual Funding Request:"))
		fmt.Fprintf(
			color.Output, "\t%s %d\t%s %d\t%s %s\t%s %s\n\n",
			lnutil.Header("Peer:"), dfReply.PeerIdx,
			lnutil.Header("Type:"), dfReply.CoinType,
			lnutil.Header("Their Amt:"), lnutil.SatoshiColor(dfReply.TheirAmount),
			lnutil.Header("Req Amt:"), lnutil.SatoshiColor(dfReply.RequestedAmount),
		)
	}

	err = lc.Call("LitRPC.TxoList", nil, tReply)

	if err != nil {
		return err
	}
	if len(tReply.Txos) > 0 {
		fmt.Fprintf(color.Output, lnutil.Header("\tTxos:\n"))
	}
	for i, t := range tReply.Txos {
		fmt.Fprintf(color.Output, "%d %s h:%d amt:%s %s %s",
			i+1, lnutil.OutPoint(t.OutPoint), t.Height,
			lnutil.SatoshiColor(t.Amt), t.KeyPath, t.CoinType)
		if t.Delay != 0 {
			fmt.Fprintf(color.Output, " delay: %d", t.Delay)
		}
		if !t.Witty {
			fmt.Fprintf(color.Output, " non-witness")
		}
		fmt.Fprintf(color.Output, "\n")
	}

	err = lc.Call("LitRPC.GetListeningPorts", nil, lReply)
	if err != nil {
		return err
	}
	if len(lReply.LisIpPorts) > 0 {
		fmt.Fprintf(color.Output, "\t%s\n", lnutil.Header("Listening Ports:"))
		fmt.Fprintf(color.Output,
			"Listening for connections on port(s) %v with key %s\n",
			lnutil.White(lReply.LisIpPorts), lReply.Adr)
	}

	err = lc.Call("LitRPC.Address", nil, aReply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, lnutil.Header("\tAddresses:\n"))
	for i, a := range aReply.WitAddresses {
		fmt.Fprintf(color.Output, "%d %s (%s)\n", i+1,
			lnutil.Address(a), lnutil.Address(aReply.LegacyAddresses[i]))
	}

	err = lc.Call("LitRPC.Balance", nil, bReply)
	if err != nil {
		return err
	}

	for _, walBal := range bReply.Balances {
		fmt.Fprintf(
			color.Output, "\t%s %d\t%s %d\t%s %d\t%s %s\t%s %s %s %s\n",
			lnutil.Header("Type:"), walBal.CoinType,
			lnutil.Header("Sync Height:"), walBal.SyncHeight,
			lnutil.Header("FeeRate:"), walBal.FeeRate,
			lnutil.Header("Utxo:"), lnutil.SatoshiColor(walBal.TxoTotal),
			lnutil.Header("WitConf:"), lnutil.SatoshiColor(walBal.MatureWitty),
			lnutil.Header("Channel:"), lnutil.SatoshiColor(walBal.ChanTotal),
		)
	}

	return nil
}

func (lc *litAfClient) Stop(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, offCommand.Format)
		fmt.Fprintf(color.Output, offCommand.Description)
		return nil
	}

	reply := new(litrpc.StatusReply)

	err := lc.Call("LitRPC.Stop", nil, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)

	lc.rpccon.Close()
	return fmt.Errorf("stopped remote lit node")
}

func printHelp(commands []*Command) {
	for _, command := range commands {
		fmt.Fprintf(color.Output, "%s\t%s", command.Format, command.ShortDescription)
	}
}

func printCointypes() {
	for k, v := range coinparam.RegisteredNets {
		fmt.Fprintf(color.Output, "CoinType: %s\n", strconv.Itoa(int(k)))
		fmt.Fprintf(color.Output, "└────── Name: %-13sBech32Prefix: %s\n\n", v.Name+",", v.Bech32Prefix)
	}
}

func (lc *litAfClient) Help(textArgs []string) error {
	if len(textArgs) == 0 {

		fmt.Fprintf(color.Output, lnutil.Header("Commands:\n"))
		listofCommands := []*Command{helpCommand, sayCommand, lsCommand, addressCommand, sendCommand, fanCommand, sweepCommand, lisCommand, conCommand, dlcCommand, fundCommand, dualFundCommand, watchCommand, pushCommand, closeCommand, breakCommand, addHTLCCommand, clearHTLCCommand, historyCommand, offCommand, exitCommand}
		printHelp(listofCommands)
		fmt.Fprintf(color.Output, "\n\n")
		fmt.Fprintf(color.Output, lnutil.Header("Coins:\n"))
		printCointypes()

		return nil
	}

	if textArgs[0] == "help" || textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, helpCommand.Format)
		fmt.Fprintf(color.Output, helpCommand.Description)
		return nil
	}
	res := make([]string, 0)
	res = append(res, textArgs[0])
	res = append(res, "-h")
	return lc.Shellparse(res)
}
