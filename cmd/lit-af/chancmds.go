package main

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var fundCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("fund"),
		lnutil.ReqColor("peer", "coinType", "capacity", "initialSend"), lnutil.OptColor("data")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n",
		"Establish and fund a new lightning channel with the given peer.",
		"The capacity is the amount of satoshi we insert into the channel,",
		"and initialSend is the amount we initially hand over to the other party.",
		"data is an optional field that can contain 32 bytes of hex to send as part of the channel fund",
	),
	ShortDescription: "Establish and fund a new lightning channel with the given peer.\n",
}

var dualFundCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dualfund"),
		lnutil.ReqColor("subcommand"), lnutil.OptColor("parameters...")),
	Description: fmt.Sprintf("%s\n%s\n",
		"Commands for establishing and mutually funding a new lightning channel with the given peer.",
		"Subcommands: start, accept, decline"),

	ShortDescription: "Commands for dual funding\n",
}

var dualFundStartCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dualfund start"),
		lnutil.ReqColor("peer", "coinType", "ourAmount", "theirAmount")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Establish and mutually fund a new lightning channel with the given peer.",
		"The capacity is the sum of the amounts both peers insert into the channel,",
		"each party will end up with their funded amount in the channel."),
	ShortDescription: "Establish and mutually fund a new lightning channel with the given peer.\n",
}

var dualFundDeclineCommand = &Command{
	Format:           fmt.Sprintf("%s\n", lnutil.White("dualfund decline")),
	Description:      "Declines the pending dual funding request received from another peer (if any)\n",
	ShortDescription: "Declines the pending dual funding request received from another peer (if any)\n",
}

var dualFundAcceptCommand = &Command{
	Format:           fmt.Sprintf("%s\n", lnutil.White("dualfund accept")),
	Description:      "Accepts the pending dual funding request received from another peer (if any)\n",
	ShortDescription: "Accepts the pending dual funding request received from another peer (if any)\n",
}

var watchCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("watch"),
		lnutil.ReqColor("channel idx", "watchPeerIdx")),
	Description: fmt.Sprintf("%s\n%s\n",
		"Send channel data to a watcher",
		"The watcher can defend your channel while you're offline."),
	ShortDescription: "Send channel watch data to watcher.\n",
}

var pushCommand = &Command{
	Format: fmt.Sprintf("%s%s%s%s\n", lnutil.White("push"), lnutil.ReqColor("channel idx", "amount"), lnutil.OptColor("times"), lnutil.OptColor("data")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Push the given amount (in satoshis) to the other party on the given channel.",
		"Optionally, the push operation can be associated with a 32 byte value hex encoded.",
		"Optionally, the push operation can be repeated <times> number of times."),
	ShortDescription: "Push the given amount (in satoshis) to the other party on the given channel.\n",
}

var closeCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("close"), lnutil.ReqColor("channel idx")),
	Description: fmt.Sprintf("%s\n%s\n%s%s\n",
		"Cooperatively close the channel with the given index by asking",
		"the other party to finalize the channel pay-out.",
		"See also: ", lnutil.White("break")),
	ShortDescription: "Cooperatively close the channel with the given index by asking\n",
}

var breakCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("break"), lnutil.ReqColor("channel idx")),
	Description: fmt.Sprintf("%s\n%s\n%s%s\n",
		"Forcibly break the given channel. Note that you need to wait",
		"a set number of blocks before you can use the money.",
		"See also: ", lnutil.White("stop")),
	ShortDescription: "Forcibly break the given channel.\n",
}

var historyCommand = &Command{
	Format:           lnutil.White("history"),
	Description:      "Show all the metadata for justice txs",
	ShortDescription: "Show all the metadata for justice txs.\n",
}

var addHTLCCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("add"), lnutil.ReqColor("channel idx", "amount", "locktime", "RHash"), lnutil.OptColor("data")),
	Description: fmt.Sprintf("%s\n%s\n",
		"Add an HTLC of the given amount (in satoshis) to the given channel. Locktime specifies the number of blocks the HTLC stays active before timing out",
		"Optionally, the push operation can be associated with a 32 byte value hex encoded."),
	ShortDescription: "Add HTLC of the given amount (in satoshis) to the given channel.\n",
}

var clearHTLCCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("clear"), lnutil.ReqColor("channel idx", "HTLC idx", "R"), lnutil.OptColor("data")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Clear an HTLC of the given index from the given channel.",
		"Optionally, the push operation can be associated with a 32 byte value hex encoded.",
		"Set R to zero to timeout the HTLC"),
	ShortDescription: "Clear HTLC of the given index from the given channel.\n",
}

var claimHTLCCommand = &Command{
	Format:           fmt.Sprintf("%s%s\n", lnutil.White("claim"), lnutil.ReqColor("R")),
	Description:      "Claim any on-chain HTLC that matches the given preimage. Use this to claim an HTLC after the channel is broken.\n",
	ShortDescription: "Clear HTLC of the given index from the given channel.\n",
}

func (lc *litAfClient) History(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, historyCommand.Format)
		fmt.Fprintf(color.Output, historyCommand.Description)
		return nil
	}

	args := new(litrpc.StateDumpArgs)
	reply := new(litrpc.StateDumpReply)

	err := lc.Call("LitRPC.StateDump", args, reply)
	if err != nil {
		return err
	}

	for _, tx := range reply.Txs {
		fmt.Fprintf(color.Output, "Pkh: %x, Idx: %d, Sig: %x, Txid: %x, Data: %x, Amt: %d\n", tx.Pkh, tx.Idx, tx.Sig, tx.Txid, tx.Data, tx.Amt)
	}

	return nil
}

// CheckHelpCommand checks whether the user wants help regarding the command
// or passed invalid arguments. Also checks for expected length of command
// and returns and error if the expected length is different.
func CheckHelpCommand(command *Command, textArgs []string, expectedLength int) (bool, error) {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, command.Format)
		fmt.Fprintf(color.Output, command.Description)
		return true, nil // stop Execution if the guy just wants help
	}
	if len(textArgs) < expectedLength {
		// if number of args are less than expected, return
		return true, fmt.Errorf(command.Format) // stop execution in case of err
	}
	return false, nil
}

func (lc *litAfClient) FundChannel(textArgs []string) error {
	stopEx, err := CheckHelpCommand(fundCommand, textArgs, 4)
	if err != nil || stopEx {
		return err
	}
	args := new(litrpc.FundArgs)
	reply := new(litrpc.FundReply)

	peer, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	coinType, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}

	cCap, err := strconv.Atoi(textArgs[2])
	if err != nil {
		return err
	}
	iSend, err := strconv.Atoi(textArgs[3])
	if err != nil {
		return err
	}

	if len(textArgs) > 4 {
		data, err := hex.DecodeString(textArgs[4])
		if err != nil {
			// Wasn't valid hex, copy directly and truncate
			copy(args.Data[:], textArgs[3])
		} else {
			copy(args.Data[:], data[:])
		}
	}

	args.Peer = uint32(peer)
	args.CoinType = uint32(coinType)
	args.Capacity = int64(cCap)
	args.InitialSend = int64(iSend)

	err = lc.Call("LitRPC.FundChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "funded channel %d (height: %d)\n", reply.ChanIdx, reply.FundHeight)
	return nil
}

func (lc *litAfClient) DualFund(textArgs []string) error {
	stopEx, err := CheckHelpCommand(dualFundCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}
	return nil
}

// Mutually fund a channel
func (lc *litAfClient) DualFundChannel(textArgs []string) error {
	stopEx, err := CheckHelpCommand(dualFundStartCommand, textArgs, 4)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.DualFundArgs)
	reply := new(litrpc.StatusReply)

	peer, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	coinType, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}

	ourAmt, err := strconv.Atoi(textArgs[2])
	if err != nil {
		return err
	}

	theirAmt, err := strconv.Atoi(textArgs[3])
	if err != nil {
		return err
	}

	args.Peer = uint32(peer)
	args.CoinType = uint32(coinType)
	args.OurAmount = int64(ourAmt)
	args.TheirAmount = int64(theirAmt)

	err = lc.rpccon.Call("LitRPC.DualFundChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) dualFundRespond(aor bool) error {
	reply := new(litrpc.StatusReply)
	args := new(litrpc.DualFundRespondArgs)
	args.AcceptOrDecline = aor
	err := lc.rpccon.Call("LitRPC.DualFundRespond", args, reply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

// Decline mutual funding of a channel
func (lc *litAfClient) DualFundDecline(textArgs []string) error {
	stopEx, err := CheckHelpCommand(dualFundDeclineCommand, textArgs, 0)
	if err != nil || stopEx {
		return err
	}

	return lc.dualFundRespond(false)
}

// Accept mutual funding of a channel
func (lc *litAfClient) DualFundAccept(textArgs []string) error {
	stopEx, err := CheckHelpCommand(dualFundAcceptCommand, textArgs, 0)
	if err != nil || stopEx {
		return err
	}

	return lc.dualFundRespond(true)
}

// Request close of a channel.  Need to pass in peer, channel index
func (lc *litAfClient) CloseChannel(textArgs []string) error {
	stopEx, err := CheckHelpCommand(closeCommand, textArgs, 1)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.ChanArgs)
	reply := new(litrpc.StatusReply)

	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	args.ChanIdx = uint32(cIdx)

	err = lc.Call("LitRPC.CloseChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

// Almost exactly the same as CloseChannel.  Maybe make "break" a bool...?
func (lc *litAfClient) BreakChannel(textArgs []string) error {
	stopEx, err := CheckHelpCommand(breakCommand, textArgs, 1)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.ChanArgs)
	reply := new(litrpc.StatusReply)

	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	args.ChanIdx = uint32(cIdx)

	err = lc.Call("LitRPC.BreakChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

// Push is the shell command which calls PushChannel
func (lc *litAfClient) Push(textArgs []string) error {
	stopEx, err := CheckHelpCommand(pushCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.PushArgs)
	reply := new(litrpc.PushReply)

	// this stuff is all the same as in cclose, should put into a function...
	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	amt, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}

	times := int(1)
	if len(textArgs) > 2 {
		times, err = strconv.Atoi(textArgs[2])
		if err != nil {
			return err
		}
	}

	if len(textArgs) > 3 {
		data, err := hex.DecodeString(textArgs[3])
		if err != nil {
			// Wasn't valid hex, copy directly and truncate
			copy(args.Data[:], textArgs[3])
		} else {
			copy(args.Data[:], data[:])
		}
	}

	args.ChanIdx = uint32(cIdx)
	args.Amt = int64(amt)

	for times > 0 {
		err := lc.Call("LitRPC.Push", args, reply)
		if err != nil {
			return err
		}
		fmt.Fprintf(color.Output, "Pushed %s at state %s\n", lnutil.SatoshiColor(int64(amt)), lnutil.White(reply.StateIndex))
		times--
	}

	return nil
}

func (lc *litAfClient) Dump(textArgs []string) error {
	pReply := new(litrpc.DumpReply)
	pArgs := new(litrpc.NoArgs)

	err := lc.Call("LitRPC.DumpPrivs", pArgs, pReply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, "Private keys for all channels and utxos:\n")

	// Display DumpPriv info
	for i, t := range pReply.Privs {
		fmt.Fprintf(color.Output, "%d %s h:%d amt:%s %s ",
			i, lnutil.OutPoint(t.OutPoint), t.Height,
			lnutil.SatoshiColor(t.Amt), t.CoinType)
		if t.Delay != 0 {
			fmt.Fprintf(color.Output, " delay: %d", t.Delay)
		}
		if !t.Witty {
			fmt.Fprintf(color.Output, " non-witness")
		}
		if len(t.PairKey) > 1 {
			fmt.Fprintf(
				color.Output, "\nPair Pubkey: %s", lnutil.Green(t.PairKey))
		}
		fmt.Fprintf(color.Output, "\n\tprivkey: %s", lnutil.Red(t.WIF))
		fmt.Fprintf(color.Output, "\n")
	}

	return nil
}

func (lc *litAfClient) Watch(textArgs []string) error {
	stopEx, err := CheckHelpCommand(watchCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.WatchArgs)
	reply := new(litrpc.WatchReply)

	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	peer, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}

	args.ChanIdx = uint32(cIdx)
	args.SendToPeer = uint32(peer)

	err = lc.Call("LitRPC.Watch", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "Send channel %d data to peer %d\n",
		args.ChanIdx, args.SendToPeer)

	return nil
}

// Add is the shell command which calls AddHTLC
func (lc *litAfClient) AddHTLC(textArgs []string) error {
	stopEx, err := CheckHelpCommand(addHTLCCommand, textArgs, 3)

	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.AddHTLCArgs)
	reply := new(litrpc.AddHTLCReply)

	// this stuff is all the same as in cclose, should put into a function...
	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	amt, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	locktime, err := strconv.Atoi(textArgs[2])
	if err != nil {
		return err
	}

	RHash, err := hex.DecodeString(textArgs[3])
	if err != nil {
		return err
	}
	copy(args.RHash[:], RHash[:])

	if len(textArgs) > 4 {
		data, err := hex.DecodeString(textArgs[4])
		if err != nil {
			// Wasn't valid hex, copy directly and truncate
			copy(args.Data[:], textArgs[4])
		} else {
			copy(args.Data[:], data[:])
		}
	}

	args.ChanIdx = uint32(cIdx)
	args.Amt = int64(amt)
	args.LockTime = uint32(locktime)

	err = lc.Call("LitRPC.AddHTLC", args, reply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, "Added HTLC %s at state %s idx %s\n", lnutil.SatoshiColor(int64(amt)), lnutil.White(reply.StateIndex), lnutil.White(reply.HTLCIndex))

	return nil
}

// Clear is the shell command which calls ClearHTLC
func (lc *litAfClient) ClearHTLC(textArgs []string) error {
	stopEx, err := CheckHelpCommand(clearHTLCCommand, textArgs, 3)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.ClearHTLCArgs)
	reply := new(litrpc.ClearHTLCReply)

	// this stuff is all the same as in cclose, should put into a function...
	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	HTLCIdx, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}

	R, err := hex.DecodeString(textArgs[2])
	if err != nil {
		return err
	}
	copy(args.R[:], R[:])

	if len(textArgs) > 3 {
		data, err := hex.DecodeString(textArgs[3])
		if err != nil {
			// Wasn't valid hex, copy directly and truncate
			copy(args.Data[:], textArgs[3])
		} else {
			copy(args.Data[:], data[:])
		}
	}

	args.ChanIdx = uint32(cIdx)
	args.HTLCIdx = uint32(HTLCIdx)

	err = lc.Call("LitRPC.ClearHTLC", args, reply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, "Cleared HTLC %s at state %s\n", lnutil.White(HTLCIdx), lnutil.White(reply.StateIndex))

	return nil
}

// Clear is the shell command which calls ClearHTLC
func (lc *litAfClient) ClaimHTLC(textArgs []string) error {
	stopEx, err := CheckHelpCommand(claimHTLCCommand, textArgs, 1)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.ClaimHTLCArgs)
	reply := new(litrpc.TxidsReply)

	R, err := hex.DecodeString(textArgs[0])
	if err != nil {
		return err
	}
	copy(args.R[:], R[:])

	err = lc.Call("LitRPC.ClaimHTLC", args, reply)
	if err != nil {
		return err
	}
	for _, txid := range reply.Txids {
		fmt.Fprintf(color.Output, "Claimed HTLC with txid %s\n", lnutil.White(txid))
	}

	return nil
}
