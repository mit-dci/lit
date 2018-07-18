package qln

import (
	"fmt"
	"log"

	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/txscript"
	"github.com/mit-dci/lit/btcutil/txsort"
	"github.com/mit-dci/lit/dlc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/sig64"
	"github.com/mit-dci/lit/wire"
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

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot offer a contract to someone that is not in draft stage")
	}

	if !nd.ConnectedToPeer(peerIdx) {
		return fmt.Errorf("You are not connected to peer %d, do that first", peerIdx)
	}

	var nullBytes [33]byte
	// Check if everything's set
	if c.OracleA == nullBytes {
		return fmt.Errorf("You need to set an oracle for the contract before offering it")
	}

	if c.OracleR == nullBytes {
		return fmt.Errorf("You need to set an R-point for the contract before offering it")
	}

	if c.OracleTimestamp == 0 {
		return fmt.Errorf("You need to set a settlement time for the contract before offering it")
	}

	if c.CoinType == dlc.COINTYPE_NOT_SET {
		return fmt.Errorf("You need to set a coin type for the contract before offering it")
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

	nd.OmniOut <- msg

	return nil
}

func (nd *LitNode) AcceptDlc(cIdx uint64) error {
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
			log.Printf("Error while getting multisig pubkey: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}

		c.OurPayoutBase, err = nd.GetUsePub(kg, UseContractPayoutBase)
		if err != nil {
			log.Printf("Error while getting payoutbase: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}

		ourPayoutPKHKey, err := nd.GetUsePub(kg, UseContractPayoutPKH)
		if err != nil {
			log.Printf("Error while getting our payout pubkey: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}
		copy(c.OurPayoutPKH[:], btcutil.Hash160(ourPayoutPKHKey[:]))

		// Now we can sign the division
		sigs, err := nd.SignSettlementDivisions(c)
		if err != nil {
			log.Printf("Error signing settlement divisions: %s", err.Error())
			c.Status = lnutil.ContractStatusError
			nd.DlcManager.SaveContract(c)
			return
		}

		msg := lnutil.NewDlcOfferAcceptMsg(c, sigs)
		c.Status = lnutil.ContractStatusAccepted

		nd.DlcManager.SaveContract(c)
		nd.OmniOut <- msg
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

	err := nd.DlcManager.SaveContract(c)
	if err != nil {
		log.Printf("DlcOfferHandler SaveContract err %s\n", err.Error())
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
		log.Printf("DlcDeclineHandler FindContract err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusDeclined
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		log.Printf("DlcDeclineHandler SaveContract err %s\n", err.Error())
		return
	}
}

func (nd *LitNode) DlcAcceptHandler(msg lnutil.DlcOfferAcceptMsg, peer *RemotePeer) error {
	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		log.Printf("DlcAcceptHandler FindContract err %s\n", err.Error())
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

	c.Status = lnutil.ContractStatusAccepted
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		log.Printf("DlcAcceptHandler SaveContract err %s\n", err.Error())
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
	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		log.Printf("DlcContractAckHandler FindContract err %s\n", err.Error())
		return
	}

	// TODO: Check signatures

	c.Status = lnutil.ContractStatusAcknowledged

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		log.Printf("DlcContractAckHandler SaveContract err %s\n", err.Error())
		return
	}

	// We have everything now, send our signatures to the funding TX
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		log.Printf("DlcContractAckHandler No wallet for cointype %d\n", c.CoinType)
		return
	}

	tx, err := nd.BuildDlcFundingTransaction(c)
	if err != nil {
		log.Printf("DlcContractAckHandler BuildDlcFundingTransaction err %s\n", err.Error())
		return
	}

	err = wal.SignMyInputs(&tx)
	if err != nil {
		log.Printf("DlcContractAckHandler SignMyInputs err %s\n", err.Error())
		return
	}

	outMsg := lnutil.NewDlcContractFundingSigsMsg(c, &tx)

	nd.OmniOut <- outMsg
}

func (nd *LitNode) DlcFundingSigsHandler(msg lnutil.DlcContractFundingSigsMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		log.Printf("DlcFundingSigsHandler FindContract err %s\n", err.Error())
		return
	}

	// TODO: Check signatures

	// We have everything now. Sign our inputs to the funding TX and send it to the blockchain.
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		log.Printf("DlcFundingSigsHandler No wallet for cointype %d\n", c.CoinType)
		return
	}

	wal.SignMyInputs(msg.SignedFundingTx)

	wal.DirectSendTx(msg.SignedFundingTx)

	err = wal.WatchThis(c.FundingOutpoint)
	if err != nil {
		log.Printf("DlcFundingSigsHandler WatchThis err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusActive
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		log.Printf("DlcFundingSigsHandler SaveContract err %s\n", err.Error())
		return
	}

	outMsg := lnutil.NewDlcContractSigProofMsg(c, msg.SignedFundingTx)

	nd.OmniOut <- outMsg
}

func (nd *LitNode) DlcSigProofHandler(msg lnutil.DlcContractSigProofMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.LoadContract(msg.Idx)
	if err != nil {
		log.Printf("DlcSigProofHandler FindContract err %s\n", err.Error())
		return
	}

	// TODO: Check signatures
	wal, ok := nd.SubWallet[c.CoinType]
	if !ok {
		log.Printf("DlcSigProofHandler No wallet for cointype %d\n", c.CoinType)
		return
	}

	err = wal.WatchThis(c.FundingOutpoint)
	if err != nil {
		log.Printf("DlcSigProofHandler WatchThis err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusActive
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		log.Printf("DlcSigProofHandler SaveContract err %s\n", err.Error())
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

	c.FundingOutpoint = wire.OutPoint{fundingTx.TxHash(), 0}

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

func (nd *LitNode) SettleContract(cIdx uint64, oracleValue int64, oracleSig [32]byte) ([32]byte, [32]byte, error) {

	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		log.Printf("SettleContract FindContract err %s\n", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	c.Status = lnutil.ContractStatusSettling
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		log.Printf("SettleContract SaveContract err %s\n", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	d, err := c.GetDivision(oracleValue)
	if err != nil {
		log.Printf("SettleContract GetDivision err %s\n", err.Error())
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
		log.Printf("SettleContract SettlementTx err %s\n", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	mySig, err := nd.SignSettlementTx(c, settleTx, priv)
	if err != nil {
		log.Printf("SettleContract SignSettlementTx err %s", err.Error())
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
		log.Printf("SettleContract FundTxScript err %s", err.Error())
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
		log.Printf("SettleContract DirectSendTx (settle) err %s", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	// TODO: Claim the contract settlement output back to our wallet - otherwise the peer can claim it after locktime.
	txClaim := wire.NewMsgTx()
	txClaim.Version = 2

	settleOutpoint := wire.OutPoint{settleTx.TxHash(), 0}
	txClaim.AddTxIn(wire.NewTxIn(&settleOutpoint, nil, nil))

	addr, err := wal.NewAdr()
	txClaim.AddTxOut(wire.NewTxOut(d.ValueOurs-1000, lnutil.DirectWPKHScriptFromPKH(addr))) // todo calc fee - fee is double here because the contract output already had the fee deducted in the settlement TX

	kg.Step[2] = UseContractPayoutBase
	privSpend, _ := wal.GetPriv(kg)

	pubSpend := wal.GetPub(kg)
	privOracle, pubOracle := btcec.PrivKeyFromBytes(btcec.S256(), oracleSig[:])
	privContractOutput := lnutil.CombinePrivateKeys(privSpend, privOracle)

	var pubOracleBytes [33]byte
	copy(pubOracleBytes[:], pubOracle.SerializeCompressed())
	var pubSpendBytes [33]byte
	copy(pubSpendBytes[:], pubSpend.SerializeCompressed())

	settleScript := lnutil.DlcCommitScript(c.OurPayoutBase, pubOracleBytes, c.TheirPayoutBase, 5)
	err = nd.SignClaimTx(txClaim, settleTx.TxOut[0].Value, settleScript, privContractOutput, false)
	if err != nil {
		log.Printf("SettleContract SignClaimTx err %s", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	// Claim TX should be valid here, so publish it.
	err = wal.DirectSendTx(txClaim)
	if err != nil {
		log.Printf("SettleContract DirectSendTx (claim) err %s", err.Error())
		return [32]byte{}, [32]byte{}, err
	}

	c.Status = lnutil.ContractStatusClosed
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return [32]byte{}, [32]byte{}, err
	}
	return settleTx.TxHash(), txClaim.TxHash(), nil
}
