package main

import (
	"fmt"
	"strconv"
	"github.com/fatih/color"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/litrpc"
)

// RequestAsync keeps requesting messages from the server.  The server blocks
// and will send a response once it gets one.  Once the rpc client receives a
// response, it will immediately request another.
func (lc *litAfClient) RequestAsync() {
	for {
		args := new(litrpc.NoArgs)
		reply := new(litrpc.StatusReply)

		err := lc.rpccon.Call("LitRPC.GetMessages", args, reply)
		if err != nil {
			fmt.Fprintf(color.Output,"RequestAsync error %s\n", lnutil.Red(err.Error()))
			break
			// end loop on first error.  it's probably a connection error

		}
		fmt.Fprintf(color.Output,"%s\n", reply.Status)
	}
}

// Lis starts listening.  Takes args of port to listen on.
func (lc *litAfClient) Lis(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output,"%s%s\n", lnutil.White("lis"), lnutil.OptColor("port"))
		fmt.Fprintf(color.Output,"Start listening for incoming connections.\n")
		fmt.Fprintf(color.Output,"The port number, if omitted, defaults to 2448.\n")
		return nil
	}

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

	fmt.Fprintf(color.Output,"%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) Connect(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output,"%s <%s>@<%s>[:<%s>]\n", lnutil.White("con"), lnutil.White("pubkeyhash"), lnutil.White("hostname"), lnutil.White("port"))
		fmt.Fprintf(color.Output,"Make a connection to another host by connecting to their pubkeyhash\n")
		fmt.Fprintf(color.Output,"(printed when listening using the lis command), on the given host.\n")
		fmt.Fprintf(color.Output,"A port may be provided; if omitted, 2448 is used.\n")
		return nil
	}

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

	fmt.Fprintf(color.Output,"%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) Say(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output,"%s%s\n", lnutil.White("say"), lnutil.ReqColor("peer", "message"))
		fmt.Fprintf(color.Output,"Send a message to a peer.\n")
		return nil
	}

	args := new(litrpc.SayArgs)
	reply := new(litrpc.StatusReply)

	if len(textArgs) < 2 {
		return fmt.Errorf("usage: say peerNum message")
	}

	peerIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	textArgs = textArgs[1:]

	for _, s := range textArgs {
		args.Message += s + " "
	}

	args.Peer = uint32(peerIdx)

	err = lc.rpccon.Call("LitRPC.Say", args, reply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output,"%s\n", reply.Status)
	return nil
}
