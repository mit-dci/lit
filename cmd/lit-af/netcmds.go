package main

import (
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var sayCommand = &Command{
	Format:           fmt.Sprintf("%s%s\n", lnutil.White("say"), lnutil.ReqColor("peer", "message")),
	Description:      "Send a message to a peer.\n",
	ShortDescription: "Send a message to a peer.\n",
}

var lisCommand = &Command{
	Format:           fmt.Sprintf("%s%s\n", lnutil.White("lis"), lnutil.OptColor("port")),
	Description:      fmt.Sprintf("Start listening for incoming connections. The port number, if omitted, defaults to 2448.\n"),
	ShortDescription: "Start listening for incoming connections.\n",
}

var conCommand = &Command{
	Format: fmt.Sprintf("%s <%s>@<%s>[:<%s>]\n", lnutil.White("con"), lnutil.White("pubkeyhash"), lnutil.White("hostname"), lnutil.White("port")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Make a connection to another host by connecting to their pubkeyhash",
		"(printed when listening using the lis command), on the given host.",
		"A port may be provided; if omitted, 2448 is used."),
	ShortDescription: "Make a connection to another host by connecting to their pubkeyhash\n",
}

var graphCommand = &Command{
	Format:           fmt.Sprintf("%s\n", lnutil.White("graph")),
	Description:      fmt.Sprintf("Dump the channel graph in graphviz DOT format\n"),
	ShortDescription: "Shows the channel map\n",
}

// graph gets the channel map
func (lc *litAfClient) Graph(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, graphCommand.Format)
		fmt.Fprintf(color.Output, graphCommand.Description)
		return nil
	}

	args := new(litrpc.NoArgs)
	reply := new(litrpc.ChannelGraphReply)

	err := lc.Call("LitRPC.GetChannelMap", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Graph)

	return nil
}

// RequestAsync keeps requesting messages from the server.  The server blocks
// and will send a response once it gets one.  Once the rpc client receives a
// response, it will immediately request another.
func (lc *litAfClient) RequestAsync() {
	for {
		args := new(litrpc.NoArgs)
		reply := new(litrpc.StatusReply)

		err := lc.rpccon.Call("LitRPC.GetMessages", args, reply)
		if err != nil {
			fmt.Fprintf(color.Output, "RequestAsync error %s\n", lnutil.Red(err.Error()))
			break
			// end loop on first error.  it's probably a connection error

		}
		fmt.Fprintf(color.Output, "%s\n", reply.Status)
	}
}

// Lis starts listening.  Takes args of port to listen on.
func (lc *litAfClient) Lis(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, lisCommand.Format)
		fmt.Fprintf(color.Output, lisCommand.Description)
		return nil
	}

	args := new(litrpc.ListenArgs)
	reply := new(litrpc.ListeningPortsReply)

	args.Port = ":2448"
	if len(textArgs) >= 1 {
		// check whether the argument is a port
		_, err := strconv.Atoi(textArgs[0])
		if err != nil {
			// assume this error is due to the arg is not an int
			fmt.Println("expected")
			if len(textArgs) > 1 {
				if textArgs[0] == "short" {
					args.Short = true
					if len(textArgs) == 2 {
						shortzeros, err := strconv.Atoi(textArgs[1])
						if err != nil {
							fmt.Println("failed to activate short listen address mode")
							return fmt.Errorf("%s", err)
						}
						args.ShortZeros = uint8(shortzeros)
						fmt.Println("activating short listen address mode with zeros:", args.ShortZeros)
					} else {
						return fmt.Errorf("please specify how much work (in zeros of hashes) to perform")
					}
				}
			}
		} else {
			// user input: lis <portNumber> blah
			// check whether blah refers to short / vanity addresses
			args.Port = ":" + textArgs[0]
			if len(textArgs) > 1 {
				if textArgs[1] == "short" {
					args.Short = true
					if len(textArgs) == 3 {
						shortzeros, err := strconv.Atoi(textArgs[2])
						if err != nil {
							fmt.Println("failed to activate short listen address mode")
							return fmt.Errorf("%s", err)
						}
						args.ShortZeros = uint8(shortzeros)
						fmt.Println("activating short listen address mode with zeros:", args.ShortZeros)
					} else {
						return fmt.Errorf("please specify how much work (in zeros of hashes) to perform")
					}
				}
			}
		}
	}
	type ListenArgs struct {
		Port       string
		Short      bool
		ShortZeros uint8
	}

	err := lc.Call("LitRPC.Listen", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "listening on %s@%s\n", reply.Adr, args.Port)

	return nil
}

func (lc *litAfClient) Connect(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, conCommand.Format)
		fmt.Fprintf(color.Output, conCommand.Description)
		return nil
	}

	args := new(litrpc.ConnectArgs)
	reply := new(litrpc.StatusReply)

	if len(textArgs) == 0 {
		return fmt.Errorf("need: con pubkeyhash@hostname:port")
	}

	args.LNAddr = textArgs[0]

	err := lc.Call("LitRPC.Connect", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) Say(textArgs []string) error {
	stopEx, err := CheckHelpCommand(sayCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.SayArgs)
	reply := new(litrpc.StatusReply)

	peerIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	textArgs = textArgs[1:]

	for _, s := range textArgs {
		args.Message += s + " "
	}

	args.Peer = uint32(peerIdx)

	err = lc.Call("LitRPC.Say", args, reply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}
