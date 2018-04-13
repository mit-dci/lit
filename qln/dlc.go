package qln

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/adiabat/btcd/chaincfg/chainhash"
	"github.com/adiabat/btcd/wire"
	"github.com/adiabat/btcutil/txsort"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

func (nd *LitNode) AddContract() (*lnutil.DlcContract, error) {

	c, err := nd.DlcManager.AddContract()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (nd *LitNode) OfferDlc(peerIdx uint32, cIdx uint64) error {
	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	msg := lnutil.NewDlcOfferMsg(peerIdx, c)
	c.Status = lnutil.ContractStatusOfferedByMe
	c.PeerIdx = peerIdx

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = c.CoinType | 1<<31
	kg.Step[2] = UseContractPayout
	kg.Step[3] = c.PeerIdx | 1<<31
	kg.Step[4] = uint32(c.Idx) | 1<<31

	c.OurPayoutPub, err = nd.GetUsePub(kg, UseContractPayout)
	if err != nil {
		return err
	}

	// Fund the contract
	err = nd.FundContract(c)
	if err != nil {
		return err
	}

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.OmniOut <- msg

	return nil
}

func (nd *LitNode) DeclineDlc(cIdx uint64) error {
	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	msg := lnutil.NewDlcOfferDeclineMsg(c.PeerIdx, 0x01, c.PubKey)
	c.Status = lnutil.ContractStatusDeclined

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.OmniOut <- msg

	return nil
}

func (nd *LitNode) AcceptDlc(cIdx uint64) error {
	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	// Fund the contract
	err = nd.FundContract(c)
	if err != nil {
		return err
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = c.CoinType | 1<<31
	kg.Step[2] = UseContractPayout
	kg.Step[3] = c.PeerIdx | 1<<31
	kg.Step[4] = uint32(c.Idx) | 1<<31

	c.OurPayoutPub, err = nd.GetUsePub(kg, UseContractPayout)
	if err != nil {
		return err
	}

	// Now we can calculate the funding TX, all inputs are known
	tx, err := nd.BuildDlcFundingTransaction(c)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	tx.Serialize(w)
	w.Flush()
	fmt.Printf("Funding TX created:\n%x", buf.Bytes())

	// Now we can sign the division
	sigs, err := nd.SignSettlementDivisions(tx.TxHash(), c)
	if err != nil {
		return err
	}

	msg := lnutil.NewDlcOfferAcceptMsg(c, sigs)
	c.Status = lnutil.ContractStatusAccepted

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.OmniOut <- msg

	return nil
}

func (nd *LitNode) DlcOfferHandler(msg lnutil.DlcOfferMsg, peer *RemotePeer) {
	c := new(lnutil.DlcContract)

	c.PeerIdx = peer.Idx
	c.Status = lnutil.ContractStatusOfferedToMe
	// Reverse copy from the contract we received
	c.OurFundingAmount = msg.Contract.TheirFundingAmount
	c.TheirFundingAmount = msg.Contract.OurFundingAmount
	c.OurFundingInputs = msg.Contract.TheirFundingInputs
	c.TheirFundingInputs = msg.Contract.OurFundingInputs
	c.OurPayoutPub = msg.Contract.TheirPayoutPub
	c.TheirPayoutPub = msg.Contract.OurPayoutPub
	c.ValueAllOurs = msg.Contract.ValueAllTheirs
	c.ValueAllTheirs = msg.Contract.ValueAllOurs
	c.OurChangePKH = msg.Contract.TheirChangePKH
	c.TheirChangePKH = msg.Contract.OurChangePKH

	// Copy
	c.CoinType = msg.Contract.CoinType
	c.OracleA = msg.Contract.OracleA
	c.OracleR = msg.Contract.OracleR
	c.OracleTimestamp = msg.Contract.OracleTimestamp
	c.PubKey = msg.Contract.PubKey

	err := nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcOfferHandler SaveContract err %s\n", err.Error())
		return
	}
}

func (nd *LitNode) DlcDeclineHandler(msg lnutil.DlcOfferDeclineMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.FindContractByKey(msg.ContractPubKey)
	if err != nil {
		fmt.Printf("DlcDeclineHandler FindContract err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusDeclined
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcDeclineHandler SaveContract err %s\n", err.Error())
		return
	}
}

func (nd *LitNode) DlcAcceptHandler(msg lnutil.DlcOfferAcceptMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.FindContractByKey(msg.ContractPubKey)
	if err != nil {
		fmt.Printf("DlcAcceptHandler FindContract err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusAccepted
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcAcceptHandler SaveContract err %s\n", err.Error())
		return
	}

}

func (nd *LitNode) SignSettlementDivisions(txId chainhash.Hash, c *lnutil.DlcContract) ([]lnutil.DlcContractSettlementSignature, error) {
	return []lnutil.DlcContractSettlementSignature{}, nil
}

func (nd *LitNode) BuildDlcFundingTransaction(c *lnutil.DlcContract) (wire.MsgTx, error) {

	// make the tx
	tx := wire.NewMsgTx()

	w, ok := nd.SubWallet[c.CoinType]
	if !ok {
		err := fmt.Errorf("BuildDlcFundingTransaction err no wallet for type %d", c.CoinType)
		return *tx, err
	}

	// set version 2, for op_csv
	tx.Version = 2
	// set the time, the way core does.
	tx.LockTime = uint32(w.CurrentHeight())

	// add all the txins
	var ourInputTotal int64
	var theirInputTotal int64

	for _, u := range c.OurFundingInputs {
		tx.AddTxIn(wire.NewTxIn(&u.Outpoint, nil, nil))
		ourInputTotal += u.Value
	}
	for _, u := range c.TheirFundingInputs {
		tx.AddTxIn(wire.NewTxIn(&u.Outpoint, nil, nil))
		theirInputTotal += u.Value
	}

	// get txo for channel
	txo, err := lnutil.FundTxOut(c.TheirPayoutPub, c.OurPayoutPub, c.OurFundingAmount+c.TheirFundingAmount)
	if err != nil {
		return *tx, err
	}
	tx.AddTxOut(txo)
	tx.AddTxOut(wire.NewTxOut(theirInputTotal-c.TheirFundingAmount-500, lnutil.DirectWPKHScriptFromPKH(c.TheirChangePKH)))
	tx.AddTxOut(wire.NewTxOut(ourInputTotal-c.OurFundingAmount-500, lnutil.DirectWPKHScriptFromPKH(c.OurChangePKH)))

	txsort.InPlaceSort(tx)

	return *tx, nil

}

func (nd *LitNode) FundContract(c *lnutil.DlcContract) error {
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		return fmt.Errorf("No wallet of type %d connected", c.CoinType)
	}

	utxos, _, err := wal.PickUtxos(int64(c.OurFundingAmount), 500, wal.Fee(), true)
	if err != nil {
		return err
	}

	c.OurFundingInputs = make([]lnutil.DlcContractFundingInput, len(utxos))
	for i := 0; i < len(utxos); i++ {
		c.OurFundingInputs[i] = lnutil.DlcContractFundingInput{utxos[i].Op, utxos[i].Value}
	}

	c.OurChangePKH, err = wal.NewAdr()
	if err != nil {
		return err
	}

	return nil
}
