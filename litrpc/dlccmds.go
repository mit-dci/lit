package litrpc

import (
	"encoding/hex"

	"github.com/mit-dci/lit/dlc"
)

// ------------------------- statedump
type ListOraclesArgs struct {
	// none
}

type ListOraclesReply struct {
	Oracles []*dlc.Oracle
}

// ListOracles will return all oracles know to LIT
func (r *LitRPC) ListOracles(args ListOraclesArgs, reply *ListOraclesReply) error {
	var err error
	reply.Oracles, err = r.Node.DlcManager.LoadAllOracles()
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
	Oracle *dlc.Oracle
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
	Keys string
	Name string
}

type AddOracleReply struct {
	Oracle *dlc.Oracle
}

func (r *LitRPC) AddOracle(args AddOracleArgs, reply *AddOracleReply) error {
	var err error
	parsedKeys, err := hex.DecodeString(args.Keys)
	if err != nil {
		return err
	}

	var keys [99]byte
	copy(keys[:], parsedKeys)

	reply.Oracle, err = r.Node.DlcManager.AddOracle(keys, args.Name)
	if err != nil {
		return err
	}

	return nil
}
