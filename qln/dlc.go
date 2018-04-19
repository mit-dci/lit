package qln

import (
	"fmt"

	"github.com/adiabat/btcutil"

	"github.com/adiabat/btcd/btcec"
	"github.com/adiabat/btcd/txscript"
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

	c.PeerIdx = peerIdx

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = c.CoinType | 1<<31
	kg.Step[2] = UseContractFundMultisig
	kg.Step[3] = c.PeerIdx | 1<<31
	kg.Step[4] = uint32(c.Idx) | 1<<31

	c.OurFundMultisigPub, err = nd.GetUsePub(kg, UseContractFundMultisig)
	if err != nil {
		return err
	}

	c.OurPayoutPub, err = nd.GetUsePub(kg, UseContractFundMultisig)
	if err != nil {
		return err
	}

	// Fund the contract
	err = nd.FundContract(c)
	if err != nil {
		return err
	}

	msg := lnutil.NewDlcOfferMsg(peerIdx, c)

	c.Status = lnutil.ContractStatusOfferedByMe
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
	kg.Step[2] = UseContractFundMultisig
	kg.Step[3] = c.PeerIdx | 1<<31
	kg.Step[4] = uint32(c.Idx) | 1<<31

	c.OurFundMultisigPub, err = nd.GetUsePub(kg, UseContractFundMultisig)
	if err != nil {
		return err
	}

	c.OurPayoutPub, err = nd.GetUsePub(kg, UseContractPayout)
	if err != nil {
		return err
	}

	// Now we can sign the division
	sigs, err := nd.SignSettlementDivisions(c)
	if err != nil {
		return err
	}

	fmt.Printf("Sending acceptance. Our funding input length: [%d] - Their funding input length: [%d]", len(c.OurFundingInputs), len(c.TheirFundingInputs))

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
	c.OurFundMultisigPub = msg.Contract.TheirFundMultisigPub
	c.TheirFundMultisigPub = msg.Contract.OurFundMultisigPub
	c.OurPayoutPub = msg.Contract.TheirPayoutPub
	c.TheirPayoutPub = msg.Contract.OurPayoutPub
	c.OurChangePKH = msg.Contract.TheirChangePKH
	c.TheirChangePKH = msg.Contract.OurChangePKH

	c.Division = make([]lnutil.DlcContractDivision, len(msg.Contract.Division))
	for i := 0; i < len(msg.Contract.Division); i++ {
		c.Division[i].OracleValue = msg.Contract.Division[i].OracleValue
		c.Division[i].ValueOurs = (c.TheirFundingAmount + c.OurFundingAmount) - msg.Contract.Division[i].ValueOurs
	}

	// Copy
	c.CoinType = msg.Contract.CoinType
	c.OracleA = msg.Contract.OracleA
	c.OracleR = msg.Contract.OracleR
	c.OracleTimestamp = msg.Contract.OracleTimestamp
	c.PubKey = msg.Contract.PubKey

	fmt.Printf("Received contract offer. Funding input length: [%d]\n", len(c.TheirFundingInputs))

	err := nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcOfferHandler SaveContract err %s\n", err.Error())
		return
	}

	c, err = nd.DlcManager.FindContractByKey(msg.Contract.PubKey)
	if err != nil {
		fmt.Printf("DlcOfferHandler FindContract err %s\n", err.Error())
		return
	}
	fmt.Printf("DlcOfferHandler after reloading - Their inputs [%d]\n", len(c.TheirFundingInputs))

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

func (nd *LitNode) DlcAcceptHandler(msg lnutil.DlcOfferAcceptMsg, peer *RemotePeer) error {
	c, err := nd.DlcManager.FindContractByKey(msg.ContractPubKey)
	if err != nil {
		fmt.Printf("DlcAcceptHandler FindContract err %s\n", err.Error())
		return err
	}

	// TODO: Check signatures

	c.TheirChangePKH = msg.MyChangePKH
	c.TheirFundingInputs = msg.FundingInputs
	c.TheirSettlementSignatures = msg.SettlementSignatures

	fmt.Printf("DlcAcceptHandler - Our inputs [%d] - Their inputs [%d]\n", len(c.OurFundingInputs), len(c.TheirFundingInputs))

	c.Status = lnutil.ContractStatusAccepted
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcAcceptHandler SaveContract err %s\n", err.Error())
		return err
	}

	// create our settlement signatures and ack
	sigs, err := nd.SignSettlementDivisions(c)
	if err != nil {
		return err
	}

	outMsg := lnutil.NewDlcContractAckMsg(c, sigs)
	c.Status = lnutil.ContractStatusAcknowledged

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.OmniOut <- outMsg

	return nil

}

func (nd *LitNode) DlcContractAckHandler(msg lnutil.DlcContractAckMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.FindContractByKey(msg.ContractPubKey)
	if err != nil {
		fmt.Printf("DlcContractAckHandler FindContract err %s\n", err.Error())
		return
	}

	// TODO: Check signatures

	c.Status = lnutil.ContractStatusAcknowledged

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcContractAckHandler SaveContract err %s\n", err.Error())
		return
	}

	// We have everything now, send our signatures to the funding TX
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		fmt.Printf("DlcContractAckHandler No wallet for cointype %d\n", c.CoinType)
		return
	}

	tx, err := nd.BuildDlcFundingTransaction(c)
	if err != nil {
		fmt.Printf("DlcContractAckHandler BuildDlcFundingTransaction err %s\n", err.Error())
		return
	}

	err = wal.SignMyInputs(&tx)
	if err != nil {
		fmt.Printf("DlcContractAckHandler SignMyInputs err %s\n", err.Error())
		return
	}

	fmt.Printf("My funding TX:\n")
	lnutil.PrintTx(&tx)

	outMsg := lnutil.NewDlcContractFundingSigsMsg(c, &tx)

	nd.OmniOut <- outMsg
}

func (nd *LitNode) DlcFundingSigsHandler(msg lnutil.DlcContractFundingSigsMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.FindContractByKey(msg.ContractPubKey)
	if err != nil {
		fmt.Printf("DlcFundingSigsHandler FindContract err %s\n", err.Error())
		return
	}

	// TODO: Check signatures

	// We have everything now. Sign our inputs to the funding TX and send it to the blockchain.
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		fmt.Printf("DlcFundingSigsHandler No wallet for cointype %d\n", c.CoinType)
		return
	}

	fmt.Printf("Received funding TX:\n")
	lnutil.PrintTx(msg.SignedFundingTx)

	wal.SignMyInputs(msg.SignedFundingTx)

	fmt.Printf("Signed funding TX:\n")
	lnutil.PrintTx(msg.SignedFundingTx)

	wal.DirectSendTx(msg.SignedFundingTx)

	c.Status = lnutil.ContractStatusActive
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcFundingSigsHandler SaveContract err %s\n", err.Error())
		return
	}

	outMsg := lnutil.NewDlcContractSigProofMsg(c, msg.SignedFundingTx)

	nd.OmniOut <- outMsg
}

func (nd *LitNode) DlcSigProofHandler(msg lnutil.DlcContractSigProofMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.FindContractByKey(msg.ContractPubKey)
	if err != nil {
		fmt.Printf("DlcSigProofHandler FindContract err %s\n", err.Error())
		return
	}

	// TODO: Check signatures

	c.Status = lnutil.ContractStatusActive
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcSigProofHandler SaveContract err %s\n", err.Error())
		return
	}
}

func (nd *LitNode) SignSettlementDivisions(c *lnutil.DlcContract) ([]lnutil.DlcContractSettlementSignature, error) {
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		return nil, fmt.Errorf("Wallet of type %d not found", c.CoinType)
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = c.CoinType | 1<<31
	kg.Step[2] = UseContractFundMultisig
	kg.Step[3] = c.PeerIdx | 1<<31
	kg.Step[4] = uint32(c.Idx) | 1<<31

	priv := wal.GetPriv(kg)
	if priv == nil {
		return nil, fmt.Errorf("Could not get private key for contract %d", c.Idx)
	}

	totalContractValue := c.OurFundingAmount + c.TheirFundingAmount
	var ourPayoutPKH [20]byte
	copy(ourPayoutPKH[:], btcutil.Hash160(c.OurPayoutPub[:]))

	fundingTx, err := nd.BuildDlcFundingTransaction(c)
	if err != nil {
		return nil, err
	}

	contractInput := wire.OutPoint{fundingTx.TxHash(), 0}

	returnValue := make([]lnutil.DlcContractSettlementSignature, len(c.Division))
	for i, d := range c.Division {
		tx := wire.NewMsgTx()
		// set version 2, for op_csv
		tx.Version = 2

		tx.AddTxIn(wire.NewTxIn(&contractInput, nil, nil))
		totalFee := int64(1000) // TODO: Calculate
		feeEach := int64(float64(totalFee) / float64(2))
		feeOurs := feeEach
		feeTheirs := feeEach
		valueOurs := d.ValueOurs
		// We don't have enough to pay for a fee. We get 0, our contract partner pays the rest of the fee
		if valueOurs < feeEach {
			feeOurs = valueOurs
			valueOurs = 0
		} else {
			valueOurs = d.ValueOurs - feeOurs
		}

		valueTheirs := totalContractValue - d.ValueOurs

		if valueTheirs < feeEach {
			feeTheirs = valueTheirs
			valueTheirs = 0
			feeOurs = totalFee - feeTheirs
			valueOurs = d.ValueOurs - feeOurs
		} else {
			valueTheirs -= feeTheirs
		}

		if valueTheirs > 0 {
			tx.AddTxOut(lnutil.DlcOutput(c.TheirPayoutPub, c.TheirPayoutPub, c.OurPayoutPub, valueTheirs-(totalFee-feeOurs)))
		}

		if valueOurs > 0 {
			tx.AddTxOut(wire.NewTxOut(valueOurs, lnutil.DirectWPKHScriptFromPKH(ourPayoutPKH)))
		}

		sig, err := nd.SignSettlementTx(tx, &fundingTx, priv)
		if err != nil {
			return nil, err
		}
		returnValue[i].Outcome = d.OracleValue
		returnValue[i].Signature = sig
	}

	return returnValue, nil
}

func (nd *LitNode) SignSettlementTx(txSettle, txFund *wire.MsgTx, priv *btcec.PrivateKey) ([]byte, error) {

	hCache := txscript.NewTxSigHashes(txSettle)
	sig, err := txscript.RawTxInWitnessSignature(txSettle, hCache, 0,
		txFund.TxOut[0].Value, txFund.TxOut[0].PkScript, txscript.SigHashAll, priv)
	if err != nil {
		return []byte{}, err
	}
	return sig, nil

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

	fmt.Printf("Building funding TX. Our input len: [%d]. Their input len: [%d]\n", len(c.OurFundingInputs), len(c.TheirFundingInputs))

	for _, u := range c.OurFundingInputs {
		fmt.Printf("Adding our outpoint %s\n", u.Outpoint.String())
		tx.AddTxIn(wire.NewTxIn(&u.Outpoint, nil, nil))
		ourInputTotal += u.Value
	}
	for _, u := range c.TheirFundingInputs {
		fmt.Printf("Adding their outpoint %s\n", u.Outpoint.String())
		tx.AddTxIn(wire.NewTxIn(&u.Outpoint, nil, nil))
		theirInputTotal += u.Value
	}

	// get txo for channel
	txo, err := lnutil.FundTxOut(c.TheirFundMultisigPub, c.OurFundMultisigPub, c.OurFundingAmount+c.TheirFundingAmount)
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
