package main

import (
	"fmt"
	"strconv"

	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/litrpc"
)

func (lc *litAfClient) FundChannel(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Printf("%s%s\n", lnutil.White("fund"), lnutil.ReqColor("peer", "capacity", "initialSend"))
		fmt.Printf("Establish and fund a new lightning channel with the given peer.\n")
		fmt.Printf("The capacity is the amount of satoshi we insert into the channel,\n")
		fmt.Printf("and initialSend is the amount we initially hand over to the other party.\n")
		return nil
	}

	args := new(litrpc.FundArgs)
	reply := new(litrpc.StatusReply)

	if len(textArgs) < 3 {
		return fmt.Errorf("need args: fund peer capacity initialSend")
	}

	peer, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	cCap, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	iSend, err := strconv.Atoi(textArgs[2])
	if err != nil {
		return err
	}
	args.Peer = uint32(peer)
	args.Capacity = int64(cCap)
	args.InitialSend = int64(iSend)

	err = lc.rpccon.Call("LitRPC.FundChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", reply.Status)
	return nil
}

// Request close of a channel.  Need to pass in peer, channel index
func (lc *litAfClient) CloseChannel(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Printf("%s%s\n", lnutil.White("close"), lnutil.ReqColor("channel idx"))
		fmt.Printf("Cooperatively close the channel with the given index by asking\n")
		fmt.Printf("the other party to finalize the channel pay-out.\n")
		fmt.Printf("See also: break\n")
		return nil
	}

	args := new(litrpc.ChanArgs)
	reply := new(litrpc.StatusReply)

	// need args, fail
	if len(textArgs) < 1 {
		return fmt.Errorf("need args: close chanIdx")
	}

	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	args.ChanIdx = uint32(cIdx)

	err = lc.rpccon.Call("LitRPC.CloseChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", reply.Status)
	return nil
}

// Almost exactly the same as CloseChannel.  Maybe make "break" a bool...?
func (lc *litAfClient) BreakChannel(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Printf("%s%s\n", lnutil.White("break"), lnutil.ReqColor("channel idx"))
		fmt.Printf("Forcibly break the given channel. Note that we need to wait\n")
		fmt.Printf("a set number of blocks before we can use the money.\n")
		fmt.Printf("See also: stop\n")
		return nil
	}

	args := new(litrpc.ChanArgs)
	reply := new(litrpc.StatusReply)

	// need args, fail
	if len(textArgs) < 1 {
		return fmt.Errorf("need args: break chanIdx")
	}

	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	args.ChanIdx = uint32(cIdx)

	err = lc.rpccon.Call("LitRPC.BreakChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", reply.Status)
	return nil
}

// Push is the shell command which calls PushChannel
func (lc *litAfClient) Push(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Printf("%s%s%s\n", lnutil.White("push"), lnutil.ReqColor("channel idx", "amount"), lnutil.OptColor("times"))
		fmt.Printf("Push the given amount (in satoshis) to the other party on the given channel.\n")
		fmt.Printf("Optionally, the push operation can be repeated <times> number of times.\n")
		return nil
	}

	args := new(litrpc.PushArgs)
	reply := new(litrpc.PushReply)

	if len(textArgs) < 2 {
		return fmt.Errorf("need args: push chanIdx amt (times)")
	}

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

	args.ChanIdx = uint32(cIdx)
	args.Amt = int64(amt)

	for times > 0 {
		err := lc.rpccon.Call("LitRPC.Push", args, reply)
		if err != nil {
			return err
		}
		fmt.Printf("Pushed %s at state %s\n", lnutil.SatoshiColor(int64(amt)), lnutil.White(reply.StateIndex))
		times--
	}

	return nil
}
