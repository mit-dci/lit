package qln

import (
	"fmt"
	"log"

	"github.com/adiabat/btcd/btcec"
	"github.com/adiabat/btcd/txscript"
	"github.com/adiabat/btcd/wire"
	"github.com/adiabat/btcutil/txsort"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/sig64"
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

	c.TheirChangePKH = msg.OurChangePKH
	c.TheirFundingInputs = msg.FundingInputs
	c.TheirSettlementSignatures = msg.SettlementSignatures
	c.TheirFundMultisigPub = msg.OurFundMultisigPub
	c.TheirPayoutPub = msg.OurPayoutPub
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
	fmt.Printf("Received hash: %s\n", msg.SignedFundingTx.TxHash().String())

	ftx, err := nd.BuildDlcFundingTransaction(c)
	if err != nil {
		fmt.Printf("DlcFundingSigsHandler BuildDlcFundingTransaction err %s\n", err.Error())
		return
	}
	fmt.Printf("Self-generated funding TX:\n")
	lnutil.PrintTx(&ftx)
	fmt.Printf("Self-generated hash: %s\n", ftx.TxHash().String())

	wal.SignMyInputs(msg.SignedFundingTx)

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

	fundingTx, err := nd.BuildDlcFundingTransaction(c)
	if err != nil {
		return nil, err
	}

	contractInput := wire.OutPoint{fundingTx.TxHash(), 0}

	returnValue := make([]lnutil.DlcContractSettlementSignature, len(c.Division))
	for i, d := range c.Division {
		tx, err := lnutil.SettlementTx(c, d, contractInput, true)

		sig, err := nd.SignSettlementTx(c, tx, priv)
		if err != nil {
			return nil, err
		}
		returnValue[i].Outcome = d.OracleValue
		returnValue[i].Signature = sig
	}

	return returnValue, nil
}

func (nd *LitNode) BuildDlcFundingTransaction(c *lnutil.DlcContract) (wire.MsgTx, error) {
	// make the tx
	tx := wire.NewMsgTx()

	// set version 2, for op_csv
	tx.Version = 2

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

	// add change and sort
	tx.AddTxOut(wire.NewTxOut(theirInputTotal-c.TheirFundingAmount-500, lnutil.DirectWPKHScriptFromPKH(c.TheirChangePKH)))
	tx.AddTxOut(wire.NewTxOut(ourInputTotal-c.OurFundingAmount-500, lnutil.DirectWPKHScriptFromPKH(c.OurChangePKH)))

	txsort.InPlaceSort(tx)

	// get txo for channel
	txo, err := lnutil.FundTxOut(c.TheirFundMultisigPub, c.OurFundMultisigPub, c.OurFundingAmount+c.TheirFundingAmount)
	if err != nil {
		return *tx, err
	}

	// Ensure contract funding output is always at position 0
	txos := make([]*wire.TxOut, len(tx.TxOut)+1)
	txos[0] = txo
	copy(txos[1:], tx.TxOut)
	tx.TxOut = txos

	fmt.Printf("Returning funding TX: %s\n", tx.TxHash().String())
	lnutil.PrintTx(tx)
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

func (nd *LitNode) SettleContract(cIdx uint64, oracleValue int64, oracleSig [32]byte) error {
	fmt.Printf("Settling contract %d on value %d (sig: %x)\n", cIdx, oracleValue, oracleSig)

	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		fmt.Printf("SettleContract FindContract err %s\n", err.Error())
		return err
	}

	d, err := c.GetDivision(oracleValue)
	if err != nil {
		fmt.Printf("SettleContract GetDivision err %s\n", err.Error())
		return err
	}

	fundingTx, err := nd.BuildDlcFundingTransaction(c)
	if err != nil {
		fmt.Printf("SettleContract BuildDlcFundingTransaction err %s\n", err.Error())
		return err
	}

	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		return fmt.Errorf("SettleContract Wallet of type %d not found", c.CoinType)
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
		return fmt.Errorf("SettleContract Could not get private key for contract %d", c.Idx)
	}

	contractInput := wire.OutPoint{fundingTx.TxHash(), 0}

	settleTx, err := lnutil.SettlementTx(c, *d, contractInput, false)
	if err != nil {
		fmt.Printf("SettleContract SettlementTx err %s\n", err.Error())
		return err
	}

	mySig, err := nd.SignSettlementTx(c, settleTx, priv)
	if err != nil {
		log.Printf("SettleContract SignSettlementTx err %s", err.Error())
		return err
	}

	myBigSig := sig64.SigDecompress(mySig)

	theirSig, err := c.GetTheirSettlementSignature(oracleValue)
	theirBigSig := sig64.SigDecompress(theirSig)

	// put the sighash all byte on the end of both signatures
	myBigSig = append(myBigSig, byte(txscript.SigHashAll))
	theirBigSig = append(theirBigSig, byte(txscript.SigHashAll))

	pre, swap, err := lnutil.FundTxScript(c.OurFundMultisigPub, c.TheirFundMultisigPub)
	if err != nil {
		log.Printf("SettleContract FundTxScript err %s", err.Error())
		return err
	}

	// swap if needed
	if swap {
		settleTx.TxIn[0].Witness = SpendMultiSigWitStack(pre, theirBigSig, myBigSig)
	} else {
		settleTx.TxIn[0].Witness = SpendMultiSigWitStack(pre, myBigSig, theirBigSig)
	}

	fmt.Printf("SettleTX before publish: %s\n", settleTx.TxHash().String())
	lnutil.PrintTx(settleTx)

	// Settlement TX should be valid here, so publish it.
	err = wal.DirectSendTx(settleTx)
	if err != nil {
		log.Printf("SettleContract DirectSendTx (settle) err %s", err.Error())
		return err
	}

	// TODO: Claim the contract settlement output back to our wallet - otherwise the peer can claim it after locktime.
	txClaim := wire.NewMsgTx()
	txClaim.Version = 2

	settleOutpoint := wire.OutPoint{settleTx.TxHash(), 0}
	txClaim.AddTxIn(wire.NewTxIn(&settleOutpoint, nil, nil))

	addr, err := wal.NewAdr()
	txClaim.AddTxOut(wire.NewTxOut(d.ValueOurs-1000, lnutil.DirectWPKHScriptFromPKH(addr))) // todo calc fee - fee is double here because the contract output already had the fee deducted in the settlement TX

	kg.Step[2] = UseContractPayout
	privSpend := wal.GetPriv(kg)
	privOracle, _ := btcec.PrivKeyFromBytes(btcec.S256(), oracleSig[:])
	privContractOutput := lnutil.CombinePrivateKeys(privSpend, privOracle)

	fmt.Printf("ClaimTX before publish: %s\n", txClaim.TxHash().String())
	lnutil.PrintTx(txClaim)

	nd.SignClaimTx(settleTx, txClaim, privContractOutput)
	// Claim TX should be valid here, so publish it.
	err = wal.DirectSendTx(txClaim)
	if err != nil {
		log.Printf("SettleContract DirectSendTx (claim) err %s", err.Error())
		return err
	}

	return nil
}
