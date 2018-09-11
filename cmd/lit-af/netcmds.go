package main

import (
	"encoding/hex"
	"fmt"
	"strconv"

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
	Format:           fmt.Sprintf("%s%s%s%s\n", lnutil.White("lis"), lnutil.OptColor("port"), lnutil.OptColor("short"), lnutil.OptColor("powBytes")),
	Description:      fmt.Sprintf("%s\n%s\n%s%s\n",
		"Start listening for incoming connections",
		"The port number, if omitted, defaults to 2448",
		"Short addresses requires an argument (powbytes) and does sufficient ",
		"computation to achieve a lower difficulty than the target specified"),
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

// Request remote control access
var rcRequestCommand = &Command{
	Format:           fmt.Sprintf("%s\n", lnutil.White("rcreq")),
	Description:      "If this lit-af key has not been authorized on the server, this command will let the server know you want access. Another client that is privileged can authorize you then.\n",
	ShortDescription: "Requests remote control authorization",
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
						fmt.Printf("generating short listen address with %d pow bytes\n", args.ShortZeros)
					} else {
						return fmt.Errorf("please specify how much work in bytes you want to perform")
					}
				}
			}
		} else {
			// user input: lis <portNumber> blah
			// check whether blah refers to short addresses with / without port number
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
						fmt.Printf("generating short listen address with %d pow bytes", args.ShortZeros)
					} else {
						return fmt.Errorf("please specify how much work in bytes you want to perform")
					}
				}
			}
		}
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

func (lc *litAfClient) RemoteControlRequest(textArgs []string) error {
	stopEx, err := CheckHelpCommand(rcRequestCommand, textArgs, 0)
	if err != nil || stopEx {
		return err
	}
	args := new(litrpc.RCRequestAuthArgs)
	reply := new(litrpc.StatusReply)

	// No need to fill it in, since this will be handled
	// by the RPC server (inserts the pubkey from the lndc connection)

	err = lc.Call("LitRPC.RequestRemoteControlAuthorization", args, reply)
	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}
