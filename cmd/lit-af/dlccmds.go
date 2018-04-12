package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var dlcCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc"),
		lnutil.ReqColor("subcommand"), lnutil.OptColor("parameters...")),
	Description: fmt.Sprintf("%s\n%s\n",
		"Command for working with discreet log contracts. Subcommand can be one of:",
		fmt.Sprintf("%-10s %s", lnutil.White("oracle"), "Command to manage oracles"),
	),
	ShortDescription: "Command for working with Discreet Log Contracts.\n",
}

var oracleCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc oracle"),
		lnutil.ReqColor("subcommand"), lnutil.OptColor("parameters...")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n",
		"Command for managing oracles. Subcommand can be one of:",
		fmt.Sprintf("%s\t%s", lnutil.White("add"), "Adds a new oracle by manually providing the pubkeys"),
		fmt.Sprintf("%s\t%s", lnutil.White("import"), "Imports a new oracle using a URL to its REST interface"),
		fmt.Sprintf("%s\t%s", lnutil.White("ls"), "Shows a list of known oracles"),
	),
	ShortDescription: "Manages oracles for the Discreet Log Contracts.\n",
}

var listOraclesCommand = &Command{
	Format:           fmt.Sprintf("%s\n", lnutil.White("dlc oracle ls")),
	Description:      "Shows a list of known oracles\n",
	ShortDescription: "Shows a list of known oracles\n",
}

func (lc *litAfClient) Dlc(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, dlcCommand.Format)
		fmt.Fprintf(color.Output, dlcCommand.Description)
		return nil
	}

	if len(textArgs) > 0 && textArgs[0] == "oracle" {
		return lc.Oracle(textArgs[1:])
	}

	return fmt.Errorf(dlcCommand.Format)
}

func (lc *litAfClient) Oracle(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, oracleCommand.Format)
		fmt.Fprintf(color.Output, oracleCommand.Description)
		return nil
	}

	if len(textArgs) > 0 && textArgs[0] == "ls" {
		return lc.ListOracles(textArgs[1:])
	}

	/*
		if len(textArgs) > 0 && textArgs[0] == "add" {
			return lc.AddOracle(textArgs[1:])
		}*/

	if len(textArgs) > 0 && textArgs[0] == "import" {
		return lc.ImportOracle(textArgs[1:])
	}

	return fmt.Errorf(oracleCommand.Format)
}

func (lc *litAfClient) ListOracles(textArgs []string) error {
	args := new(litrpc.ListOraclesArgs)
	reply := new(litrpc.ListOraclesReply)

	err := lc.rpccon.Call("LitRPC.ListOracles", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "System knows %d oracles\n", len(reply.Oracles))
	return nil
}

func (lc *litAfClient) ImportOracle(textArgs []string) error {
	args := new(litrpc.ImportOracleArgs)
	reply := new(litrpc.ImportOracleReply)

	args.Url = textArgs[0]
	args.Name = textArgs[1]

	err := lc.rpccon.Call("LitRPC.ImportOracle", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "Oracle succesfully registered under ID %d\n", reply.Oracle.Idx)
	return nil
}
