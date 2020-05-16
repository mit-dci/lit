package qln

import (
	"fmt"
	"errors"

	"encoding/hex"

	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/txscript"
	"github.com/mit-dci/lit/btcutil/txsort"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/dlc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/sig64"
	"github.com/mit-dci/lit/wire"
	"github.com/mit-dci/lit/consts"
)

var _,_ = hex.DecodeString("")

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

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot offer a contract to someone that is not in draft stage")
	}

	if !nd.ConnectedToPeer(peerIdx) {
		return fmt.Errorf("You are not connected to peer %d, do that first", peerIdx)
	}

	var nullBytes [33]byte
	// Check if everything's set


	if c.OraclesNumber == dlc.ORACLESNUMBER_NOT_SET {
		return fmt.Errorf("You need to set an oracles number for the contract before offering it")
	}

	if c.OraclesNumber > consts.MaxOraclesNumber {
		return fmt.Errorf("The number of oracles have to be less than 8.")
	}	

	for o := uint32(0); o < c.OraclesNumber; o++ {

		if c.OracleA[o] == nullBytes {
			return fmt.Errorf("You need to set all %d oracls for the contract before offering it", c.OraclesNumber)
		}
	
		if c.OracleR[o] == nullBytes {
			return fmt.Errorf("You need to set all %d R-points for the contract before offering it", c.OraclesNumber)
		}		
		
	}

	if c.OracleTimestamp == 0 {
		return fmt.Errorf("You need to set a settlement time for the contract before offering it")
	}

	if c.RefundTimestamp == 0 {
		return fmt.Errorf("You need to set a refund time for the contract before offering it")
	}	

	if c.CoinType == dlc.COINTYPE_NOT_SET {
		return fmt.Errorf("You need to set a coin type for the contract before offering it")
	}

	if c.FeePerByte == dlc.FEEPERBYTE_NOT_SET {
		return fmt.Errorf("You need to set a fee per byte for the contract before offering it")
	}		

	if c.Division == nil {
		return fmt.Errorf("You need to set a payout division for the contract before offering it")
	}

	if c.OurFundingAmount+c.TheirFundingAmount == 0 {
		return fmt.Errorf("You need to set a funding amount for the peers in contract before offering it")
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

	c.OurPayoutBase, err = nd.GetUsePub(kg, UseContractPayoutBase)
	if err != nil {
		return err
	}

    ourPayoutPKHKey, err := nd.GetUsePub(kg, UseContractPayoutPKH)
    if err != nil {
        logging.Errorf("Error while getting our payout pubkey: %s", err.Error())
        c.Status = lnutil.ContractStatusError
        nd.DlcManager.SaveContract(c)
        return err
	}

	copy(c.OurPayoutPKH[:], btcutil.Hash160(ourPayoutPKHKey[:]))
	
	// Fund the contract
	err = nd.FundContract(c)
	if err != nil {
		return err
	}

	wal, _ := nd.SubWallet[c.CoinType]
	c.OurRefundPKH, err = wal.NewAdr()
	msg := lnutil.NewDlcOfferMsg(peerIdx, c)

	c.Status = lnutil.ContractStatusOfferedByMe
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.tmpSendLitMsg(msg)

	return nil
}

func (nd *LitNode) DeclineDlc(cIdx uint64, reason uint8) error {
	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusOfferedToMe {
		return fmt.Errorf("You cannot decline a contract unless it is in the 'Offered/Awaiting reply' state")
	}

	if !nd.ConnectedToPeer(c.PeerIdx) {
		return fmt.Errorf("You are not connected to peer %d, do that first", c.PeerIdx)
	}

	msg := lnutil.NewDlcOfferDeclineMsg(c.PeerIdx, reason, c.TheirIdx)
	c.Status = lnutil.ContractStatusDeclined

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.tmpSendLitMsg(msg)

	return nil
}

func (nd *LitNode) AcceptDlc(cIdx uint64) error {
	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusOfferedToMe {
		return fmt.Errorf("You cannot accept a contract unless it is in the 'Offered/Awaiting reply' state")
	}

	if !nd.ConnectedToPeer(c.PeerIdx) {
		return fmt.Errorf("You are not connected to peer %d, do that first", c.PeerIdx)
	}

	// Preconditions checked - Go execute the acceptance in a separate go routine
	// while returning the status back to the client
	go func(nd *LitNode, c *lnutil.DlcContract) {
		c.Status = lnutil.ContractStatusAccepting
		nd.DlcManager.SaveContract(c)

		// Fund the contract
		err = nd.FundContract(c)
		if err != nil {
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
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
			logging.Errorf("Error while getting multisig pubkey: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}

		c.OurPayoutBase, err = nd.GetUsePub(kg, UseContractPayoutBase)
		if err != nil {
			logging.Errorf("Error while getting payoutbase: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}

		ourPayoutPKHKey, err := nd.GetUsePub(kg, UseContractPayoutPKH)
		if err != nil {
			logging.Errorf("Error while getting our payout pubkey: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}

		copy(c.OurPayoutPKH[:], btcutil.Hash160(ourPayoutPKHKey[:]))

		wal, _ := nd.SubWallet[c.CoinType]
		c.OurRefundPKH, err = wal.NewAdr()

		// Now we can sign the division
		sigs, err := nd.SignSettlementDivisions(c)
		if err != nil {
			logging.Errorf("Error signing settlement divisions: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}

		refundTx, err := lnutil.RefundTx(c)
		if err != nil {
			logging.Errorf("Error of RefundTx: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}
		
		kg.Step[2] = UseContractFundMultisig
		mypriv, err := wal.GetPriv(kg)
		
		err = lnutil.SignRefundTx(c, refundTx, mypriv)
		if err != nil {
			logging.Errorf("Error of SignRefundTx: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}		
	
		msg := lnutil.NewDlcOfferAcceptMsg(c, sigs)
		c.Status = lnutil.ContractStatusAccepted

		nd.DlcManager.SaveContract(c)
		nd.tmpSendLitMsg(msg)
	}(nd, c)
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
	c.OurPayoutBase = msg.Contract.TheirPayoutBase
	c.TheirPayoutBase = msg.Contract.OurPayoutBase
	c.OurChangePKH = msg.Contract.TheirChangePKH
	c.TheirChangePKH = msg.Contract.OurChangePKH
	c.TheirIdx = msg.Contract.Idx
	c.TheirPayoutPKH = msg.Contract.OurPayoutPKH
	c.TheirRefundPKH = msg.Contract.OurRefundPKH

	c.Division = make([]lnutil.DlcContractDivision, len(msg.Contract.Division))
	for i := 0; i < len(msg.Contract.Division); i++ {
		c.Division[i].OracleValue = msg.Contract.Division[i].OracleValue
		c.Division[i].ValueOurs = (c.TheirFundingAmount + c.OurFundingAmount) - msg.Contract.Division[i].ValueOurs
	}

	// Copy
	c.CoinType = msg.Contract.CoinType
	c.FeePerByte = msg.Contract.FeePerByte

	c.OraclesNumber = msg.Contract.OraclesNumber

	for i:=uint32(0); i < c.OraclesNumber; i++ {

		c.OracleA[i] = msg.Contract.OracleA[i]
		c.OracleR[i] = msg.Contract.OracleR[i]

	}

	c.OracleTimestamp = msg.Contract.OracleTimestamp
	c.RefundTimestamp = msg.Contract.RefundTimestamp

	err := nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcOfferHandler SaveContract err %s\n", err.Error())
		return
	}

	_, ok := nd.SubWallet[msg.Contract.CoinType]
	if !ok {
		// We don't have this coin type, automatically decline
		nd.DeclineDlc(c.Idx, 0x02)
	}

}

func (nd *LitNode) DlcDeclineHandler(msg lnutil.DlcOfferDeclineMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		logging.Errorf("DlcDeclineHandler FindContract err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusDeclined
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcDeclineHandler SaveContract err %s\n", err.Error())
		return
	}
}

func (nd *LitNode) DlcAcceptHandler(msg lnutil.DlcOfferAcceptMsg, peer *RemotePeer) error {
	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		logging.Errorf("DlcAcceptHandler FindContract err %s\n", err.Error())
		return err
	}

	// TODO: Check signatures

	c.TheirChangePKH = msg.OurChangePKH
	c.TheirFundingInputs = msg.FundingInputs
	c.TheirSettlementSignatures = msg.SettlementSignatures
	c.TheirFundMultisigPub = msg.OurFundMultisigPub
	c.TheirPayoutBase = msg.OurPayoutBase
	c.TheirPayoutPKH = msg.OurPayoutPKH
	c.TheirIdx = msg.OurIdx
	c.TheirRefundPKH = msg.OurRefundPKH
	c.TheirrefundTxSig64 = msg.OurrefundTxSig64


	c.Status = lnutil.ContractStatusAccepted
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcAcceptHandler SaveContract err %s\n", err.Error())
		return err
	}

	// create our settlement signatures and ack
	sigs, err := nd.SignSettlementDivisions(c)
	if err != nil {
		return err
	}

	wal, _ := nd.SubWallet[c.CoinType]

	refundTx, err := lnutil.RefundTx(c)
	if err != nil {
		logging.Errorf("Error of RefundTx: %s", err.Error())
		c.Status = lnutil.ContractStatusError
		nd.DlcManager.SaveContract(c)
		return err
	}

	
	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = c.CoinType | 1<<31
	kg.Step[2] = UseContractFundMultisig
	kg.Step[3] = c.PeerIdx | 1<<31
	kg.Step[4] = uint32(c.Idx) | 1<<31

	mypriv, err := wal.GetPriv(kg)

	err = lnutil.SignRefundTx(c, refundTx, mypriv)
	if err != nil {
		logging.Errorf("Error of SignRefundTx: %s", err.Error())
		c.Status = lnutil.ContractStatusError
		nd.DlcManager.SaveContract(c)
		return err
	}

	outMsg := lnutil.NewDlcContractAckMsg(c, sigs, c.OurrefundTxSig64)
	c.Status = lnutil.ContractStatusAcknowledged

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}
	nd.tmpSendLitMsg(outMsg)

	return nil
}

func (nd *LitNode) DlcContractAckHandler(msg lnutil.DlcContractAckMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		logging.Errorf("DlcContractAckHandler FindContract err %s\n", err.Error())
		return
	}

	// TODO: Check signatures

	c.Status = lnutil.ContractStatusAcknowledged
	c.TheirSettlementSignatures = msg.SettlementSignatures
	c.TheirrefundTxSig64 = msg.OurrefundTxSig64

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcContractAckHandler SaveContract err %s\n", err.Error())
		return
	}

	// We have everything now, send our signatures to the funding TX
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		logging.Errorf("DlcContractAckHandler No wallet for cointype %d\n", c.CoinType)
		return
	}

	tx, err := nd.BuildDlcFundingTransaction(c)
	if err != nil {
		logging.Errorf("DlcContractAckHandler BuildDlcFundingTransaction err %s\n", err.Error())
		return
	}

	err = wal.SignMyInputs(&tx)
	if err != nil {
		logging.Errorf("DlcContractAckHandler SignMyInputs err %s\n", err.Error())
		return
	}

	outMsg := lnutil.NewDlcContractFundingSigsMsg(c, &tx)

	nd.tmpSendLitMsg(outMsg)
}

func (nd *LitNode) DlcFundingSigsHandler(msg lnutil.DlcContractFundingSigsMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		logging.Errorf("DlcFundingSigsHandler FindContract err %s\n", err.Error())
		return
	}

	// TODO: Check signatures

	// We have everything now. Sign our inputs to the funding TX and send it to the blockchain.
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		logging.Errorf("DlcFundingSigsHandler No wallet for cointype %d\n", c.CoinType)
		return
	}

	wal.SignMyInputs(msg.SignedFundingTx)
	wal.DirectSendTx(msg.SignedFundingTx)

	err = wal.WatchThis(c.FundingOutpoint)
	if err != nil {
		logging.Errorf("DlcFundingSigsHandler WatchThis err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusActive
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcFundingSigsHandler SaveContract err %s\n", err.Error())
		return
	}

	outMsg := lnutil.NewDlcContractSigProofMsg(c, msg.SignedFundingTx)

	nd.tmpSendLitMsg(outMsg)
}

func (nd *LitNode) DlcSigProofHandler(msg lnutil.DlcContractSigProofMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		logging.Errorf("DlcSigProofHandler FindContract err %s\n", err.Error())
		return
	}

	// TODO: Check signatures
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		logging.Errorf("DlcSigProofHandler No wallet for cointype %d\n", c.CoinType)
		return
	}

	err = wal.WatchThis(c.FundingOutpoint)
	if err != nil {
		logging.Errorf("DlcSigProofHandler WatchThis err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusActive
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcSigProofHandler SaveContract err %s\n", err.Error())
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

	priv, err := wal.GetPriv(kg)
	if err != nil {
		return nil, fmt.Errorf("Could not get private key for contract %d", c.Idx)
	}

	fundingTx, err := nd.BuildDlcFundingTransaction(c)
	if err != nil {
		return nil, err
	}

	c.FundingOutpoint = wire.OutPoint{Hash: fundingTx.TxHash(), Index: 0}

	returnValue := make([]lnutil.DlcContractSettlementSignature, len(c.Division))
	for i, d := range c.Division {
		tx, err := lnutil.SettlementTx(c, d, true)
		if err != nil {
			return nil, err
		}

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

	our_txin_num := 0
	for _, u := range c.OurFundingInputs {
		txin := wire.NewTxIn(&u.Outpoint, nil, nil)

		tx.AddTxIn(txin)
		ourInputTotal += u.Value
		our_txin_num += 1

	}


	their_txin_num := 0
	for _, u := range c.TheirFundingInputs {
		txin := wire.NewTxIn(&u.Outpoint, nil, nil)

		tx.AddTxIn(txin)
		theirInputTotal += u.Value
		their_txin_num += 1

	}


	// Here can be a situation when peers have different number of inputs.
	// Therefore we have to calculate fees for each peer separately.

	// This transaction always will have 3 outputs ( 43 + 31 + 31)
	tx_basesize := 10 + 43 + 31 + 31
	tx_size_foreach := tx_basesize / 2
	tx_size_foreach += 1 // rounding

	input_wit_size := 107

	our_tx_vsize := uint32(((tx_size_foreach + (41 * our_txin_num)) * 3 + (tx_size_foreach + (41 * our_txin_num) + (input_wit_size*our_txin_num) )) / 4)
	their_tx_vsize := uint32(((tx_size_foreach + (41 * their_txin_num)) * 3 + (tx_size_foreach + (41 * their_txin_num) + (input_wit_size*their_txin_num) )) / 4)

	//rounding
	our_tx_vsize += 1
	their_tx_vsize += 1


	our_fee := int64(our_tx_vsize * c.FeePerByte)
	their_fee := int64(their_tx_vsize * c.FeePerByte)

	// add change and sort
	their_txout := wire.NewTxOut(theirInputTotal-c.TheirFundingAmount-their_fee, lnutil.DirectWPKHScriptFromPKH(c.TheirChangePKH)) 
	tx.AddTxOut(their_txout)

	our_txout := wire.NewTxOut(ourInputTotal-c.OurFundingAmount-our_fee, lnutil.DirectWPKHScriptFromPKH(c.OurChangePKH))
	tx.AddTxOut(our_txout)

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
		c.OurFundingInputs[i] = lnutil.DlcContractFundingInput{Outpoint: utxos[i].Op, Value: utxos[i].Value}
	}

	c.OurChangePKH, err = wal.NewAdr()
	if err != nil {
		return err
	}

	return nil
}

func (nd *LitNode) SettleContract(cIdx uint64, oracleValue int64, oraclesSig[consts.MaxOraclesNumber][32]byte) ([32]byte, [32]byte, error) {

	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		logging.Errorf("SettleContract FindContract err %s\n", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	c.Status = lnutil.ContractStatusSettling
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("SettleContract SaveContract err %s\n", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	d, err := c.GetDivision(oracleValue)
	if err != nil {
		logging.Errorf("SettleContract GetDivision err %s\n", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		return [32]byte{}, [32]byte{}, fmt.Errorf("SettleContract Wallet of type %d not found", c.CoinType)
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = c.CoinType | 1<<31
	kg.Step[2] = UseContractFundMultisig
	kg.Step[3] = c.PeerIdx | 1<<31
	kg.Step[4] = uint32(c.Idx) | 1<<31

	priv, err := wal.GetPriv(kg)
	if err != nil {
		return [32]byte{}, [32]byte{}, fmt.Errorf("SettleContract Could not get private key for contract %d", c.Idx)
	}

	settleTx, err := lnutil.SettlementTx(c, *d, false)
	if err != nil {
		logging.Errorf("SettleContract SettlementTx err %s\n", err.Error())
		return [32]byte{}, [32]byte{}, err
	}


	mySig, err := nd.SignSettlementTx(c, settleTx, priv)
	if err != nil {
		logging.Errorf("SettleContract SignSettlementTx err %s", err.Error())
		return [32]byte{}, [32]byte{}, err
	}
	

	myBigSig := sig64.SigDecompress(mySig)
	
	theirSig, err := c.GetTheirSettlementSignature(oracleValue)
	theirBigSig := sig64.SigDecompress(theirSig)

	// put the sighash all byte on the end of both signatures
	myBigSig = append(myBigSig, byte(txscript.SigHashAll))
	theirBigSig = append(theirBigSig, byte(txscript.SigHashAll))

	pre, swap, err := lnutil.FundTxScript(c.OurFundMultisigPub, c.TheirFundMultisigPub)
	if err != nil {
		logging.Errorf("SettleContract FundTxScript err %s", err.Error())
		return [32]byte{}, [32]byte{}, err
	}
		

	// swap if needed
	if swap {
		settleTx.TxIn[0].Witness = SpendMultiSigWitStack(pre, theirBigSig, myBigSig)
	} else {
		settleTx.TxIn[0].Witness = SpendMultiSigWitStack(pre, myBigSig, theirBigSig)
	}

	// Settlement TX should be valid here, so publish it.
	err = wal.DirectSendTx(settleTx)
	if err != nil {
		logging.Errorf("SettleContract DirectSendTx (settle) err %s", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	//===========================================
	// Claim TX
	//===========================================


	// Here the transaction size is always the same
	// n := 8 + VarIntSerializeSize(uint64(len(msg.TxIn))) +
	// 	VarIntSerializeSize(uint64(len(msg.TxOut)))
	// n = 10
	// Plus Single input 41
	// Plus Single output 31
	// Plus 2 for all wittness transactions
	// Plus Witness Data 151

	// TxSize = 4 + 4 + 1 + 1 + 2 + 151 + 41 + 31 = 235
	// Vsize = ((235 - 151 - 2) * 3 + 235) / 4 = 120,25


	if ( d.ValueOurs != 0){

		vsize := uint32(121)
		fee := vsize * c.FeePerByte
	
		// TODO: Claim the contract settlement output back to our wallet - otherwise the peer can claim it after locktime.
		txClaim := wire.NewMsgTx()
		txClaim.Version = 2

		settleOutpoint := wire.OutPoint{Hash: settleTx.TxHash(), Index: 0}
		txClaim.AddTxIn(wire.NewTxIn(&settleOutpoint, nil, nil))

		addr, err := wal.NewAdr()
		txClaim.AddTxOut(wire.NewTxOut(settleTx.TxOut[0].Value-int64(fee), lnutil.DirectWPKHScriptFromPKH(addr)))

		kg.Step[2] = UseContractPayoutBase
		privSpend, _ := wal.GetPriv(kg)


		var pubOracleBytes [][33]byte
	
		privOracle0, pubOracle0 := koblitz.PrivKeyFromBytes(koblitz.S256(), oraclesSig[0][:])
		privContractOutput := lnutil.CombinePrivateKeys(privSpend, privOracle0)

		var pubOracleBytes0 [33]byte
		copy(pubOracleBytes0[:], pubOracle0.SerializeCompressed())		
		pubOracleBytes = append(pubOracleBytes, pubOracleBytes0)

		for i:=uint32(1); i < c.OraclesNumber; i++ {

			privOracle, pubOracle := koblitz.PrivKeyFromBytes(koblitz.S256(), oraclesSig[i][:])

			privContractOutput = lnutil.CombinePrivateKeys(privContractOutput, privOracle)

			var pubOracleBytes1 [33]byte
			copy(pubOracleBytes1[:], pubOracle.SerializeCompressed())
			pubOracleBytes = append(pubOracleBytes, pubOracleBytes1)

		}

		settleScript := lnutil.DlcCommitScript(c.OurPayoutBase, c.TheirPayoutBase, pubOracleBytes , 5)
		err = nd.SignClaimTx(txClaim, settleTx.TxOut[0].Value, settleScript, privContractOutput, false)
		if err != nil {
			logging.Errorf("SettleContract SignClaimTx err %s", err.Error())
			return [32]byte{}, [32]byte{}, err
		}

		// Claim TX should be valid here, so publish it.
		err = wal.DirectSendTx(txClaim)
		if err != nil {
			logging.Errorf("SettleContract DirectSendTx (claim) err %s", err.Error())
			return [32]byte{}, [32]byte{}, err
		}

		c.Status = lnutil.ContractStatusClosed
		err = nd.DlcManager.SaveContract(c)
		if err != nil {
			return [32]byte{}, [32]byte{}, err
		}
		return settleTx.TxHash(), txClaim.TxHash(), nil

	}else{

		return settleTx.TxHash(), [32]byte{}, nil

	}

}



func (nd *LitNode) RefundContract(cIdx uint64) (bool, error) {

	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		logging.Errorf("SettleContract FindContract err %s\n", err.Error())
		return false, err
	}

	if (c.Status != lnutil.ContractStatusActive) && (c.Status != lnutil.ContractStatusNegotiateDeclinedByHim) {
		return false, errors.New("You cannot refund a contract that is not in active stage")
	}


	wal, _ := nd.SubWallet[c.CoinType]

	refundTx, err := lnutil.RefundTx(c)
	myBigSig := sig64.SigDecompress(c.OurrefundTxSig64)
	myBigSig = append(myBigSig, byte(txscript.SigHashAll))	
	theirBigSig := sig64.SigDecompress(c.TheirrefundTxSig64)
	theirBigSig = append(theirBigSig, byte(txscript.SigHashAll))
	pre, swap, err := lnutil.FundTxScript(c.OurFundMultisigPub, c.TheirFundMultisigPub)

	// swap if needed
	if swap {
		refundTx.TxIn[0].Witness = SpendMultiSigWitStack(pre, theirBigSig, myBigSig)
	} else {
		refundTx.TxIn[0].Witness = SpendMultiSigWitStack(pre, myBigSig, theirBigSig)
	}	

	err = wal.DirectSendTx(refundTx)

	return true, nil

}




func (nd *LitNode) DlcNegotiateContract(cIdx uint64, DesiredOracleValue int64) error {

	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if !nd.ConnectedToPeer(c.PeerIdx) {
		return fmt.Errorf("You are not connected to peer %d, do that first", c.PeerIdx)
	}


	if (c.Status != lnutil.ContractStatusActive) && (c.Status != lnutil.ContractStatusNegotiateDeclinedByHim) {
		return fmt.Errorf("You cannot negotiate a contract that is not in active stage")
	}

	c.DesiredOracleValue = DesiredOracleValue


	//----------------------------------------------------------------------------
	// Create Tx

	var negotiateTx *wire.MsgTx
	negotiateTx, err = lnutil.NegotiateTx(c)

	//----------------------------------------------------------------------------
	// Sign

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = c.CoinType | 1<<31
	kg.Step[2] = UseContractFundMultisig
	kg.Step[3] = c.PeerIdx | 1<<31
	kg.Step[4] = uint32(c.Idx) | 1<<31

	wal, _ := nd.SubWallet[c.CoinType]
	priv, err := wal.GetPriv(kg)

	err = lnutil.SignNegotiateTx(c, negotiateTx, priv)
	if err != nil {
		logging.Errorf("Error of SignRefundTx: %s", err.Error())
		c.Status = lnutil.ContractStatusError
		nd.DlcManager.SaveContract(c)
		return err
	}


	
	outMsg := lnutil.NewDlcContractNegotiateMsg(c)

	nd.tmpSendLitMsg(outMsg)


	c.Status = lnutil.ContractStatusNegotiatingByMe


	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcNegotiateContractHandler SaveContract err %s\n", err.Error())
		return err
	}	

	return nil

}


// DlcNegotiateContractHandler
func (nd *LitNode) DlcNegotiateContractHandler(msg lnutil.DlcContractNegotiateMsg, peer *RemotePeer) error {
	//Тут он принимает запрос. Т.е просто видит что он есть.
	//Он получает DlcNegotiateMsg. Там подпись для DesiredOracleValue


	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		logging.Errorf("DlcFundingSigsHandler FindContract err %s\n", err.Error())
		return err
	}	

	c.TheirnegotiateTxSig64 = msg.OurnegotiateTxSig64

	// TODO: wire.ReadVarInt(buf, 0) returns only uint64, why?
	c.DesiredOracleValue = int64(msg.DesiredOracleValue)

	//-------------------------
	//Verify

	theirBigSig := sig64.SigDecompress(c.TheirnegotiateTxSig64)

	var negotiateTx *wire.MsgTx
	negotiateTx, err = lnutil.NegotiateTx(c)

	hCache := txscript.NewTxSigHashes(negotiateTx)

	pre, _, err := lnutil.FundTxScript(c.OurFundMultisigPub, c.TheirFundMultisigPub)
	if err != nil {
		return err
	}

	parsed, err := txscript.ParseScript(pre)
	if err != nil {
		logging.Errorf("DlcNegotiateContractHandler Sig err %s", err.Error())
		return err
	}	

	// always sighash all
	hash := txscript.CalcWitnessSignatureHash(parsed, hCache, txscript.SigHashAll, negotiateTx, 0, c.OurFundingAmount+c.TheirFundingAmount)

	theirparsedPubKey, parsepuberr := koblitz.ParsePubKey(c.TheirFundMultisigPub[:], koblitz.S256())
	if parsepuberr != nil {
		logging.Errorf("DlcFundingSigsHandler err %s\n", parsepuberr.Error())
	}

	theirparsedSig, parsesigerr := koblitz.ParseDERSignature(theirBigSig, koblitz.S256())
	if parsesigerr != nil {
		logging.Errorf("DlcFundingSigsHandler err %s\n", parsesigerr.Error())
	}

	theirSigvalid := theirparsedSig.Verify(hash, theirparsedPubKey)

	fmt.Printf("DlcNegotiateContractHandler(): c.TheirnegotiateTxSig64 valid?: %t \n", theirSigvalid)


	if theirSigvalid {
		c.Status = lnutil.ContractStatusNegotiatingToMe
	}else{
		return errors.New("DlcNegotiateContractHandler: c.TheirnegotiateTxSig64 is invalid.")
	}

	//-------------------------

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcNegotiateContractHandler SaveContract err %s\n", err.Error())
		return err
	}


	return nil	
}


func (nd *LitNode) DlcAcceptNegotiate(cIdx uint64) error {
	//Тут он может согласится
	//Если он соглашается, то публикует транзакцию и шлет Ack

	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusNegotiatingToMe {
		return fmt.Errorf("You cannot accept negotiate a contract unless it is in the 'Negotiating to me' state")
	}


	if !nd.ConnectedToPeer(c.PeerIdx) {
		return fmt.Errorf("You are not connected to peer %d, do that first", c.PeerIdx)
	}	

	var negotiateTx *wire.MsgTx
	negotiateTx, err = lnutil.NegotiateTx(c)
	
	
	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = c.CoinType | 1<<31
	kg.Step[2] = UseContractFundMultisig
	kg.Step[3] = c.PeerIdx | 1<<31
	kg.Step[4] = uint32(c.Idx) | 1<<31

	wal, _ := nd.SubWallet[c.CoinType]
	priv, err := wal.GetPriv(kg)

	err = lnutil.SignNegotiateTx(c, negotiateTx, priv)
	if err != nil {
		logging.Errorf("Error of SignRefundTx: %s", err.Error())
		c.Status = lnutil.ContractStatusError
		nd.DlcManager.SaveContract(c)
		return err
	}	

	pre, swap, err := lnutil.FundTxScript(c.OurFundMultisigPub, c.TheirFundMultisigPub)
	
	
	myBigSig := sig64.SigDecompress(c.OurnegotiateTxSig64)
	myBigSig = append(myBigSig, byte(txscript.SigHashAll))	
	theirBigSig := sig64.SigDecompress(c.TheirnegotiateTxSig64)
	theirBigSig = append(theirBigSig, byte(txscript.SigHashAll))	

	// swap if needed
	if swap {
		negotiateTx.TxIn[0].Witness = SpendMultiSigWitStack(pre, theirBigSig, myBigSig)
	} else {
		negotiateTx.TxIn[0].Witness = SpendMultiSigWitStack(pre, myBigSig, theirBigSig)
	}	

	err = wal.DirectSendTx(negotiateTx)

	c.Status = lnutil.ContractStatusNegotiatedToMe

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcNegotiateContractHandler SaveContract err %s\n", err.Error())
		return err
	}	

	// Send ACK
	outMsg := lnutil.NewDlcAcceptNegotiateMsg(c, negotiateTx)
	nd.tmpSendLitMsg(outMsg)	


	return nil
}


func (nd *LitNode) DlcAcceptNegotiateAck(msg lnutil.DlcContractAcceptNegotiateMsg, peer *RemotePeer) error {
	//Я вижу что он принял и все

	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		return err
	}

	c.Status = lnutil.ContractStatusNegotiatedByMe

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcAcceptNegotiateAck SaveContract err %s\n", err.Error())
		return err
	}		

	return nil

}



func (nd *LitNode) DlcDeclineNegotiate(cIdx uint64) error {
	//Или отказатся
	//Шлет Ack

	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}
	
	if !nd.ConnectedToPeer(c.PeerIdx) {
		return fmt.Errorf("You are not connected to peer %d, do that first", c.PeerIdx)
	}
	
	
	if c.Status != lnutil.ContractStatusNegotiatingToMe {
		return fmt.Errorf("You cannot decline negotiate a contract unless it is in the 'Negotiating to me' state")
	}



	c.Status = lnutil.ContractStatusNegotiateDeclinedByMe

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcAcceptNegotiateAck SaveContract err %s\n", err.Error())
		return err
	}

	// Send ACK
	outMsg := lnutil.NewDlcContractDeclineNegotiateMsg(c)
	nd.tmpSendLitMsg(outMsg)
	
	return nil

}


func (nd *LitNode) DlcDeclineNegotiateAck(msg lnutil.DlcContractDeclineNegotiateMsg, peer *RemotePeer) error {
	//Я вижу что он отказался

	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		return err
	}	

	c.Status = lnutil.ContractStatusNegotiateDeclinedByHim

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		logging.Errorf("DlcDeclineNegotiateAck SaveContract err %s\n", err.Error())
		return err
	}		

	return nil
}





