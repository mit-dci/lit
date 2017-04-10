package main

import (
	"bytes"
	"fmt"

	"io/ioutil"
	"net/http"

	"github.com/fatih/color"
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

var stopCommand = &Command{
	Format:           lnutil.White("stop\n"),
	Description:      "Shut down the lit node.\n",
	ShortDescription: "Shut down the lit node.\n",
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
		if err != nil {
			fmt.Fprintf(color.Output, "help error: %s\n", err)
		}
		return nil
	}
	// address a new address and displays it
	if cmd == "adr" {
		err = lc.Address(args)
		if err != nil {
			fmt.Fprintf(color.Output, "adr error: %s\n", err)
		}
		return nil
	}

	// ls shows the current set of utxos, addresses and score
	if cmd == "ls" {
		err = lc.Ls(args)
		if err != nil {
			fmt.Fprintf(color.Output, "ls error: %s\n", err)
		}
		return nil
	}

	// send sends coins to the address specified
	if cmd == "send" {
		err = lc.Send(args)
		if err != nil {
			fmt.Fprintf(color.Output, "send error: %s\n", err)
		}
		return nil
	}

	if cmd == "lis" { // listen for lnd peers
		err = lc.Lis(args)
		if err != nil {
			fmt.Fprintf(color.Output, "lis error: %s\n", err)
		}
		return nil
	}

	if cmd == "stop" { // stop remote node
		// actually returns an error
		return lc.Stop(args)
	}

	if cmd == "sweep" { // make lots of 1-in 1-out txs
		err = lc.Sweep(args)
		if err != nil {
			fmt.Fprintf(color.Output, "sweep error: %s\n", err)
		}
		return nil
	}

	// push money in a channel away from you
	if cmd == "push" {
		err = lc.Push(args)
		if err != nil {
			fmt.Fprintf(color.Output, "push error: %s\n", err)
		}
		return nil
	}

	if cmd == "con" { // connect to lnd host
		err = lc.Connect(args)
		if err != nil {
			fmt.Fprintf(color.Output, "con error: %s\n", err)
		}
		return nil
	}
	// fund and create a new channel
	if cmd == "fund" {
		err = lc.FundChannel(args)
		if err != nil {
			fmt.Fprintf(color.Output, "fund error: %s\n", err)
		}
		return nil
	}

	// cooperateive close of a channel
	if cmd == "close" {
		err = lc.CloseChannel(args)
		if err != nil {
			fmt.Fprintf(color.Output, "close error: %s\n", err)
		}
		return nil
	}
	if cmd == "break" {
		err = lc.BreakChannel(args)
		if err != nil {
			fmt.Fprintf(color.Output, "break error: %s\n", err)
		}
		return nil
	}
	if cmd == "say" {
		err = lc.Say(args)
		if err != nil {
			fmt.Fprintf(color.Output, "say error: %s\n", err)
		}
		return nil
	}

	if cmd == "fan" { // fan-out tx
		err = lc.Fan(args)
		if err != nil {
			fmt.Fprintf(color.Output, "fan error: %s\n", err)
		}
		return nil
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
	bReply := new(litrpc.BalReply)
	sReply := new(litrpc.SyncHeightReply)
	lReply := new(litrpc.ListeningPortsReply)

	err := lc.rpccon.Call("LitRPC.ListConnections", nil, pReply)
	if err != nil {
		return err
	}
	if len(pReply.Connections) > 0 {
		fmt.Fprintf(color.Output, "\t%s\n", lnutil.Header("Peers:"))
		for _, peer := range pReply.Connections {
			fmt.Fprintf(color.Output, "%s %s\n",
				lnutil.White(peer.PeerNumber), peer.RemoteHost)
		}
	}

	err = lc.rpccon.Call("LitRPC.ChannelList", nil, cReply)
	if err != nil {
		return err
	}
	if len(cReply.Channels) > 0 {
		fmt.Fprintf(color.Output, "\t%s\n", lnutil.Header("Channels:"))
	}

	for _, c := range cReply.Channels {
		if c.Closed {
			fmt.Fprintf(color.Output, lnutil.Red("Closed  "))
		} else {
			fmt.Fprintf(color.Output, lnutil.Green("Channel "))
		}
		fmt.Fprintf(color.Output, "%s (peer %d) %s\n\t cap: %s bal: %s h: %d state: %d\n",
			lnutil.White(c.CIdx), c.PeerIdx, lnutil.OutPoint(c.OutPoint),
			lnutil.SatoshiColor(c.Capacity), lnutil.SatoshiColor(c.MyBalance),
			c.Height, c.StateNum)
	}

	err = lc.rpccon.Call("LitRPC.TxoList", nil, tReply)
	if err != nil {
		return err
	}
	if len(tReply.Txos) > 0 {
		fmt.Fprintf(color.Output, lnutil.Header("\tTxos:\n"))
	}
	for i, t := range tReply.Txos {
		fmt.Fprintf(color.Output, "%d %s h:%d amt:%s %s",
			i, lnutil.OutPoint(t.OutPoint), t.Height,
			lnutil.SatoshiColor(t.Amt), t.KeyPath)
		if t.Delay != 0 {
			fmt.Fprintf(color.Output, " delay: %d", t.Delay)
		}
		if !t.Witty {
			fmt.Fprintf(color.Output, " non-witness")
		}
		fmt.Fprintf(color.Output, "\n")
	}

	err = lc.rpccon.Call("LitRPC.GetListeningPorts", nil, lReply)
	if err != nil {
		return err
	}
	if len(lReply.LisIpPorts) > 0 {
		fmt.Fprintf(color.Output, "\t%s\n", lnutil.Header("Listening Ports:"))
		fmt.Fprintf(color.Output,
			"Listening for connections on port(s) %v with key %s\n",
			lnutil.White(lReply.LisIpPorts), lReply.Adr)
	}

	err = lc.rpccon.Call("LitRPC.Address", nil, aReply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, lnutil.Header("\tAddresses:\n"))
	for i, a := range aReply.WitAddresses {
		fmt.Fprintf(color.Output, "%d %s (%s)\n", i,
			lnutil.Address(a), lnutil.Address(aReply.LegacyAddresses[i]))
	}
	err = lc.rpccon.Call("LitRPC.Bal", nil, bReply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "\t%s %s %s %s %s %s\n",
		lnutil.Header("Utxo:"), lnutil.SatoshiColor(bReply.TxoTotal),
		lnutil.Header("WitConf:"), lnutil.SatoshiColor(bReply.MatureWitty),
		lnutil.Header("Channel:"), lnutil.SatoshiColor(bReply.ChanTotal))

	err = lc.rpccon.Call("LitRPC.SyncHeight", nil, sReply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s %d\n",
		lnutil.Header("Sync Height:"), sReply.SyncHeight)

	return nil
}

func (lc *litAfClient) Stop(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, stopCommand.Format)
		fmt.Fprintf(color.Output, stopCommand.Description)
		return nil
	}

	reply := new(litrpc.StatusReply)

	err := lc.rpccon.Call("LitRPC.Stop", nil, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)

	lc.rpccon.Close()
	return fmt.Errorf("stopped remote lit node")
}

func (lc *litAfClient) Help(textArgs []string) error {
	if len(textArgs) == 0 {
		fmt.Fprintf(color.Output, "commands:\n")
		fmt.Fprintf(color.Output, "%s\t%s", helpCommand.Format, helpCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", sayCommand.Format, sayCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", lsCommand.Format, lsCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", addressCommand.Format, addressCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", sendCommand.Format, sendCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", fanCommand.Format, fanCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", sweepCommand.Format, sweepCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", lisCommand.Format, lisCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", conCommand.Format, conCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", fundCommand.Format, fundCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", pushCommand.Format, pushCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", closeCommand.Format, closeCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", breakCommand.Format, breakCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", stopCommand.Format, stopCommand.ShortDescription)
		fmt.Fprintf(color.Output, "%s\t%s", exitCommand.Format, exitCommand.ShortDescription)
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
