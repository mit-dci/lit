package main

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
	"errors"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
)

var dlcCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("dlc"),
		lnutil.ReqColor("subcommand"), lnutil.OptColor("parameters...")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n",
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
			lnutil.White("setrefundtime"),
			"Sets the refund time of a contract"),			
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
			lnutil.White("setfeeperbyte"),
		fmt.Sprintf("%-20s %s",
			lnutil.White("setoraclesnumber"),			
			"Sets the oracles number for a contract"),			
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
	Format: fmt.Sprintf("%s%s%s%s%s\n", lnutil.White("dlc contract viewpayout"),
		lnutil.ReqColor("id"), lnutil.ReqColor("start"),
		lnutil.ReqColor("end"), lnutil.ReqColor("increment")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n",
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


var setContractRefundTimeCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract settime"),
		lnutil.ReqColor("cid", "time")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the refund time for the contract",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("time"),
			"The refund time (unix timestamp)"),
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
var setContractFeePerByteCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setfeeperbyte"),
		lnutil.ReqColor("cid", "feeperbyte")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the fee per byte to use for the contract",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("feeperbyte"),
			"The fee per byte in satoshi to use for the contract"),
	),
	ShortDescription: "Sets the fee per byte in satoshi to use for the contract\n",
}

var setContractOraclesNumberCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("dlc contract setoraclesnumber"),
		lnutil.ReqColor("cid", "oraclesnumber")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Sets the oracles number to use for the contract",
		fmt.Sprintf("%-10s %s",
			lnutil.White("cid"),
			"The ID of the contract"),
		fmt.Sprintf("%-10s %s",
			lnutil.White("oraclesnumber"),
			"The oracles number to use for the contract"),
	),
	ShortDescription: "Sets a number of oracles required for the contract\n",
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

	err := lc.Call("LitRPC.ListOracles", args, reply)
	if err != nil {
		return err
	}
	if len(reply.Oracles) == 0 {
		logging.Infof("No oracles found")
	}
	for _, o := range reply.Oracles {
		fmt.Fprintf(color.Output, "%04d: [%x...%x...%x]  %s\n",
			o.Idx, o.A[:2], o.A[15:16], o.A[31:], o.Name)
	}

	return nil
}

func (lc *litAfClient) DlcImportOracle(textArgs []string) error {
	stopEx, err := CheckHelpCommand(importOracleCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.ImportOracleArgs)
	reply := new(litrpc.ImportOracleReply)

	args.Url = textArgs[0]
	args.Name = textArgs[1]

	err = lc.Call("LitRPC.ImportOracle", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "Oracle successfully registered under ID %d\n",
		reply.Oracle.Idx)
	return nil
}

func (lc *litAfClient) DlcAddOracle(textArgs []string) error {
	stopEx, err := CheckHelpCommand(addOracleCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	if err != nil {
		return err
	}

	args := new(litrpc.AddOracleArgs)
	reply := new(litrpc.AddOracleReply)

	args.Key = textArgs[0]
	args.Name = textArgs[1]

	err = lc.Call("LitRPC.AddOracle", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "Oracle successfully registered under ID %d\n",
		reply.Oracle.Idx)
	return nil
}

func (lc *litAfClient) DlcContract(textArgs []string) error {
	if len(textArgs) < 1 { // this shouldn't happen?
		return fmt.Errorf("No argument specified")
	}
	cmd := textArgs[0]
	textArgs = textArgs[1:]
	if cmd == "-h" {
		fmt.Fprintf(color.Output, contractCommand.Format)
		fmt.Fprintf(color.Output, contractCommand.Description)
		return nil
	}

	if cmd == "ls" {
		return lc.DlcListContracts(textArgs)
	}

	if cmd == "new" {
		return lc.DlcNewContract(textArgs)
	}

	if cmd == "view" {
		return lc.DlcViewContract(textArgs)
	}

	if cmd == "viewpayout" {
		return lc.DlcViewContractPayout(textArgs)
	}

	if cmd == "setoracle" {
		return lc.DlcSetContractOracle(textArgs)
	}

	if cmd == "setdatafeed" {
		return lc.DlcSetContractDatafeed(textArgs)
	}

	if cmd == "setrpoint" {
		return lc.DlcSetContractRPoint(textArgs)
	}

	if cmd == "settime" {
		return lc.DlcSetContractSettlementTime(textArgs)
	}

	if cmd == "setrefundtime" {
		return lc.DlcSetContractRefundTime(textArgs)
	}	

	if cmd == "setfunding" {
		return lc.DlcSetContractFunding(textArgs)
	}

	if cmd == "setdivision" {
		return lc.DlcSetContractDivision(textArgs)
	}

	if cmd == "setcointype" {
		return lc.DlcSetContractCoinType(textArgs)
	}

	if cmd == "setfeeperbyte" {
		return lc.DlcSetContractFeePerByte(textArgs)
	}
	
	if cmd == "setoraclesnumber" {
		return lc.DlcSetContractOraclesNumber(textArgs)
	}		

	if cmd == "offer" {
		return lc.DlcOfferContract(textArgs)
	}

	if cmd == "decline" {
		return lc.DlcDeclineContract(textArgs)
	}

	if cmd == "accept" {
		return lc.DlcAcceptContract(textArgs)
	}

	if cmd == "settle" {
		return lc.DlcSettleContract(textArgs)
	}
	return fmt.Errorf(contractCommand.Format)
}

func (lc *litAfClient) DlcListContracts(textArgs []string) error {
	args := new(litrpc.ListContractsArgs)
	reply := new(litrpc.ListContractsReply)

	err := lc.Call("LitRPC.ListContracts", args, reply)
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

	err := lc.Call("LitRPC.NewContract", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Contract successfully created\n\n")
	PrintContract(reply.Contract)
	return nil
}

func (lc *litAfClient) DlcViewContract(textArgs []string) error {
	stopEx, err := CheckHelpCommand(viewContractCommand, textArgs, 1)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.GetContractArgs)
	reply := new(litrpc.GetContractReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	args.Idx = cIdx

	err = lc.Call("LitRPC.GetContract", args, reply)
	if err != nil {
		return err
	}

	PrintContract(reply.Contract)
	return nil
}

func (lc *litAfClient) DlcViewContractPayout(textArgs []string) error {
	stopEx, err := CheckHelpCommand(viewContractPayoutCommand, textArgs, 4)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.GetContract", args, reply)
	if err != nil {
		return err
	}

	PrintPayout(reply.Contract, start, end, increment)
	return nil
}

func (lc *litAfClient) DlcSetContractOracle(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractOracleCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.SetContractOracle", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Oracle set successfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractDatafeed(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractDatafeedCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.SetContractDatafeed", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Datafeed set successfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractRPoint(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractRPointCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.SetContractRPoint", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "R-point set successfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractSettlementTime(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractSettlementTimeCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.SetContractSettlementTime", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Settlement time set successfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractRefundTime(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractRefundTimeCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.SetContractRefundTime", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Refund time set successfully\n")

	return nil
}



func (lc *litAfClient) DlcSetContractFunding(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractFundingCommand, textArgs, 3)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.SetContractFunding", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Funding set successfully\n")

	return nil
}

func (lc *litAfClient) DlcSetContractCoinType(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractCoinTypeCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.SetContractCoinType", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Cointype set successfully\n")

	return nil
}


func (lc *litAfClient) DlcSetContractFeePerByte(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractFeePerByteCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.SetContractFeePerByteArgs)
	reply := new(litrpc.SetContractFeePerByteReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	feeperbyte, err := strconv.ParseUint(textArgs[1], 10, 64)
	if err != nil {
		return err
	}

	args.CIdx = cIdx
	args.FeePerByte = uint32(feeperbyte)

	err = lc.Call("LitRPC.SetContractFeePerByte", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Fee per byte set successfully\n")

	return nil
}




func (lc *litAfClient) DlcSetContractOraclesNumber(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractOraclesNumberCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
	}

	args := new(litrpc.SetContractOraclesNumberArgs)
	reply := new(litrpc.SetContractOraclesNumberReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}
	OraclesNumber, err := strconv.ParseUint(textArgs[1], 10, 64)
	if err != nil {
		return err
	}

	if OraclesNumber > 1 {
		return errors.New("Multiple oracles supported only from RPC cals.")
	}

	args.CIdx = cIdx
	args.OraclesNumber = uint32(OraclesNumber)

	err = lc.Call("LitRPC.SetContractOraclesNumber", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "SetContractOraclesNumber set successfully\n")

	return nil
}




func (lc *litAfClient) DlcSetContractDivision(textArgs []string) error {
	stopEx, err := CheckHelpCommand(setContractDivisionCommand, textArgs, 3)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.SetContractDivision", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Funding set successfully\n")

	return nil
}

func (lc *litAfClient) DlcOfferContract(textArgs []string) error {
	stopEx, err := CheckHelpCommand(offerContractCommand, textArgs, 2)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.OfferContract", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Offer sent set successfully\n")

	return nil
}

func (lc *litAfClient) dlcContractRespond(textArgs []string, aor bool) error {
	args := new(litrpc.ContractRespondArgs)
	reply := new(litrpc.ContractRespondReply)

	cIdx, err := strconv.ParseUint(textArgs[0], 10, 64)
	if err != nil {
		return err
	}

	args.CIdx = cIdx
	args.AcceptOrDecline = aor

	err = lc.Call("LitRPC.ContractRespond", args, reply)
	if err != nil {
		return err
	}

	if aor {
		fmt.Fprintf(color.Output, "Offer acceptance initiated. Use [dlc contract view %d] to see the status.\n", cIdx)
	} else {
		fmt.Fprint(color.Output, "Offer declined successfully\n")
	}

	return nil
}

func (lc *litAfClient) DlcDeclineContract(textArgs []string) error {
	stopEx, err := CheckHelpCommand(declineContractCommand, textArgs, 1)
	if err != nil || stopEx {
		return err
	}

	return lc.dlcContractRespond(textArgs, false)
}

func (lc *litAfClient) DlcAcceptContract(textArgs []string) error {
	stopEx, err := CheckHelpCommand(acceptContractCommand, textArgs, 1)
	if err != nil || stopEx {
		return err
	}

	return lc.dlcContractRespond(textArgs, true)
}

func (lc *litAfClient) DlcSettleContract(textArgs []string) error {
	stopEx, err := CheckHelpCommand(settleContractCommand, textArgs, 3)
	if err != nil || stopEx {
		return err
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

	err = lc.Call("LitRPC.SettleContract", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprint(color.Output, "Contract settled successfully\n")

	return nil
}

func PrintContract(c *lnutil.DlcContract) {
	fmt.Fprintf(color.Output, "%-30s : %d\n", lnutil.White("Index"), c.Idx)
	fmt.Fprintf(color.Output, "%-30s : [%x...%x...%x]\n",
		lnutil.White("Oracle public key"),
		c.OracleA[0][:2], c.OracleA[0][15:16], c.OracleA[0][31:])
	fmt.Fprintf(color.Output, "%-30s : [%x...%x...%x]\n",
		lnutil.White("Oracle R-point"), c.OracleR[:2],
		c.OracleR[0][15:16], c.OracleR[0][31:])
	fmt.Fprintf(color.Output, "%-30s : %s\n",
		lnutil.White("Settlement time"),
		time.Unix(int64(c.OracleTimestamp), 0).UTC().Format(time.UnixDate))
	fmt.Fprintf(color.Output, "%-30s : %s\n",
		lnutil.White("Refund time"),
		time.Unix(int64(c.RefundTimestamp), 0).UTC().Format(time.UnixDate))		
	fmt.Fprintf(color.Output, "%-30s : %d\n",
		lnutil.White("Funded by us"), c.OurFundingAmount)
	fmt.Fprintf(color.Output, "%-30s : %d\n",
		lnutil.White("Funded by peer"), c.TheirFundingAmount)
	fmt.Fprintf(color.Output, "%-30s : %d\n",
		lnutil.White("Coin type"), c.CoinType)
	fmt.Fprintf(color.Output, "%-30s : %d\n",
		lnutil.White("Fee per byte"), c.FeePerByte)		
	fmt.Fprintf(color.Output, "%-30s : %d\n",
		lnutil.White("Oracles number"), c.OraclesNumber)	


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
	case lnutil.ContractStatusAccepting:
		status = "Accepting"
	case lnutil.ContractStatusAccepted:
		status = "Accepted"
	case lnutil.ContractStatusAcknowledged:
		status = "Acknowledged"
	case lnutil.ContractStatusError:
		status = "Error"
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
