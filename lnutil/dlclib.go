package lnutil

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/wire"
)

// DlcContractStatus is an enumeration containing the various statuses a
// contract can have
type DlcContractStatus int

const (
	ContractStatusDraft        DlcContractStatus = 0
	ContractStatusOfferedByMe  DlcContractStatus = 1
	ContractStatusOfferedToMe  DlcContractStatus = 2
	ContractStatusDeclined     DlcContractStatus = 3
	ContractStatusAccepted     DlcContractStatus = 4
	ContractStatusAcknowledged DlcContractStatus = 5
	ContractStatusActive       DlcContractStatus = 6
	ContractStatusSettling     DlcContractStatus = 7
	ContractStatusClosed       DlcContractStatus = 8
	ContractStatusError        DlcContractStatus = 9
	ContractStatusAccepting    DlcContractStatus = 10
)

// scalarSize is the size of an encoded big endian scalar.
const scalarSize = 32

// DlcContract is a struct containing all elements to work with a Discreet
// Log Contract. This struct is stored in the database
type DlcContract struct {
	// Index of the contract for referencing in commands
	Idx uint64
	// Index of the contract on the other peer (so we can reference it in
	// messages)
	TheirIdx uint64
	// Index of the peer we've offered the contract to or received the contract
	// from
	PeerIdx uint32
	// Coin type
	CoinType uint32
	// Pub keys of the oracle and the R point used in the contract
	OracleA, OracleR [33]byte
	// The time we expect the oracle to publish
	OracleTimestamp uint64
	// The payout specification
	Division []DlcContractDivision
	// The amounts either side are funding
	OurFundingAmount, TheirFundingAmount int64
	// PKH to which the contracts funding change should go
	OurChangePKH, TheirChangePKH [20]byte
	// Pubkey used in the funding multisig output
	OurFundMultisigPub, TheirFundMultisigPub [33]byte
	// Pubkey to be used in the commit script (combined with oracle pubkey
	// or CSV timeout)
	OurPayoutBase, TheirPayoutBase [33]byte
	// Pubkeyhash to which the contract pays out (directly)
	OurPayoutPKH, TheirPayoutPKH [20]byte
	// Status of the contract
	Status DlcContractStatus
	// Outpoints used to fund the contract
	OurFundingInputs, TheirFundingInputs []DlcContractFundingInput
	// Signatures for the settlement transactions
	TheirSettlementSignatures []DlcContractSettlementSignature
	// The outpoint of the funding TX we want to spend in the settlement
	// for easier monitoring
	FundingOutpoint wire.OutPoint
}

// DlcContractDivision describes a single division of the contract. If the
// oracle predicts OracleValue, we receive ValueOurs
type DlcContractDivision struct {
	OracleValue int64
	ValueOurs   int64
}

// DlcContractFundingInput describes a UTXO that is offered to fund the
// contract with
type DlcContractFundingInput struct {
	Outpoint wire.OutPoint
	Value    int64
}

// DlcContractFromBytes deserializes a byte array back into a DlcContract struct
func DlcContractFromBytes(b []byte) (*DlcContract, error) {
	buf := bytes.NewBuffer(b)
	c := new(DlcContract)

	ourIdx, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		logging.Errorf("Error while deserializing varint for theirIdx: %s", err.Error())
		return nil, err
	}
	c.Idx = ourIdx

	theirIdx, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		logging.Errorf("Error while deserializing varint for theirIdx: %s", err.Error())
		return nil, err
	}
	c.TheirIdx = theirIdx

	copy(c.OracleA[:], buf.Next(33))
	copy(c.OracleR[:], buf.Next(33))

	peerIdx, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		logging.Errorf("Error while deserializing varint for peerIdx: %s", err.Error())
		return nil, err
	}
	c.PeerIdx = uint32(peerIdx)

	coinType, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		logging.Errorf("Error while deserializing varint for coinType: %s", err.Error())
		return nil, err
	}
	c.CoinType = uint32(coinType)
	c.OracleTimestamp, err = wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}
	ourFundingAmount, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}
	c.OurFundingAmount = int64(ourFundingAmount)
	theirFundingAmount, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}
	c.TheirFundingAmount = int64(theirFundingAmount)

	copy(c.OurChangePKH[:], buf.Next(20))
	copy(c.TheirChangePKH[:], buf.Next(20))

	copy(c.OurFundMultisigPub[:], buf.Next(33))
	copy(c.TheirFundMultisigPub[:], buf.Next(33))

	copy(c.OurPayoutBase[:], buf.Next(33))
	copy(c.TheirPayoutBase[:], buf.Next(33))

	copy(c.OurPayoutPKH[:], buf.Next(20))
	copy(c.TheirPayoutPKH[:], buf.Next(20))

	status, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		logging.Errorf("Error while deserializing varint for status: %s", err.Error())
		return nil, err
	}

	c.Status = DlcContractStatus(status)

	ourInputsLen, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	c.OurFundingInputs = make([]DlcContractFundingInput, ourInputsLen)
	var op [36]byte
	for i := uint64(0); i < ourInputsLen; i++ {
		copy(op[:], buf.Next(36))
		c.OurFundingInputs[i].Outpoint = *OutPointFromBytes(op)
		inputValue, err := wire.ReadVarInt(buf, 0)
		if err != nil {
			return nil, err
		}
		c.OurFundingInputs[i].Value = int64(inputValue)
	}

	theirInputsLen, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	c.TheirFundingInputs = make([]DlcContractFundingInput, theirInputsLen)
	for i := uint64(0); i < theirInputsLen; i++ {
		copy(op[:], buf.Next(36))
		c.TheirFundingInputs[i].Outpoint = *OutPointFromBytes(op)
		inputValue, err := wire.ReadVarInt(buf, 0)
		if err != nil {

			return nil, err
		}
		c.TheirFundingInputs[i].Value = int64(inputValue)
	}

	divisionLen, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	c.Division = make([]DlcContractDivision, divisionLen)
	for i := uint64(0); i < divisionLen; i++ {
		oracleValue, err := wire.ReadVarInt(buf, 0)
		if err != nil {
			return nil, err
		}
		valueOurs, err := wire.ReadVarInt(buf, 0)
		if err != nil {
			return nil, err
		}
		c.Division[i].OracleValue = int64(oracleValue)
		c.Division[i].ValueOurs = int64(valueOurs)
	}

	theirSigCount, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}
	c.TheirSettlementSignatures = make([]DlcContractSettlementSignature,
		theirSigCount)

	for i := uint64(0); i < theirSigCount; i++ {
		outcome, err := wire.ReadVarInt(buf, 0)
		if err != nil {
			return nil, err
		}
		c.TheirSettlementSignatures[i].Outcome = int64(outcome)
		copy(c.TheirSettlementSignatures[i].Signature[:], buf.Next(64))
	}

	copy(op[:], buf.Next(36))
	c.FundingOutpoint = *OutPointFromBytes(op)

	return c, nil
}

// Bytes serializes a DlcContract struct into a byte array
func (self *DlcContract) Bytes() []byte {
	var buf bytes.Buffer

	wire.WriteVarInt(&buf, 0, uint64(self.Idx))
	wire.WriteVarInt(&buf, 0, uint64(self.TheirIdx))
	buf.Write(self.OracleA[:])
	buf.Write(self.OracleR[:])
	wire.WriteVarInt(&buf, 0, uint64(self.PeerIdx))
	wire.WriteVarInt(&buf, 0, uint64(self.CoinType))
	wire.WriteVarInt(&buf, 0, uint64(self.OracleTimestamp))
	wire.WriteVarInt(&buf, 0, uint64(self.OurFundingAmount))
	wire.WriteVarInt(&buf, 0, uint64(self.TheirFundingAmount))

	buf.Write(self.OurChangePKH[:])
	buf.Write(self.TheirChangePKH[:])
	buf.Write(self.OurFundMultisigPub[:])
	buf.Write(self.TheirFundMultisigPub[:])
	buf.Write(self.OurPayoutBase[:])
	buf.Write(self.TheirPayoutBase[:])
	buf.Write(self.OurPayoutPKH[:])
	buf.Write(self.TheirPayoutPKH[:])

	var status = uint64(self.Status)
	wire.WriteVarInt(&buf, 0, status)

	ourInputsLen := uint64(len(self.OurFundingInputs))
	wire.WriteVarInt(&buf, 0, ourInputsLen)

	for i := 0; i < len(self.OurFundingInputs); i++ {
		opArr := OutPointToBytes(self.OurFundingInputs[i].Outpoint)
		buf.Write(opArr[:])
		wire.WriteVarInt(&buf, 0, uint64(self.OurFundingInputs[i].Value))
	}

	theirInputsLen := uint64(len(self.TheirFundingInputs))
	wire.WriteVarInt(&buf, 0, theirInputsLen)

	for i := 0; i < len(self.TheirFundingInputs); i++ {
		opArr := OutPointToBytes(self.TheirFundingInputs[i].Outpoint)
		buf.Write(opArr[:])
		wire.WriteVarInt(&buf, 0, uint64(self.TheirFundingInputs[i].Value))
	}

	divisionLen := uint64(len(self.Division))
	wire.WriteVarInt(&buf, 0, divisionLen)

	for i := 0; i < len(self.Division); i++ {
		wire.WriteVarInt(&buf, 0, uint64(self.Division[i].OracleValue))
		wire.WriteVarInt(&buf, 0, uint64(self.Division[i].ValueOurs))
	}

	theirSigLen := uint64(len(self.TheirSettlementSignatures))
	wire.WriteVarInt(&buf, 0, theirSigLen)

	for i := 0; i < len(self.TheirSettlementSignatures); i++ {
		outcome := uint64(self.TheirSettlementSignatures[i].Outcome)
		wire.WriteVarInt(&buf, 0, outcome)
		buf.Write(self.TheirSettlementSignatures[i].Signature[:])
	}

	opArr := OutPointToBytes(self.FundingOutpoint)
	buf.Write(opArr[:])

	return buf.Bytes()
}

// GetDivision loops over all division specifications inside the contract and
// returns the one matching the requested oracle value
func (c DlcContract) GetDivision(value int64) (*DlcContractDivision, error) {
	for _, d := range c.Division {
		if d.OracleValue == value {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("Division not found in contract")
}

// GetTheirSettlementSignature loops over all stored settlement signatures from
// the counter party and returns the one matching the requested oracle value
func (c DlcContract) GetTheirSettlementSignature(val int64) ([64]byte, error) {

	for _, s := range c.TheirSettlementSignatures {
		if s.Outcome == val {
			return s.Signature, nil
		}
	}

	return [64]byte{}, fmt.Errorf("Signature not found in contract")
}

// PrintTx prints out a transaction as serialized byte array to StdOut
func PrintTx(tx *wire.MsgTx) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	tx.Serialize(w)
	w.Flush()
	logging.Infof("%x\n", buf.Bytes())
}

// DlcOutput returns a Txo for a particular value that pays to
// (PubKeyPeer+PubKeyOracleSig or (OurPubKey and TimeDelay))
func DlcOutput(pkPeer, pkOracleSig, pkOurs [33]byte, value int64) *wire.TxOut {
	scriptBytes := DlcCommitScript(pkPeer, pkOracleSig, pkOurs, 5)
	scriptBytes = P2WSHify(scriptBytes)

	return wire.NewTxOut(value, scriptBytes)
}

// DlcCommitScript makes a script that pays to (PubKeyPeer+PubKeyOracleSig or
// (OurPubKey and TimeDelay)). We send this over (signed) to the other side. If
// they publish the TX with the correct script they can use the oracle's
// signature and their own private key to claim the funds from the output.
// However, if they send the wrong one, they won't be able to claim the funds
// and we can claim them once the time delay has passed.
func DlcCommitScript(pubKeyPeer, pubKeyOracleSig, ourPubKey [33]byte,
	delay uint16) []byte {
	// Combine pubKey and Oracle Sig
	combinedPubKey := CombinePubs(pubKeyPeer, pubKeyOracleSig)
	return CommitScript(combinedPubKey, ourPubKey, delay)
}

// BigIntToEncodedBytes converts a big integer into its corresponding
// 32 byte big endian representation.
func BigIntToEncodedBytes(a *big.Int) *[32]byte {
	s := new([32]byte)
	if a == nil {
		return s
	}
	// Caveat: a can be longer than 32 bytes.
	aB := a.Bytes()

	// If we have a short byte string, expand
	// it so that it's long enough.
	aBLen := len(aB)
	if aBLen < scalarSize {
		diff := scalarSize - aBLen
		for i := 0; i < diff; i++ {
			aB = append([]byte{0x00}, aB...)
		}
	}

	for i := 0; i < scalarSize; i++ {
		s[i] = aB[i]
	}

	return s
}

// DlcCalcOracleSignaturePubKey computes the predicted signature s*G
// it's just R - h(R||m)A
func DlcCalcOracleSignaturePubKey(msg []byte, oracleA,
	oracleR [33]byte) ([33]byte, error) {
	return computePubKey(oracleA, oracleR, msg)
}

// calculates P = pubR - h(msg, pubR)pubA
func computePubKey(pubA, pubR [33]byte, msg []byte) ([33]byte, error) {
	var returnValue [33]byte

	// Hardcode curve
	curve := koblitz.S256()

	A, err := koblitz.ParsePubKey(pubA[:], curve)
	if err != nil {
		return returnValue, err
	}

	R, err := koblitz.ParsePubKey(pubR[:], curve)
	if err != nil {
		return returnValue, err
	}

	// e = Hash(messageType, oraclePubQ)
	var hashInput []byte
	hashInput = append(msg, R.X.Bytes()...)
	e := chainhash.HashB(hashInput)

	bigE := new(big.Int).SetBytes(e)

	if bigE.Cmp(curve.N) >= 0 {
		return returnValue, fmt.Errorf("hash of (msg, pubR) too big")
	}

	// e * B
	A.X, A.Y = curve.ScalarMult(A.X, A.Y, e)

	A.Y.Neg(A.Y)

	A.Y.Mod(A.Y, curve.P)

	P := new(koblitz.PublicKey)

	// add to R
	P.X, P.Y = curve.Add(A.X, A.Y, R.X, R.Y)
	copy(returnValue[:], P.SerializeCompressed())
	return returnValue, nil
}

// SettlementTx returns the transaction to settle the contract. ours = the one
// we generate & sign. Theirs (ours = false) = the one they generated, so we can
// use their sigs
func SettlementTx(c *DlcContract, d DlcContractDivision,
	ours bool) (*wire.MsgTx, error) {

	tx := wire.NewMsgTx()
	// set version 2, for op_csv
	tx.Version = 2

	tx.AddTxIn(wire.NewTxIn(&c.FundingOutpoint, nil, nil))

	totalFee := int64(consts.DlcSettlementTxFee) // TODO: Calculate
	feeEach := int64(float64(totalFee) / float64(2))
	feeOurs := feeEach
	feeTheirs := feeEach
	valueOurs := d.ValueOurs
	// We don't have enough to pay for a fee. We get 0, our contract partner
	// pays the rest of the fee
	if valueOurs < feeEach {
		valueOurs = 0
	} else {
		valueOurs = d.ValueOurs - feeOurs
	}
	totalContractValue := c.TheirFundingAmount + c.OurFundingAmount
	valueTheirs := totalContractValue - d.ValueOurs

	if valueTheirs < feeEach {
		feeTheirs = valueTheirs
		valueTheirs = 0
		feeOurs = totalFee - feeTheirs
		valueOurs = d.ValueOurs - feeOurs
	} else {
		valueTheirs -= feeTheirs
	}

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint64(0))
	binary.Write(&buf, binary.BigEndian, uint64(0))
	binary.Write(&buf, binary.BigEndian, uint64(0))
	binary.Write(&buf, binary.BigEndian, d.OracleValue)
	oracleSigPub, err := DlcCalcOracleSignaturePubKey(buf.Bytes(),
		c.OracleA, c.OracleR)
	if err != nil {
		return nil, err
	}

	// Ours = the one we generate & sign. Theirs (ours = false) = the one they
	// generated, so we can use their sigs
	if ours {
		if valueTheirs > 0 {
			tx.AddTxOut(DlcOutput(c.TheirPayoutBase, oracleSigPub,
				c.OurPayoutBase, valueTheirs))
		}

		if valueOurs > 0 {
			tx.AddTxOut(wire.NewTxOut(valueOurs,
				DirectWPKHScriptFromPKH(c.OurPayoutPKH)))
		}
	} else {
		if valueOurs > 0 {
			tx.AddTxOut(DlcOutput(c.OurPayoutBase, oracleSigPub,
				c.TheirPayoutBase, valueOurs))
		}

		if valueTheirs > 0 {
			tx.AddTxOut(wire.NewTxOut(valueTheirs,
				DirectWPKHScriptFromPKH(c.TheirPayoutPKH)))
		}
	}

	return tx, nil
}
