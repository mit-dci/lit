package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
	if len(textArgs) > 0 {
		if strings.Contains(textArgs[0], ":") {
			args.Port = textArgs[0]
		} else {
			args.Port = ":" + textArgs[0]
		}
	}

	err := lc.rpccon.Call("LitRPC.Listen", args, reply)
	if err != nil {
		return err
	}
	if len(reply.LisIpPorts) == 0 {
		return fmt.Errorf("no listening port returned")
	}

	fmt.Fprintf(color.Output, "listening on %s@%s\n", reply.Adr, reply.LisIpPorts[0])

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

	err := lc.rpccon.Call("LitRPC.Connect", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) CheckChatMessages() {
	var err error

	// the ticker will activate approximately every half second
	chatTicker := time.NewTicker(time.Nanosecond * 500000000)

	for {
		select {
		case <-chatTicker.C:
			args := new(litrpc.NoArgs)
			reply := new(litrpc.CheckChatMessagesReply)

			err = lc.rpccon.Call("LitRPC.CheckChatMessages", args, reply)
			if err != nil {
				fmt.Fprintf(color.Output, "CheckChatMessages error %s\n", lnutil.Red(err.Error()))
				break
			}

			if reply.Status {
				chat := new(lnutil.ChatMsg)
				err = lc.rpccon.Call("LitRPC.GetChatMessage", args, chat)

				if err != nil {
					fmt.Fprintf(color.Output, "GetChatMessage error %s\n", lnutil.Red(err.Error()))
					break
				}

				fmt.Fprintf(color.Output,
					"\nmsg from %s: %s\n", lnutil.White(chat.PeerIdx), lnutil.Green(chat.Text))
			}
		}
	}
}

func (lc *litAfClient) Say(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, sayCommand.Format)
		fmt.Fprintf(color.Output, sayCommand.Description)
		return nil
	}

	args := new(litrpc.SayArgs)
	reply := new(litrpc.StatusReply)

	if len(textArgs) < 2 {
		return fmt.Errorf(sayCommand.Format)
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
	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}
