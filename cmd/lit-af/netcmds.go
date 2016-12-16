package main

import (
	"fmt"

	"github.com/mit-dci/lit/litrpc"
)

// Lis starts listening.  Takes args of port to listen on.
func (lc *litAfClient) Lis(textArgs []string) error {
	args := new(litrpc.ListenArgs)
	reply := new(litrpc.StatusReply)

	args.Port = ":2448"
	if len(textArgs) > 0 {
		args.Port = ":" + textArgs[0]
	}

	err := lc.rpccon.Call("LitRPC.Listen", args, reply)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) Connect(textArgs []string) error {
	args := new(litrpc.ConnectArgs)
	reply := new(litrpc.StatusReply)

	if len(textArgs) == 0 {
		return fmt.Errorf("need: con pubkeyhash@hostname:port")
	}

	args.LNAddr = textArgs[0]

	err := lc.rpccon.Call("LitRPC.Connect", args, reply)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) Say(textArgs []string) error {
	args := new(litrpc.SayArgs)
	reply := new(litrpc.StatusReply)

	if len(textArgs) < 1 {
		return fmt.Errorf("you have to say something")
	}

	for _, s := range textArgs {
		args.Message += s + " "
	}

	err := lc.rpccon.Call("LitRPC.Say", args, reply)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", reply.Status)
	return nil
}
