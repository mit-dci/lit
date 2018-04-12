package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/dlc"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var dlcCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc"),
		lnutil.ReqColor("subcommand"), lnutil.OptColor("parameters...")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Command for working with discreet log contracts. Subcommand can be one of:",
		fmt.Sprintf("%-10s %s", lnutil.White("oracle"), "Command to manage oracles"),
		fmt.Sprintf("%-10s %s", lnutil.White("contract"), "Command to manage contracts"),
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

var importOracleCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc oracle import"),
		lnutil.ReqColor("url"), lnutil.ReqColor("name")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Imports a new oracle using a URL to its REST interface",
		fmt.Sprintf("%s", lnutil.White("url"), "URL to the root of the publishes dlcoracle REST interface"),
		fmt.Sprintf("%s", lnutil.White("name"), "Name under which to register the oracle in LIT"),
	),
	ShortDescription: "Imports a new oracle into LIT from a REST interface\n",
}

var addOracleCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc oracle add"),
		lnutil.ReqColor("keys"), lnutil.ReqColor("name")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Adds a new oracle by entering the pubkeys manually",
		fmt.Sprintf("%s", lnutil.White("keys"), "Concatenated A,B and Q keys for the oracle"),
		fmt.Sprintf("%s", lnutil.White("name"), "Name under which to register the oracle in LIT"),
	),
	ShortDescription: "Adds a new oracle into LIT\n",
}

var contractCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc contract"),
		lnutil.ReqColor("subcommand"), lnutil.OptColor("parameters...")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n",
		"Command for managing contracts. Subcommand can be one of:",
		fmt.Sprintf("%-20s\t%s", lnutil.White("new"), "Adds a new draft contract"),
		fmt.Sprintf("%-20s\t%s", lnutil.White("view"), "Views a contract"),
		fmt.Sprintf("%-20s\t%s", lnutil.White("setoracle"), "Sets a contract to use a particular oracle"),
		fmt.Sprintf("%-20s\t%s", lnutil.White("setdatafeed"), "Sets the data feed to use for the contract"),
		fmt.Sprintf("%-20s\t%s", lnutil.White("settime"), "Sets the settlement time of a contract"),
		fmt.Sprintf("%-20s\t%s", lnutil.White("ls"), "Shows a list of known contracts"),
	),
	ShortDescription: "Manages oracles for the Discreet Log Contracts.\n",
}

var listContractsCommand = &Command{
	Format:           fmt.Sprintf("%s\n", lnutil.White("dlc contract ls")),
	Description:      "Shows a list of known contracts\n",
	ShortDescription: "Shows a list of known contracts\n",
}

var addContractCommand = &Command{
	Format:           fmt.Sprintf("%s\n", lnutil.White("dlc contract add")),
	Description:      "Adds a new draft contract\n",
	ShortDescription: "Adds a new draft contract\n",
}

var viewContractCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract view"),
		lnutil.ReqColor("id")),
	Description: fmt.Sprintf("%s\n%s\n",
		"Views the current status of a contract",
		fmt.Sprintf("%-10s %s", lnutil.White("id"), "The ID of the contract to view"),
	),
	ShortDescription: "Views the current status of a contract\n",
}

var setContractOracleCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setoracle"),
		lnutil.ReqColor("cid", "oid")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Configures a contract for using a specific oracle",
		fmt.Sprintf("%-10s %s", lnutil.White("cid"), "The ID of the contract"),
		fmt.Sprintf("%-10s %s", lnutil.White("oid"), "The ID of the oracle"),
	),
	ShortDescription: "Configures a contract for using a specific oracle\n",
}

var setContractDatafeedCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setdatafeed"),
		lnutil.ReqColor("cid", "feed")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the data feed to use for the contract",
		fmt.Sprintf("%-10s %s", lnutil.White("cid"), "The ID of the contract"),
		fmt.Sprintf("%-10s %s", lnutil.White("feed"), "The ID of the data feed (provided by the oracle)"),
	),
	ShortDescription: "Configures a contract for using a specific oracle\n",
}
var setContractSettlementTimeCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract settime"),
		lnutil.ReqColor("cid", "time")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the settlement time for the contract",
		fmt.Sprintf("%-10s %s", lnutil.White("cid"), "The ID of the contract"),
		fmt.Sprintf("%-10s %s", lnutil.White("time"), "The settlement time (unix timestamp)"),
	),
	ShortDescription: "Sets the settlement time for the contract\n",
}
var setContractFundingCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setfunding"),
		lnutil.ReqColor("ourAmount", "theirAmount")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the amounts both parties in the contract will fund",
		fmt.Sprintf("%-10s %s", lnutil.White("ourAmount"), "The amount we will fund"),
		fmt.Sprintf("%-10s %s", lnutil.White("theirAmount"), "The amount our peer will fund"),
	),
	ShortDescription: "Sets the amounts both parties in the contract will fund\n",
}
var setContractSettlementDivisionCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setdivision"),
		lnutil.ReqColor("valueAllForUs", "valueAllForThem")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the values of the oracle data that will result in the full contract funds being paid to either peer",
		fmt.Sprintf("%-10s %s", lnutil.White("valueAllForUs"), "The outcome with which we will be entitled to the full contract value"),
		fmt.Sprintf("%-10s %s", lnutil.White("valueAllForThem"), "The outcome with which our peer will be entitled to the full contract value"),
	),
	ShortDescription: "Sets the edge values of the oracle data for dividing the funds\n",
}

func (lc *litAfClient) Dlc(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, dlcCommand.Format)
		fmt.Fprintf(color.Output, dlcCommand.Description)
		return nil
	}

	if len(textArgs) > 0 && textArgs[0] == "oracle" {
		return lc.DlcOracle(textArgs[1:])
	}
	if len(textArgs) > 0 && textArgs[0] == "contract" {
		return lc.DlcContract(textArgs[1:])
	}
	return fmt.Errorf(dlcCommand.Format)
}

func (lc *litAfClient) DlcOracle(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, oracleCommand.Format)
		fmt.Fprintf(color.Output, oracleCommand.Description)
		return nil
	}

	if len(textArgs) > 0 && textArgs[0] == "ls" {
		return lc.DlcListOracles(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "add" {
		return lc.DlcAddOracle(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "import" {
		return lc.DlcImportOracle(textArgs[1:])
	}

	return fmt.Errorf(oracleCommand.Format)
}

func (lc *litAfClient) DlcListOracles(textArgs []string) error {
	args := new(litrpc.ListOraclesArgs)
	reply := new(litrpc.ListOraclesReply)

	err := lc.rpccon.Call("LitRPC.ListOracles", args, reply)
	if err != nil {
		return err
	}
	if len(reply.Oracles) == 0 {
		fmt.Println("No oracles found")
	}
	for _, o := range reply.Oracles {
		fmt.Fprintf(color.Output, "%04d: [%x...%x] [%x...%x] [%x...%x] %s\n", o.Idx, o.A[:2], o.A[31:], o.B[:2], o.B[31:], o.Q[:2], o.Q[31:], o.Name)
	}

	return nil
}

func (lc *litAfClient) DlcImportOracle(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, importOracleCommand.Format)
		fmt.Fprintf(color.Output, importOracleCommand.Description)
		return nil
	}

	if len(textArgs) < 2 {
		return fmt.Errorf(importOracleCommand.Format)
	}

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

func (lc *litAfClient) DlcAddOracle(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, addOracleCommand.Format)
		fmt.Fprintf(color.Output, addOracleCommand.Description)
		return nil
	}

	if len(textArgs) < 2 {
		return fmt.Errorf(addOracleCommand.Format)
	}

	args := new(litrpc.AddOracleArgs)
	reply := new(litrpc.AddOracleReply)

	args.Keys = textArgs[0]
	args.Name = textArgs[1]

	err := lc.rpccon.Call("LitRPC.AddOracle", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "Oracle succesfully registered under ID %d\n", reply.Oracle.Idx)
	return nil
}

func (lc *litAfClient) DlcContract(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, contractCommand.Format)
		fmt.Fprintf(color.Output, contractCommand.Description)
		return nil
	}

	if len(textArgs) > 0 && textArgs[0] == "ls" {
		return lc.DlcListContracts(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "new" {
		return lc.DlcNewContract(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "view" {
		return lc.DlcViewContract(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "setoracle" {
		return lc.DlcSetContractOracle(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "setdatafeed" {
		return lc.DlcSetContractDatafeed(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "settime" {
		return lc.DlcSetContractSettlementTime(textArgs[1:])
	}
	return fmt.Errorf(contractCommand.Format)
}

func (lc *litAfClient) DlcListContracts(textArgs []string) error {
	args := new(litrpc.ListContractsArgs)
	reply := new(litrpc.ListContractsReply)

	err := lc.rpccon.Call("LitRPC.ListContracts", args, reply)
	if err != nil {
		return err
	}

	if len(reply.Contracts) == 0 {
		fmt.Println("No contracts found")
	}

	for _, c := range reply.Contracts {
		fmt.Fprintf(color.Output, "%04d: \n", c.Idx)
	}

	return nil
}

func (lc *litAfClient) DlcNewContract(textArgs []string) error {
	args := new(litrpc.NewContractArgs)
	reply := new(litrpc.NewContractReply)

	err := lc.rpccon.Call("LitRPC.NewContract", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Contract succesfully created\n\n")
	PrintContract(reply.Contract)
	return nil
}

func (lc *litAfClient) DlcViewContract(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, viewContractCommand.Format)
		fmt.Fprintf(color.Output, viewContractCommand.Description)
		return nil
	}

	if len(textArgs) < 1 {
		return fmt.Errorf(viewContractCommand.Format)
	}

	args := new(litrpc.GetContractArgs)
	reply := new(litrpc.GetContractReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	args.Idx = cIdx

	err = lc.rpccon.Call("LitRPC.GetContract", args, reply)
	if err != nil {
		return err
	}

	PrintContract(reply.Contract)
	return nil
}

func (lc *litAfClient) DlcSetContractOracle(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, setContractOracleCommand.Format)
		fmt.Fprintf(color.Output, setContractOracleCommand.Description)
		return nil
	}

	if len(textArgs) < 1 {
		return fmt.Errorf(setContractOracleCommand.Format)
	}

	args := new(litrpc.SetContractOracleArgs)
	reply := new(litrpc.SetContractOracleReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	oIdx, err := strconv.ParseUint(textArgs[1], 10, 64)
	if err != nil {
		return err
	}
	args.CIdx = cIdx
	args.OIdx = oIdx

	err = lc.rpccon.Call("LitRPC.SetContractOracle", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Oracle set succesfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractDatafeed(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, setContractDatafeedCommand.Format)
		fmt.Fprintf(color.Output, setContractDatafeedCommand.Description)
		return nil
	}

	if len(textArgs) < 1 {
		return fmt.Errorf(setContractDatafeedCommand.Format)
	}

	args := new(litrpc.SetContractDatafeedArgs)
	reply := new(litrpc.SetContractDatafeedReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	feed, err := strconv.ParseUint(textArgs[1], 10, 64)
	if err != nil {
		return err
	}
	args.CIdx = cIdx
	args.Feed = feed

	err = lc.rpccon.Call("LitRPC.SetContractDatafeed", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Datafeed set succesfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractSettlementTime(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, setContractSettlementTimeCommand.Format)
		fmt.Fprintf(color.Output, setContractSettlementTimeCommand.Description)
		return nil
	}

	if len(textArgs) < 1 {
		return fmt.Errorf(setContractSettlementTimeCommand.Format)
	}

	args := new(litrpc.SetContractSettlementTimeArgs)
	reply := new(litrpc.SetContractSettlementTimeReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	time, err := strconv.ParseUint(textArgs[1], 10, 64)
	if err != nil {
		return err
	}
	args.CIdx = cIdx
	args.Time = time

	err = lc.rpccon.Call("LitRPC.SetContractSettlementTime", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Settlement time set succesfully\n")

	return nil
}

func PrintContract(c *dlc.DlcContract) {
	fmt.Fprintf(color.Output, "%-30s : %d\n", lnutil.White("Index"), c.Idx)
	fmt.Fprintf(color.Output, "%-30s : [%x...%x] [%x...%x] [%x...%x]\n", lnutil.White("Oracle keys"), c.OracleA[:2], c.OracleA[31:], c.OracleB[:2], c.OracleB[31:], c.OracleQ[:2], c.OracleQ[31:])
	fmt.Fprintf(color.Output, "%-30s : %04x\n", lnutil.White("Oracle feed"), c.OracleDataFeed)
	fmt.Fprintf(color.Output, "%-30s : %s\n", lnutil.White("Settlement time"), time.Unix(int64(c.OracleTimestamp), 0).UTC().Format(time.UnixDate))
	fmt.Fprintf(color.Output, "%-30s : %d\n", lnutil.White("Funded by us"), c.OurFundingAmount)
	fmt.Fprintf(color.Output, "%-30s : %d\n", lnutil.White("Funded by peer"), c.TheirFundingAmount)
	fmt.Fprintf(color.Output, "%-30s : %d\n", lnutil.White("Value 100% us"), c.ValueAllOurs)
	fmt.Fprintf(color.Output, "%-30s : %d\n", lnutil.White("Value 100% peer"), c.ValueAllTheirs)

	status := "Draft"
	switch c.Status {
	case dlc.ContractStatusActive:
		status = "Active"
	case dlc.ContractStatusClosed:
		status = "Closed"
	case dlc.ContractStatusOffered:
		status = "Offered, awaiting reply"
	}

	fmt.Fprintf(color.Output, "%-30s : %s\n", lnutil.White("Status"), status)

}
