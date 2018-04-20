package litrpc

import (
	"encoding/hex"

	"github.com/mit-dci/lit/dlc"
	"github.com/mit-dci/lit/lnutil"
)

type ListOraclesArgs struct {
	// none
}

type ListOraclesReply struct {
	Oracles []*dlc.DlcOracle
}

// ListOracles will return all oracles know to LIT
func (r *LitRPC) ListOracles(args ListOraclesArgs, reply *ListOraclesReply) error {
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

func (r *LitRPC) ImportOracle(args ImportOracleArgs, reply *ImportOracleReply) error {
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

func (r *LitRPC) NewContract(args NewContractArgs, reply *NewContractReply) error {
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

// ListOracles will return all contracts know to LIT
func (r *LitRPC) ListContracts(args ListContractsArgs, reply *ListContractsReply) error {
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

func (r *LitRPC) GetContract(args GetContractArgs, reply *GetContractReply) error {
	var err error

	reply.Contract, err = r.Node.DlcManager.LoadContract(args.Idx)
	if err != nil {
		return err
	}

	return nil
}

type SetContractOracleArgs struct {
	CIdx uint64
	OIdx uint64
}

type SetContractOracleReply struct {
	Success bool
}

func (r *LitRPC) SetContractOracle(args SetContractOracleArgs, reply *SetContractOracleReply) error {
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

func (r *LitRPC) SetContractDatafeed(args SetContractDatafeedArgs, reply *SetContractDatafeedReply) error {
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
	RPoint [33]byte
}

type SetContractRPointReply struct {
	Success bool
}

func (r *LitRPC) SetContractRPoint(args SetContractRPointArgs, reply *SetContractRPointReply) error {
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

func (r *LitRPC) SetContractSettlementTime(args SetContractSettlementTimeArgs, reply *SetContractSettlementTimeReply) error {
	var err error

	err = r.Node.DlcManager.SetContractSettlementTime(args.CIdx, args.Time)
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

func (r *LitRPC) SetContractFunding(args SetContractFundingArgs, reply *SetContractFundingReply) error {
	var err error

	err = r.Node.DlcManager.SetContractFunding(args.CIdx, args.OurAmount, args.TheirAmount)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type SetContractSettlementDivisionArgs struct {
	CIdx             uint64
	ValueFullyOurs   int64
	ValueFullyTheirs int64
}

type SetContractSettlementDivisionReply struct {
	Success bool
}

func (r *LitRPC) SetContractSettlementDivision(args SetContractSettlementDivisionArgs, reply *SetContractSettlementDivisionReply) error {
	var err error

	err = r.Node.DlcManager.SetContractSettlementDivision(args.CIdx, args.ValueFullyOurs, args.ValueFullyTheirs)
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

func (r *LitRPC) SetContractCoinType(args SetContractCoinTypeArgs, reply *SetContractCoinTypeReply) error {
	var err error

	err = r.Node.DlcManager.SetContractCoinType(args.CIdx, args.CoinType)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type OfferContractArgs struct {
	CIdx    uint64
	PeerIdx uint32
}

type OfferContractReply struct {
	Success bool
}

func (r *LitRPC) OfferContract(args OfferContractArgs, reply *OfferContractReply) error {
	var err error

	err = r.Node.OfferDlc(args.PeerIdx, args.CIdx)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type DeclineContractArgs struct {
	CIdx uint64
}

type DeclineContractReply struct {
	Success bool
}

func (r *LitRPC) DeclineContract(args DeclineContractArgs, reply *DeclineContractReply) error {
	var err error

	err = r.Node.DeclineDlc(args.CIdx)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type AcceptContractArgs struct {
	CIdx uint64
}

type AcceptContractReply struct {
	Success bool
}

func (r *LitRPC) AcceptContract(args AcceptContractArgs, reply *AcceptContractReply) error {
	var err error

	err = r.Node.AcceptDlc(args.CIdx)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}

type SettleContractArgs struct {
	CIdx        uint64
	OracleValue int64
	OracleSig   [32]byte
}

type SettleContractReply struct {
	Success bool
}

func (r *LitRPC) SettleContract(args SettleContractArgs, reply *SettleContractReply) error {
	var err error

	err = r.Node.SettleContract(args.CIdx, args.OracleValue, args.OracleSig)
	if err != nil {
		return err
	}

	reply.Success = true
	return nil
}
