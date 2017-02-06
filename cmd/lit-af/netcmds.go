package main

import (
	"fmt"
	"strconv"

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
			fmt.Printf("RequestAsync error %s\n", err.Error())
			break
			// end loop on first error.  it's probably a connection error

		}
		fmt.Printf("%s\n", reply.Status)
	}
}

// Lis starts listening.  Takes args of port to listen on.
func (lc *litAfClient) Lis(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Printf("Syntax: lis [<port>]\n")
		fmt.Printf("Start listening for incoming connections.\n")
		fmt.Printf("The port number, if omitted, defaults to 2448.\n")
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

	fmt.Printf("%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) Connect(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Printf("Syntax: con <pubkeyhash>@<hostname>[:<port>]\n")
		fmt.Printf("Make a connection to another host by connecting to their pubkeyhash\n")
		fmt.Printf("(printed when listening using the lis command), on the given host.\n")
		fmt.Printf("A port may be provided; if omitted, 2448 is used.\n")
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

	fmt.Printf("%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) Say(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Printf("Syntax: say <peer> <message>\n")
		fmt.Printf("Send a message to a peer.\n")
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
	fmt.Printf("%s\n", reply.Status)
	return nil
}
