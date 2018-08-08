package main

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/mit-dci/lit/qln"

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

var rcAuthCommand = &Command{
	Format:           fmt.Sprintf("%s %s\n", lnutil.White("rcauth"), lnutil.ReqColor("pub", "auth (true|false)")),
	Description:      "Authorizes a remote peer by pubkey for sending remote control commands over LNDC\n",
	ShortDescription: "Manages authorization for remote control",
}

// Temporary command for testing. Remove when there's an actual remote control library.
var rcSendCommand = &Command{
	Format:           fmt.Sprintf("%s %s\n", lnutil.White("rcsend"), lnutil.ReqColor("peer", "msg")),
	Description:      "Sends a remote control message to peer [peer]. [msg] is hex encoded\n",
	ShortDescription: "Sends a remote control message",
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
	if len(textArgs) > 0 {
		if strings.Contains(textArgs[0], ":") {
			args.Port = textArgs[0]
		} else {
			args.Port = ":" + textArgs[0]
		}
	}

	err := lc.Call("LitRPC.Listen", args, reply)
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

func (lc *litAfClient) RemoteControlAuth(textArgs []string) error {
	stopEx, err := CheckHelpCommand(rcAuthCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}
	args := new(litrpc.RCAuthArgs)
	reply := new(litrpc.StatusReply)

	args.PubKey, err = hex.DecodeString(textArgs[0])
	if err != nil {
		return fmt.Errorf("Could not decode pubkey: %s", err.Error())
	}

	args.Authorization = new(qln.RemoteControlAuthorization)
	args.Authorization.Allowed = lnutil.YupString(textArgs[1])

	err = lc.Call("LitRPC.RemoteControlAuth", args, reply)
	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

func (lc *litAfClient) RemoteControlSend(textArgs []string) error {
	stopEx, err := CheckHelpCommand(rcAuthCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}
	args := new(litrpc.RCSendArgs)
	reply := new(litrpc.StatusReply)

	peerIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	args.PeerIdx = uint32(peerIdx)

	args.Msg, err = hex.DecodeString(textArgs[1])
	if err != nil {
		return fmt.Errorf("Could not decode message: %s", err.Error())
	}

	err = lc.Call("LitRPC.RemoteControlSend", args, reply)
	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}
