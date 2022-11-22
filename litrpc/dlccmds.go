package litrpc

import (
	"encoding/hex"
	"fmt"

	"github.com/mit-dci/lit/dlc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/consts"
)

type ListOraclesArgs struct {
	// none
}

type ListOraclesReply struct {
	Oracles []*dlc.DlcOracle
}

// ListOracles returns all oracles know to LIT
func (r *LitRPC) ListOracles(args ListOraclesArgs,
	reply *ListOraclesReply) error {
	var err error

	reply.Oracles, err = r.Node.DlcManager.ListOracles()
	if err != nil {
		return err
	}

	return nil
}

type ImportOracleArgs struct {
	Url  string
	Name string
}

type ImportOracleReply struct {
	Oracle *dlc.DlcOracle
}

// ImportOracle imports an oracle from a REST API
func (r *LitRPC) ImportOracle(args ImportOracleArgs,
	reply *ImportOracleReply) error {
	var err error
	reply.Oracle, err = r.Node.DlcManager.ImportOracle(args.Url, args.Name)
	if err != nil {
		return err
	}

	return nil
}

type AddOracleArgs struct {
	Key  string
	Name string
}

type AddOracleReply struct {
	Oracle *dlc.DlcOracle
}

// AddOracle manually adds an oracle from its PubKey A
func (r *LitRPC) AddOracle(args AddOracleArgs, reply *AddOracleReply) error {
	var err error
	parsedKey, err := hex.DecodeString(args.Key)
	if err != nil {
		return err
	}

	var key [33]byte
	copy(key[:], parsedKey)

	reply.Oracle, err = r.Node.DlcManager.AddOracle(key, args.Name)
	if err != nil {
		return err
	}

	return nil
}

type NewContractArgs struct {
	// empty
}

type NewContractReply struct {
	Contract *lnutil.DlcContract
}

// NewContract creates a new draft contract
func (r *LitRPC) NewContract(args NewContractArgs,
	reply *NewContractReply) error {
	var err error

	reply.Contract, err = r.Node.AddContract()
	if err != nil {
		return err
	}

	return nil
}

type ListContractsArgs struct {
	// none
}

type ListContractsReply struct {
	Contracts []*lnutil.DlcContract
}

// ListContracts returns all contracts know to LIT
func (r *LitRPC) ListContracts(args ListContractsArgs,
	reply *ListContractsReply) error {
	var err error

	reply.Contracts, err = r.Node.DlcManager.ListContracts()
	if err != nil {
		return err
	}

	return nil
}

type GetContractArgs struct {
	Idx uint64
}

type GetContractReply struct {
	Contract *lnutil.DlcContract
}

// GetContract returns a single contract based on its index
func (r *LitRPC) GetContract(args GetContractArgs,
	reply *GetContractReply) error {
	var err error

	reply.Contract, err = r.Node.DlcManager.LoadContract(args.Idx)
	if err != nil {
		return err
	}

	return nil
}

type SetContractOracleArgs struct {
	CIdx uint64
	OIdx []uint64
}

type SetContractOracleReply struct {
	Success bool
}

// SetContractOracle assigns a known oracle to a (new) contract
func (r *LitRPC) SetContractOracle(args SetContractOracleArgs,
	reply *SetContractOracleReply) error {
	var err error

	err = r.Node.DlcManager.SetContractOracle(args.CIdx, args.OIdx)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type SetContractDatafeedArgs struct {
	CIdx uint64
	Feed uint64
}

type SetContractDatafeedReply struct {
	Success bool
}

// SetContractDatafeed sets a data feed by index to a contract, which is then
// used to fetch the R-point from the oracle's REST API
func (r *LitRPC) SetContractDatafeed(args SetContractDatafeedArgs,
	reply *SetContractDatafeedReply) error {
	var err error

	err = r.Node.DlcManager.SetContractDatafeed(args.CIdx, args.Feed)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type SetContractRPointArgs struct {
	CIdx   uint64
	RPoint [][33]byte
}

type SetContractRPointReply struct {
	Success bool
}

// SetContractRPoint manually sets the R-point for the contract using a pubkey
func (r *LitRPC) SetContractRPoint(args SetContractRPointArgs,
	reply *SetContractRPointReply) error {
	var err error

	err = r.Node.DlcManager.SetContractRPoint(args.CIdx, args.RPoint)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type SetContractSettlementTimeArgs struct {
	CIdx uint64
	Time uint64
}

type SetContractSettlementTimeReply struct {
	Success bool
}

// SetContractSettlementTime sets the time this the oracle will publish data (
// unix time)
func (r *LitRPC) SetContractSettlementTime(args SetContractSettlementTimeArgs,
	reply *SetContractSettlementTimeReply) error {
	var err error

	err = r.Node.DlcManager.SetContractSettlementTime(args.CIdx, args.Time)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}


// SetContractRefundTime. If until this time Oracle does not publish the data, 
// then either party can publish a RefundTransaction
func (r *LitRPC) SetContractRefundTime(args SetContractSettlementTimeArgs,
	reply *SetContractSettlementTimeReply) error {
	var err error

	err = r.Node.DlcManager.SetContractRefundTime(args.CIdx, args.Time)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}



type SetContractFundingArgs struct {
	CIdx        uint64
	OurAmount   int64
	TheirAmount int64
}

type SetContractFundingReply struct {
	Success bool
}

// SetContractFunding sets the division in funding the channel. The arguments
// decide how much we're funding and how much we expect the peer we offer the
// contract to to fund
func (r *LitRPC) SetContractFunding(args SetContractFundingArgs,
	reply *SetContractFundingReply) error {
	var err error

	err = r.Node.DlcManager.SetContractFunding(args.CIdx,
		args.OurAmount, args.TheirAmount)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type SetContractDivisionArgs struct {
	CIdx             uint64
	ValueFullyOurs   int64
	ValueFullyTheirs int64
}

type SetContractDivisionReply struct {
	Success bool
}

// SetContractDivision sets how the contract is settled. The parameters indicate
// at what value the full contract funds are ours, and at what value they are
// full funds are for our peer. Between those values, the contract will divide
// the contract funds linearly
func (r *LitRPC) SetContractDivision(args SetContractDivisionArgs,
	reply *SetContractDivisionReply) error {
	var err error

	err = r.Node.DlcManager.SetContractDivision(args.CIdx,
		args.ValueFullyOurs, args.ValueFullyTheirs)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type SetContractCoinTypeArgs struct {
	CIdx     uint64
	CoinType uint32
}

type SetContractCoinTypeReply struct {
	Success bool
}

// SetContractCoinType sets the coin type the contract will be in. Note that a
// peer that doesn't have a wallet of that type will automatically decline the
// contract.
func (r *LitRPC) SetContractCoinType(args SetContractCoinTypeArgs,
	reply *SetContractCoinTypeReply) error {
	var err error

	err = r.Node.DlcManager.SetContractCoinType(args.CIdx, args.CoinType)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}


type SetContractFeePerByteArgs struct {
	CIdx     uint64
	FeePerByte uint32
}

type SetContractFeePerByteReply struct {
	Success bool
}


// SetContractFeePerByte sets the fee per byte for the contract.
func (r *LitRPC) SetContractFeePerByte(args SetContractFeePerByteArgs,
	reply *SetContractFeePerByteReply) error {
	var err error

	err = r.Node.DlcManager.SetContractFeePerByte(args.CIdx, args.FeePerByte)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}



type SetContractOraclesNumberArgs struct {
	CIdx     uint64
	OraclesNumber uint32
}

type SetContractOraclesNumberReply struct {
	Success bool
}


func (r *LitRPC) SetContractOraclesNumber(args SetContractOraclesNumberArgs,
	reply *SetContractOraclesNumberReply) error {
	var err error

	err = r.Node.DlcManager.SetContractOraclesNumber(args.CIdx, args.OraclesNumber)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type GetContractDivisionArgs struct {
	CIdx     uint64
	OracleValue int64
}

type GetContractDivisionReply struct {
	ValueOurs int64
}

// GetContractDivision
func (r *LitRPC) GetContractDivision(args GetContractDivisionArgs,
	reply *GetContractDivisionReply) error {

	//err = r.Node.DlcManager.GetContractDivision(args.CIdx, args.OracleValue)

	c, err1 := r.Node.DlcManager.LoadContract(args.CIdx)
	if err1 != nil {
		fmt.Errorf("GetContractDivision(): LoadContract err %s\n", err1.Error())
		return err1
	}


	d, err2 := c.GetDivision(args.OracleValue)
	if err2 != nil {
		fmt.Errorf("GetContractDivision(): c.GetDivision err %s\n", err2.Error())
		return err2
	}
	reply.ValueOurs = d.ValueOurs

	return nil
}


//-----------------------------------------------------------



type OfferContractArgs struct {
	CIdx    uint64
	PeerIdx uint32
}

type OfferContractReply struct {
	Success bool
}

// OfferContract offers a contract to a (connected) peer
func (r *LitRPC) OfferContract(args OfferContractArgs,
	reply *OfferContractReply) error {
	var err error

	err = r.Node.OfferDlc(args.PeerIdx, args.CIdx)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type ContractRespondArgs struct {
	// True for accept, false for decline.
	AcceptOrDecline bool
	CIdx            uint64
}

type ContractRespondReply struct {
	Success bool
}

// DeclineContract declines an offered contract
func (r *LitRPC) ContractRespond(args ContractRespondArgs, reply *ContractRespondReply) error {
	var err error

	if args.AcceptOrDecline {
		err = r.Node.AcceptDlc(args.CIdx)
	} else {
		err = r.Node.DeclineDlc(args.CIdx, 0x01)
	}

	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type SettleContractArgs struct {
	CIdx        uint64
	OracleValue int64
	OracleSig   [consts.MaxOraclesNumber][32]byte
}

type SettleContractReply struct {
	Success      bool
	SettleTxHash [32]byte
	ClaimTxHash  [32]byte
}

// SettleContract uses the value and signature from the oracle to settle the
// contract and send the equivalent settlement transaction to the blockchain.
// It will subsequently claim the contract output back to our wallet
func (r *LitRPC) SettleContract(args SettleContractArgs,
	reply *SettleContractReply) error {
	var err error

	reply.SettleTxHash, reply.ClaimTxHash, err = r.Node.SettleContract(
		args.CIdx, args.OracleValue, args.OracleSig)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}


//======================================================================


type RefundContractArgs struct {
	CIdx        uint64
}

type RefundContractReply struct {
	Success      bool
}

// RefundContract

func (r *LitRPC) RefundContract(args RefundContractArgs,reply *RefundContractReply) error {
	var err error

	reply.Success, err = r.Node.RefundContract(args.CIdx)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}
