package main

import (
	"fmt"
	"strconv"

	"github.com/mit-dci/lit/litrpc"
)

func (lc *litAfClient) FundChannel(textArgs []string) error {
	args := new(litrpc.FundArgs)
	reply := new(litrpc.StatusReply)

	if len(textArgs) < 2 {
		return fmt.Errorf("need args: fund capacity initialSend")
	}

	cCap, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	iSend, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
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
	args := new(litrpc.CloseArgs)
	reply := new(litrpc.StatusReply)

	// need args, fail
	if len(textArgs) < 2 {
		return fmt.Errorf("need args: cclose peerIdx chanIdx")
	}
	peerIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	cIdx, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	args.PeerIdx = uint32(peerIdx)
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
	args := new(litrpc.CloseArgs)
	reply := new(litrpc.StatusReply)

	// need args, fail
	if len(textArgs) < 2 {
		return fmt.Errorf("need args: cclose peerIdx chanIdx")
	}
	peerIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	cIdx, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	args.PeerIdx = uint32(peerIdx)
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
	args := new(litrpc.PushArgs)
	reply := new(litrpc.PushReply)

	if len(textArgs) < 3 {
		return fmt.Errorf("need args: push peerIdx chanIdx amt (times)")
	}

	// this stuff is all the same as in cclose, should put into a function...
	peerIdx, err := strconv.ParseInt(textArgs[0], 10, 32)
	if err != nil {
		return err
	}
	cIdx, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	amt, err := strconv.Atoi(textArgs[2])
	if err != nil {
		return err
	}

	times := int(1)
	if len(textArgs) > 3 {
		times, err = strconv.Atoi(textArgs[3])
		if err != nil {
			return err
		}
	}

	args.PeerIdx = uint32(peerIdx)
	args.ChanIdx = uint32(cIdx)
	args.Amt = int64(amt)

	for times > 0 {
		err := lc.rpccon.Call("LitRPC.Push", args, reply)
		if err != nil {
			return err
		}
		fmt.Printf("Pushed %d new balance %d at state %d\n",
			amt, reply.MyAmt, reply.StateIndex)
		times--
	}

	return nil
}
