package main

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var dlcCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc"),
		lnutil.ReqColor("subcommand"), lnutil.OptColor("parameters...")),
	Description: fmt.Sprintf("%s%2\n%s\n%s\n",
		"Command for working with discreet log contracts. ",
		"Subcommand can be one of:",
		fmt.Sprintf("%-10s %s",
			lnutil.White("oracle"), "Command to manage oracles"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("contract"), "Command to manage contracts"),
	),
	ShortDescription: "Command for working with Discreet Log Contracts.\n",
}

var oracleCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc oracle"),
		lnutil.ReqColor("subcommand"), lnutil.OptColor("parameters...")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n",
		"Command for managing oracles. Subcommand can be one of:",
		fmt.Sprintf("%-20s %s",
			lnutil.White("add"),
			"Adds a new oracle by manually providing the pubkey"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("import"),
			"Imports a new oracle using a URL to its REST interface"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("ls"),
			"Shows a list of known oracles"),
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
		fmt.Sprintf("%-20s %s",
			lnutil.White("url"),
			"URL to the root of the publishes dlcoracle REST interface"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("name"),
			"Name under which to register the oracle in LIT"),
	),
	ShortDescription: "Imports a new oracle into LIT from a REST interface\n",
}

var addOracleCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc oracle add"),
		lnutil.ReqColor("keys"), lnutil.ReqColor("name")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Adds a new oracle by entering the pubkeys manually",
		fmt.Sprintf("%-20s %s",
			lnutil.White("keys"),
			"Public key for the oracle (33 bytes in hex)"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("name"),
			"Name under which to register the oracle in LIT"),
	),
	ShortDescription: "Adds a new oracle into LIT\n",
}

var contractCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc contract"),
		lnutil.ReqColor("subcommand"), lnutil.OptColor("parameters...")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n"+
		"%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n",
		"Command for managing contracts. Subcommand can be one of:",
		fmt.Sprintf("%-20s %s",
			lnutil.White("new"),
			"Adds a new draft contract"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("view"),
			"Views a contract"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("viewpayout"),
			"Views the payout table of a contract"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("setoracle"),
			"Sets a contract to use a particular oracle"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("settime"),
			"Sets the settlement time of a contract"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("setdatafeed"),
			"Sets the data feed to use, will fetch the R point"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("setrpoint"),
			"Sets the R point manually"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("setfunding"),
			"Sets the funding parameters of a contract"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("setdivision"),
			"Sets the settlement division of a contract"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("setcointype"),
			"Sets the cointype of a contract"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("offer"),
			"Offer a draft contract to one of your peers"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("decline"),
			"Decline a contract sent to you"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("settle"),
			"Settles the contract"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("ls"),
			"Shows a list of known contracts"),
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
		fmt.Sprintf("%-10s %s",
			lnutil.White("id"),
			"The ID of the contract to view"),
	),
	ShortDescription: "Views the current status of a contract\n",
}

var viewContractPayoutCommand = &Command{
	Format: fmt.Sprintf("%s%s%s%s\n", lnutil.White("dlc contract viewpayout"),
		lnutil.ReqColor("id"), lnutil.ReqColor("start"),
		lnutil.ReqColor("end"), lnutil.ReqColor("increment")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n",
		"Views the payout table of a contract",
		fmt.Sprintf("%-10s %s",
			lnutil.White("id"),
			"The ID of the contract to view"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("start"),
			"The start value to print payout data for"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("end"),
			"The end value to print payout data for"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("increment"),
			"Print every X oracle value (1 = all)"),
	),
	ShortDescription: "Views the payout table of a contract\n",
}

var setContractOracleCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setoracle"),
		lnutil.ReqColor("cid", "oid")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Configures a contract for using a specific oracle",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("oid"),
			"The ID of the oracle"),
	),
	ShortDescription: "Configures a contract for using a specific oracle\n",
}

var setContractDatafeedCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setdatafeed"),
		lnutil.ReqColor("cid", "feed")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the data feed to use for the contract",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("feed"),
			"The ID of the data feed (provided by the oracle)"),
	),
	ShortDescription: "Sets the data feed to use for the contract\n",
}

var setContractRPointCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setrpoint"),
		lnutil.ReqColor("cid", "rpoint")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the R point to use for the contract",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("rpoint"),
			"The Rpoint of the publication to use (33 byte in hex)"),
	),
	ShortDescription: "Sets the R point to use for the contract\n",
}

var setContractSettlementTimeCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract settime"),
		lnutil.ReqColor("cid", "time")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the settlement time for the contract",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("time"),
			"The settlement time (unix timestamp)"),
	),
	ShortDescription: "Sets the settlement time for the contract\n",
}
var setContractFundingCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setfunding"),
		lnutil.ReqColor("cid", "ourAmount", "theirAmount")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n",
		"Sets the amounts both parties in the contract will fund",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("ourAmount"),
			"The amount we will fund"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("theirAmount"),
			"The amount our peer will fund"),
	),
	ShortDescription: "Sets the amount both parties will fund\n",
}
var setContractDivisionCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setdivision"),
		lnutil.ReqColor("cid", "valueAllForUs", "valueAllForThem")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n",
		"Sets the values of the oracle data that will result in the full"+
			"contract funds being paid to either peer",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("valueAllForUs"),
			"The outcome with which we will be entitled to the full"+
				" contract value"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("valueAllForThem"),
			"The outcome with which our peer will be entitled to the full"+
				" contract value"),
	),
	ShortDescription: "Sets the edge values for dividing the funds\n",
}
var setContractCoinTypeCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setcointype"),
		lnutil.ReqColor("cid", "cointype")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the coin type to use for the contract",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("cointype"),
			"The ID of the coin type to use for the contract"),
	),
	ShortDescription: "Sets the coin type to use for the contract\n",
}
var declineContractCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract decline"),
		lnutil.ReqColor("cid")),
	Description: fmt.Sprintf("%s\n%s\n",
		"Declines a contract offered to you",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract to decline"),
	),
	ShortDescription: "Declines a contract offered to you\n",
}
var acceptContractCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract accept"),
		lnutil.ReqColor("cid")),
	Description: fmt.Sprintf("%s\n%s\n",
		"Accepts a contract offered to you",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract to accept"),
	),
	ShortDescription: "Accepts a contract offered to you\n",
}
var offerContractCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract offer"),
		lnutil.ReqColor("cid", "peer")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Offers a contract to one of your peers",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("cointype"),
			"The ID of the peer to offer the contract to"),
	),
	ShortDescription: "Offers a contract to one of your peers\n",
}
var settleContractCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract settle"),
		lnutil.ReqColor("cid", "oracleValue", "oracleSig")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n",
		"Settles the contract based on a value and signature from the oracle",
		fmt.Sprintf("%-20s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("oracleValue"),
			"The value the oracle published"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("oracleSig"),
			"The signature from the oracle"),
	),
	ShortDescription: "Settles the contract\n",
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
		fmt.Fprintf(color.Output, "%04d: [%x...%x...%x]  %s\n",
			o.Idx, o.A[:2], o.A[15:16], o.A[31:], o.Name)
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

	fmt.Fprintf(color.Output, "Oracle succesfully registered under ID %d\n",
		reply.Oracle.Idx)
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

	args.Key = textArgs[0]
	args.Name = textArgs[1]

	err := lc.rpccon.Call("LitRPC.AddOracle", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "Oracle succesfully registered under ID %d\n",
		reply.Oracle.Idx)
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

	if len(textArgs) > 0 && textArgs[0] == "viewpayout" {
		return lc.DlcViewContractPayout(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "setoracle" {
		return lc.DlcSetContractOracle(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "setdatafeed" {
		return lc.DlcSetContractDatafeed(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "setrpoint" {
		return lc.DlcSetContractRPoint(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "settime" {
		return lc.DlcSetContractSettlementTime(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "setfunding" {
		return lc.DlcSetContractFunding(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "setdivision" {
		return lc.DlcSetContractDivision(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "setcointype" {
		return lc.DlcSetContractCoinType(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "offer" {
		return lc.DlcOfferContract(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "decline" {
		return lc.DlcDeclineContract(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "accept" {
		return lc.DlcAcceptContract(textArgs[1:])
	}

	if len(textArgs) > 0 && textArgs[0] == "settle" {
		return lc.DlcSettleContract(textArgs[1:])
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

func (lc *litAfClient) DlcViewContractPayout(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, viewContractPayoutCommand.Format)
		fmt.Fprintf(color.Output, viewContractPayoutCommand.Description)
		return nil
	}

	if len(textArgs) < 4 {
		return fmt.Errorf(viewContractPayoutCommand.Format)
	}

	args := new(litrpc.GetContractArgs)
	reply := new(litrpc.GetContractReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	start, err := strconv.ParseInt(textArgs[1], 10, 64)
	if err != nil {
		return err
	}
	end, err := strconv.ParseInt(textArgs[2], 10, 64)
	if err != nil {
		return err
	}
	increment, err := strconv.ParseInt(textArgs[3], 10, 64)
	if err != nil {
		return err
	}

	args.Idx = cIdx

	err = lc.rpccon.Call("LitRPC.GetContract", args, reply)
	if err != nil {
		return err
	}

	PrintPayout(reply.Contract, start, end, increment)
	return nil
}

func (lc *litAfClient) DlcSetContractOracle(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, setContractOracleCommand.Format)
		fmt.Fprintf(color.Output, setContractOracleCommand.Description)
		return nil
	}

	if len(textArgs) < 2 {
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

	if len(textArgs) < 2 {
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

func (lc *litAfClient) DlcSetContractRPoint(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, setContractRPointCommand.Format)
		fmt.Fprintf(color.Output, setContractRPointCommand.Description)
		return nil
	}

	if len(textArgs) < 2 {
		return fmt.Errorf(setContractRPointCommand.Format)
	}

	args := new(litrpc.SetContractRPointArgs)
	reply := new(litrpc.SetContractRPointReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	rPoint, err := hex.DecodeString(textArgs[1])
	if err != nil {
		return err
	}
	args.CIdx = cIdx
	copy(args.RPoint[:], rPoint[:])

	err = lc.rpccon.Call("LitRPC.SetContractRPoint", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "R-point set succesfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractSettlementTime(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, setContractSettlementTimeCommand.Format)
		fmt.Fprintf(color.Output, setContractSettlementTimeCommand.Description)
		return nil
	}

	if len(textArgs) < 2 {
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

func (lc *litAfClient) DlcSetContractFunding(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, setContractFundingCommand.Format)
		fmt.Fprintf(color.Output, setContractFundingCommand.Description)
		return nil
	}

	if len(textArgs) < 3 {
		return fmt.Errorf(setContractFundingCommand.Format)
	}

	args := new(litrpc.SetContractFundingArgs)
	reply := new(litrpc.SetContractFundingReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	ourAmount, err := strconv.ParseInt(textArgs[1], 10, 64)
	if err != nil {
		return err
	}
	theirAmount, err := strconv.ParseInt(textArgs[2], 10, 64)
	if err != nil {
		return err
	}
	args.CIdx = cIdx
	args.OurAmount = ourAmount
	args.TheirAmount = theirAmount

	err = lc.rpccon.Call("LitRPC.SetContractFunding", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Funding set succesfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractCoinType(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, setContractCoinTypeCommand.Format)
		fmt.Fprintf(color.Output, setContractCoinTypeCommand.Description)
		return nil
	}

	if len(textArgs) < 2 {
		return fmt.Errorf(setContractCoinTypeCommand.Format)
	}

	args := new(litrpc.SetContractCoinTypeArgs)
	reply := new(litrpc.SetContractCoinTypeReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	cointype, err := strconv.ParseUint(textArgs[1], 10, 64)
	if err != nil {
		return err
	}

	args.CIdx = cIdx
	args.CoinType = uint32(cointype)

	err = lc.rpccon.Call("LitRPC.SetContractCoinType", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Cointype set successfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractDivision(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, setContractDivisionCommand.Format)
		fmt.Fprintf(color.Output, setContractDivisionCommand.Description)
		return nil
	}

	if len(textArgs) < 3 {
		return fmt.Errorf(setContractDivisionCommand.Format)
	}

	args := new(litrpc.SetContractDivisionArgs)
	reply := new(litrpc.SetContractDivisionReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	fullyOurs, err := strconv.ParseInt(textArgs[1], 10, 64)
	if err != nil {
		return err
	}
	fullyTheirs, err := strconv.ParseInt(textArgs[2], 10, 64)
	if err != nil {
		return err
	}
	args.CIdx = cIdx
	args.ValueFullyOurs = fullyOurs
	args.ValueFullyTheirs = fullyTheirs

	err = lc.rpccon.Call("LitRPC.SetContractDivision", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Funding set succesfully\n")

	return nil
}

func (lc *litAfClient) DlcOfferContract(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, offerContractCommand.Format)
		fmt.Fprintf(color.Output, offerContractCommand.Description)
		return nil
	}

	if len(textArgs) < 2 {
		return fmt.Errorf(offerContractCommand.Format)
	}

	args := new(litrpc.OfferContractArgs)
	reply := new(litrpc.OfferContractReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	peerIdx, err := strconv.ParseUint(textArgs[1], 10, 64)
	if err != nil {
		return err
	}

	args.CIdx = cIdx
	args.PeerIdx = uint32(peerIdx)

	err = lc.rpccon.Call("LitRPC.OfferContract", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Offer sent set succesfully\n")

	return nil
}

func (lc *litAfClient) DlcDeclineContract(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, declineContractCommand.Format)
		fmt.Fprintf(color.Output, declineContractCommand.Description)
		return nil
	}

	if len(textArgs) < 1 {
		return fmt.Errorf(declineContractCommand.Format)
	}

	args := new(litrpc.DeclineContractArgs)
	reply := new(litrpc.DeclineContractArgs)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}

	args.CIdx = cIdx

	err = lc.rpccon.Call("LitRPC.DeclineContract", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Offer declined succesfully\n")

	return nil
}

func (lc *litAfClient) DlcAcceptContract(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, acceptContractCommand.Format)
		fmt.Fprintf(color.Output, acceptContractCommand.Description)
		return nil
	}

	if len(textArgs) < 1 {
		return fmt.Errorf(acceptContractCommand.Format)
	}

	args := new(litrpc.AcceptContractArgs)
	reply := new(litrpc.AcceptContractReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}

	args.CIdx = cIdx

	err = lc.rpccon.Call("LitRPC.AcceptContract", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Offer accepted succesfully\n")

	return nil
}

func (lc *litAfClient) DlcSettleContract(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, settleContractCommand.Format)
		fmt.Fprintf(color.Output, settleContractCommand.Description)
		return nil
	}

	if len(textArgs) < 3 {
		return fmt.Errorf(settleContractCommand.Format)
	}

	args := new(litrpc.SettleContractArgs)
	reply := new(litrpc.SettleContractReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}

	args.CIdx = cIdx

	oracleValue, err := strconv.ParseInt(textArgs[1], 10, 64)
	if err != nil {
		return err
	}

	args.OracleValue = oracleValue
	oracleSigBytes, err := hex.DecodeString(textArgs[2])
	if err != nil {
		return err
	}

	copy(args.OracleSig[:], oracleSigBytes)

	err = lc.rpccon.Call("LitRPC.SettleContract", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Contract settled succesfully\n")

	return nil
}

func PrintContract(c *lnutil.DlcContract) {
	fmt.Fprintf(color.Output, "%-30s : %d\n", lnutil.White("Index"), c.Idx)
	fmt.Fprintf(color.Output, "%-30s : [%x...%x...%x]\n",
		lnutil.White("Oracle public key"),
		c.OracleA[:2], c.OracleA[15:16], c.OracleA[31:])
	fmt.Fprintf(color.Output, "%-30s : [%x...%x...%x]\n",
		lnutil.White("Oracle R-point"), c.OracleR[:2],
		c.OracleR[15:16], c.OracleR[31:])
	fmt.Fprintf(color.Output, "%-30s : %s\n",
		lnutil.White("Settlement time"),
		time.Unix(int64(c.OracleTimestamp), 0).UTC().Format(time.UnixDate))
	fmt.Fprintf(color.Output, "%-30s : %d\n",
		lnutil.White("Funded by us"), c.OurFundingAmount)
	fmt.Fprintf(color.Output, "%-30s : %d\n",
		lnutil.White("Funded by peer"), c.TheirFundingAmount)
	fmt.Fprintf(color.Output, "%-30s : %d\n",
		lnutil.White("Coin type"), c.CoinType)

	peer := "None"
	if c.PeerIdx > 0 {
		peer = fmt.Sprintf("Peer %d", c.PeerIdx)
	}

	fmt.Fprintf(color.Output, "%-30s : %s\n", lnutil.White("Peer"), peer)

	status := "Draft"
	switch c.Status {
	case lnutil.ContractStatusActive:
		status = "Active"
	case lnutil.ContractStatusClosed:
		status = "Closed"
	case lnutil.ContractStatusOfferedByMe:
		status = "Sent offer, awaiting reply"
	case lnutil.ContractStatusOfferedToMe:
		status = "Received offer, awaiting reply"
	case lnutil.ContractStatusAccepted:
		status = "Accepted"
	case lnutil.ContractStatusDeclined:
		status = "Declined"
	}

	fmt.Fprintf(color.Output, "%-30s : %s\n\n", lnutil.White("Status"), status)

	increment := int64(len(c.Division) / 10)
	PrintPayout(c, 0, int64(len(c.Division)), increment)
}

func PrintPayout(c *lnutil.DlcContract, start, end, increment int64) {
	fmt.Fprintf(color.Output, "Payout division:\n\n")
	fmt.Fprintf(color.Output, "%-20s | %-20s | %-20s\n",
		"Oracle value", "Our payout", "Their payout")
	fmt.Fprintf(color.Output, "%s\n", strings.Repeat("-", 66))

	for i := start; i < end; i += increment {
		fmt.Fprintf(color.Output, "%20d | %20d | %20d\n",
			c.Division[i].OracleValue, c.Division[i].ValueOurs,
			c.OurFundingAmount+c.TheirFundingAmount-c.Division[i].ValueOurs)
	}
}
