package main

import (
	"html/template"
	"io"
	"os"
)

type Command struct {
	Name        string
	Description string
}

type Group struct {
	Name     string
	Commands []Command
}

type Document struct {
	Command     *Command
	Groups      []Group
	Description string
}

func main() {
	var groups = []Group{
		Group{
			Name: "Lit Node",
			Commands: []Command{
				{
					Name:        "tn3",
					Description: "Usage: \n./lit --tn3 <ip addr of node>\n\nDescription: Connect to bitcoin testnet3.\n\nYou can specify yes, y, true, 1, ok, enable, on (or even yup!) to connect to a testnet3 host from the default seeds (same as Bitcoin Core) automatically.",
				},
				{
					Name:        "bc2",
					Description: "Description: bc2 full node.",
				},

				{
					Name:        "lt4",
					Description: "Usage: \n./lit --lt4\n\nDescription: Connect to litecoin testnet4",
				},

				{
					Name:        "reg",
					Description: "Usage: \n./lit --tower \n\nDescription: Connect to bitcoin regtest\n\nUsage: \n./lit --reg <ip addr of node / localhost:port>",
				},

				{
					Name:        "litereg",
					Description: "Long: litereg\n\nUsage: \n./lit --litereg <ip addr of node / localhost:port>\n\nDescription: Connect to litecoin regtest.\n\n",
				},

				{
					Name:        "tvtc",
					Description: "Long: tvtc\n\nUsage: \n./lit --reg <ip addr of node / localhost:port>\n\nDescription: Connect to Vertcoin test node.\n\n",
				},

				{
					Name:        "vtc",
					Description: "Long: vtc\n\nUsage: \n./lit --tn3 <ip addr of node>\n\nDescription: Connect to Vertcoin. You can specify yes, y, true, 1, ok, enable, on (or even yup!) to connect to a Vertcoin host from the default seeds (same as Vertcoin Core) automatically",
				},

				{
					Name:        "dir",
					Description: "Long: dir\n\nUsage: \n ./lit --dir=<absolute path dir> \n\nDescription: Specify Home Directory of lit as an absolute path.",
				},

				{
					Name:        "tracker",
					Description: "Long: tracker\n\nUsage: \n./lit --tracker <ip of tracker>\n\nDescription: LN address tracker URL http|https://host:port",
				},

				{
					Name:        "reSync",
					Description: "Long: reSync \nShort: r\n\nUsage: \n./lit --reSync <tip Number> \n\nDescription: Resync from the given tip.",
				},

				{
					Name:        "tower",
					Description: "Long: tower\n\n Usage: \n./lit --tower \n\nDescription: Watchtower: Run a watching node",
				},

				{
					Name:        "hard",
					Description: "Long: hard \nShort: t\n\nUsage: \n./lit --hard \n\nDescription: Flag to set networks",
				},

				{
					Name:        "verbose",
					Description: "Long: verbose \nShort: v\n\nUsage: \n./lit -v \n\nDescription: Set verbosity to true.",
				},

				{
					Name:        "rpcport",
					Description: "Long: rpcport \nShort: p\n\nUsage: \n./lit -p 8001 \n\nDescription: Set RPC port to connect to specified port",
				},
			},
		},
		Group{
			Name: "Lit Advanced Functionality Client",
			Commands: []Command{
				{
					Name:        "dir",
					Description: "Usage: \n./lit-af -dir <path to dir>\n\nDescription: \ndirectory to save settings",
				},
				{
					Name:        "node",
					Description: "Usage: \n./lit-af -node <ip addr of node> \n\nDescription:\nhost to connect to (default 127.0.0.1)",
				},

				{
					Name:        "p",
					Description: "Usage: \n./lit-af -p <port no>\n\nDescription:\nport to connect to (default 8001)",
				},

				{
					Name: "help",
					Description: "Usage: \nhelp [<command>]\n\nDescription:	\nShow information about a given command",
				},

				{
					Name: "say",
					Description: "Long: litereg\n\nUsage: say <peer> <message>\n\nDescription: 	\nSend a message to a peer.",
				},

				{
					Name: "ls",
					Description: "Long: tvtc\n\nUsage: \n./lit --reg <ip addr of node / localhost:port>\n\nDescription:	\nShow various information about our current state",
				},

				{
					Name:        "adr",
					Description: "Long: vtc\n\nUsage: \nadr <?amount> <?cointype>\n\nDescription: \nMakes new addresses",
				},

				{
					Name: "send",
					Description: "Long: dir\n\nUsage: \nsend <address> <amount> \n\nDescription: 	\nSend the given amount of satoshis to the given address.",
				},

				{
					Name:        "fan",
					Description: "Long: tracker\n\nUsage: \nfan <addr> <howmany> <howmuch>\n\nDescription:\n",
				},

				{
					Name: "sweep",
					Description: "Long: reSync \nShort: r\n\nUsage: \nsweep <addr> <howmany> [<drop>]\n\nDescription:	\nMove UTXOs with many 1-in-1-out txs.",
				},

				{
					Name: "lis",
					Description: "Long: tower\n\n Usage: \nlis [<port>]\n\nDescription:	\nStart listening for incoming connections.",
				},

				{
					Name: "con",
					Description: "Long: hard \nShort: t\n\nUsage: \ncon <pubkeyhash>@<hostname>[:<port>]\n\nDescription:	\nMake a connection to another host by connecting to their pubkeyhash",
				},

				{
					Name: "graph",
					Description: "Long: verbose \nShort: v\n\nUsage: \ngraph\n\nDescription:	\nShows the channel map",
				},

				{
					Name: "dlc",
					Description: "Long: rpcport \nShort: p\n\nUsage: \ndlc <subcommand> [<parameters...>]\n\nDescription:	\nCommand for working with Discreet Log Contracts.",
				},
				{
					Name: "fund",
					Description: "Long: verbose \nShort: v\n\nUsage: \nfund <peer> <coinType> <capacity> <initialSend> [<data>]\n\nDescription:	\nEstablish and fund a new lightning channel with the given peer.",
				},
				{
					Name: "push",
					Description: "Long: verbose \nShort: v\n\nUsage: \npush <channel idx> <amount> [<times>] [<data>]\n\nDescription:	\nPush the given amount (in satoshis) to the other party on the given channel.",
				},
				{
					Name: "close",
					Description: "Long: verbose \nShort: v\n\nUsage: \nclose <channel idx>\n\nDescription:	\nCooperatively close the channel with the given index by asking",
				},
				{
					Name: "break",
					Description: "Long: verbose \nShort: v\n\nUsage: \nbreak <channel idx>\n\nDescription: 	\nForcibly break the given channel.",
				},
				{
					Name:        "history",
					Description: "Long: verbose \nShort: v\n\nUsage: \nhistory\n\nDescription: \nShow all the metadata for justice txs.",
				},
				{
					Name:        "off",
					Description: "Long: verbose \nShort: v\n\nUsage: \noff\n\nDescription: \nShut down the lit node.",
				},
				{
					Name:        "exit",
					Description: "Long: verbose \nShort: v\n\nUsage: \nexit\n\nDescription: \nExit the interactive shell.",
				},
			},
		},
		Group{
			Name: "Discreet Log Contracts",
			Commands: []Command{
				{
					Name:        "dlc oracle",
					Description: "Usage: \nlit-af# dlc oracle <command>\n\nDescription: Command for managing and interfacing with dlc oracles.\n",
				},
				{
					Name:        "dlc oracle add",
					Description: "Usage: \nlit-af# dlc oracle add <pubkey>\n\nDescription: Adds a new oracle by manually providing the pubkey\n",
				},
				{
					Name:        "dlc oracle import",
					Description: "Usage: \nlit-af# dlc oracle import <URL>\n\nDescription: Imports a new oracle using a URL to its REST interface\n",
				},
				{
					Name:        "dlc oracle ls",
					Description: "Usage: \nlit-af# dlc oracle ls\n\nDescription: Shows a list of known oracles\n",
				},
				{
					Name:        "",
					Description: "",
				},
				{
					Name:        "dlc contract",
					Description: "Usage: \nlit-af# dlc contract <subcommand> [<parameters...>]\n\nDescription: \n",
				},
				{
					Name:        "dlc contract new",
					Description: "Usage: \nlit-af# dlc contract new\n\nDescription: Creates a new, empty draft dlc contract\n",
				},
				{
					Name:        "dlc contract view",
					Description: "Usage: \nlit-af# dlc contract view <id>\n\nDescription: \nView contract #id",
				},
				{
					Name:        "dlc contract viewpayout",
					Description: "Usage: \nlit-af# dlc contract viewpayout <id> <start> <end> <increment>\n\nDescription: \nViews the payout table of a contract",
				},
				{
					Name:        "dlc contract setoracle",
					Description: "Usage: \nlit-af# dlc contract setoracle <cid> <oid> \n\ncid The ID of the contract\noid The ID of the oracle\n\nDescription: \nConfigures a contract for using a specific oracle",
				},
				{
					Name:        "dlc contract settime",
					Description: "Usage: \nlit-af# dlc contract settime <cid> <time>\n\ncid The ID of the contract\ntime The settlement time (unix timestamp)\n\nDescription: \nSets the settlement time for the contract",
				},
				{
					Name:        "dlc contract setdatafeed",
					Description: "Usage: \nlit-af# dlc contract setdatafeed <cid> <feed>\n\ncid The ID of the contract\nfeed The ID of the data feed (provided by the oracle)\n\nDescription: \nSets the data feed to use for the contract",
				},
				{
					Name:        "dlc contract setrpoint",
					Description: "Usage: \nlit-af# dlc contract setrpoint <cid> <rpoint>\n\ncid The ID of the contract\nrpoint The Rpoint of the publication to use (33 byte in hex)\n\nDescription: \nSets the R point to use for the contract",
				},
				{
					Name:        "dlc contract setfunding",
					Description: "Usage: \nlit-af# dlc contract setfunding <cid> <ourAmount> <theirAmount>\n\ncid The ID of the contract\nourAmount The amount we will fund\ntheirAmount The amount our peer will fund\n\nDescription: \nSets the amounts both parties in the contract will fund",
				},
				{
					Name:        "dlc contract setdivision",
					Description: "Usage: \nlit-af# dlc contract setdivision <cid> <valueAllForUs> <valueAllForThem>\n\ncid The ID of the contract\nvalueAllForUs The outcome with which we will be entitled to the full contract value\nvalueAllForThem The outcome with which our peer will be entitled to the full contract value\n\nDescription: \nSets the values of the oracle data that will result in the full contract funds being paid to either peer",
				},
				{
					Name:        "dlc contract setcointype",
					Description: "Usage: \nlit-af# dlc contract setcointype <cid> <cointype>\n\ncid The ID of the contract\ncointype The ID of the coin type to use for the contract\n\nDescription: \nSets the values of the oracle data that will result in the full contract funds being paid to either peer",
				},
				{
					Name:        "dlc contract offer",
					Description: "Usage: \nlit-af# dlc contract offer <cid> <peer>\n\ncid The ID of the contract\ncointype The ID of the peer to offer the contract to\n\nDescription: \nOffers a contract to one of your peers. Before offering a contract, you should set the other parameters such as oracle, time, datafeed, rpoint, funding, etc usign the appropriate commands",
				},
				{
					Name:        "dlc contract decline",
					Description: "Usage: \nlit-af# dlc contract decline <cid>\n\ncid The ID of the contract to decline\n\nDescription: \nDeclines a contract offered to you",
				},
				{
					Name:        "dlc contract settle",
					Description: "Usage: \nlit-af# dlc contract settle <cid> <oracleValue> <oracleSig>\n\ncid The ID of the contract\noracleValue The value the oracle published\noracleSig The signature from the oracle\n\nDescription: \nSettles the contract based on a value and signature from the oracle",
				},
				{
					Name:        "dlc contract ls",
					Description: "Usage: \nlit-af# dlc contract ls\n\nDescription: \nLists all contracts (draft / open / declined)",
				},
			},
		},
	}

	tmpl := template.Must(template.ParseFiles("template.html"))
	// Use this if you want to publish to a separate directory
	// if _, err := os.Stat("litrpc"); os.IsNotExist(err) {
		// e := os.Mkdir("litrpc", 0775)
		// if e != nil {
		// 	panic(e)
		// }
	// }
	tmpl.Execute(open("./index.html"), Document{ // opens $PWD for now
		Command: nil,
		Groups:  groups,
	})

	for _, group := range groups {
		for _, command := range group.Commands {
			tmpl.Execute(open("./"+command.Name+".html"), Document{
				Command:     &command,
				Groups:      groups,
				Description: command.Description,
			})
		}
	}
}

func open(path string) io.Writer {
	f, err := os.Create(path)
	// not closing, program will close sooner
	if err != nil {
		panic(err)
	}
	return f
}
