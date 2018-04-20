package lnutil

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/adiabat/btcd/btcec"
	"github.com/adiabat/btcd/chaincfg/chainhash"
	"github.com/adiabat/btcutil"

	"github.com/adiabat/btcd/txscript"
	"github.com/adiabat/btcd/wire"
)

type DlcContractStatus int

const (
	ContractStatusDraft        DlcContractStatus = 0
	ContractStatusOfferedByMe  DlcContractStatus = 1
	ContractStatusOfferedToMe  DlcContractStatus = 2
	ContractStatusDeclined     DlcContractStatus = 3
	ContractStatusAccepted     DlcContractStatus = 4
	ContractStatusAcknowledged DlcContractStatus = 5
	ContractStatusActive       DlcContractStatus = 6
	ContractStatusClosed       DlcContractStatus = 7
)

// scalarSize is the size of an encoded big endian scalar.
const scalarSize = 32

type DlcContract struct {
	Idx                                      uint64                           // Index of the contract for referencing in commands
	PeerIdx                                  uint32                           // Index of the peer we've offered the contract to or received the contract from
	PubKey                                   [33]byte                         // Key of the contract
	CoinType                                 uint32                           // Coin type
	OracleA, OracleR                         [33]byte                         // Pub keys of the oracle
	OracleTimestamp                          uint64                           // The time we expect the oracle to publish
	Division                                 []DlcContractDivision            // The payout specification
	OurFundingAmount, TheirFundingAmount     int64                            // The amounts either side are funding
	OurChangePKH, TheirChangePKH             [20]byte                         // PKH to which the contracts funding change should go
	OurFundMultisigPub, TheirFundMultisigPub [33]byte                         // Pubkey used in the funding multisig output
	OurPayoutPub, TheirPayoutPub             [33]byte                         // Pubkey to which the contracts are supposed to pay out
	Status                                   DlcContractStatus                // Status of the contract
	OurFundingInputs, TheirFundingInputs     []DlcContractFundingInput        // Outpoints used to fund the contract
	TheirSettlementSignatures                []DlcContractSettlementSignature // Signatures for the settlement transactions
}

type DlcContractDivision struct {
	OracleValue int64
	ValueOurs   int64
}

type DlcContractFundingInput struct {
	Outpoint wire.OutPoint
	Value    int64
}

func DlcContractFromBytes(b []byte) (*DlcContract, error) {
	buf := bytes.NewBuffer(b)
	c := new(DlcContract)

	copy(c.PubKey[:], buf.Next(33))
	copy(c.OracleA[:], buf.Next(33))
	copy(c.OracleR[:], buf.Next(33))

	peerIdx, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		fmt.Println("Error while deserializing varint for peerIdx")
		return nil, err
	}
	c.PeerIdx = uint32(peerIdx)

	coinType, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		fmt.Println("Error while deserializing varint for coinType")
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

	copy(c.OurPayoutPub[:], buf.Next(33))
	copy(c.TheirPayoutPub[:], buf.Next(33))

	status, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		fmt.Println("Error while deserializing varint for status")
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
	c.TheirSettlementSignatures = make([]DlcContractSettlementSignature, theirSigCount)

	for i := uint64(0); i < theirSigCount; i++ {
		outcome, err := wire.ReadVarInt(buf, 0)
		if err != nil {
			return nil, err
		}
		c.TheirSettlementSignatures[i].Outcome = int64(outcome)
		copy(c.TheirSettlementSignatures[i].Signature[:], buf.Next(64))
	}

	return c, nil
}

func (self *DlcContract) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(self.PubKey[:])
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
	buf.Write(self.OurPayoutPub[:])
	buf.Write(self.TheirPayoutPub[:])

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
		wire.WriteVarInt(&buf, 0, uint64(self.TheirSettlementSignatures[i].Outcome))
		buf.Write(self.TheirSettlementSignatures[i].Signature[:])
	}

	return buf.Bytes()
}

func (c DlcContract) GetDivision(value int64) (*DlcContractDivision, error) {
	for _, d := range c.Division {
		if d.OracleValue == value {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("Division not found in contract")
}

func (c DlcContract) GetTheirSettlementSignature(value int64) ([64]byte, error) {

	for _, s := range c.TheirSettlementSignatures {
		if s.Outcome == value {
			return s.Signature, nil
		}
	}

	return [64]byte{}, fmt.Errorf("Signature not found in contract")
}

func PrintTx(tx *wire.MsgTx) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	tx.Serialize(w)
	w.Flush()
	fmt.Printf("%x\n", buf.Bytes())
}

func DlcOutput(pubKeyPeer, pubKeyOracleSig, ourPubKey [33]byte, value int64) *wire.TxOut {
	scriptBytes := DlcCommitScript(pubKeyPeer, pubKeyOracleSig, ourPubKey, 5)
	scriptBytes = P2WSHify(scriptBytes)

	return wire.NewTxOut(value, scriptBytes)
}

// DLC Commit script makes a script that pays to (PubKeyPeer+PubKeyOracleSig or (OurPubKey and TimeDelay))
// We send this over (signed) to the other side. If they publish the TX with the correct script they can use
// The oracle's signature and their own private key to claim the funds from the output. However,
// If they send the wrong one, they won't be able to claim the funds - and we can claim them once the
// Time delay has passed.
func DlcCommitScript(pubKeyPeer, pubKeyOracleSig, ourPubKey [33]byte, delay uint16) []byte {
	builder := txscript.NewScriptBuilder()

	// 1 for penalty / revoked, 0 for timeout
	// 1, so timeout
	builder.AddOp(txscript.OP_IF)

	// Combine pubKey and Oracle Sig
	combinedPubKey := CombinePubs(pubKeyPeer, pubKeyOracleSig)
	builder.AddData(combinedPubKey[:])

	// 0, so revoked
	builder.AddOp(txscript.OP_ELSE)

	// CSV delay
	builder.AddInt64(int64(delay))
	// CSV check, fails here if too early
	builder.AddOp(txscript.OP_NOP3) // really OP_CHECKSEQUENCEVERIFY
	// Drop delay value
	builder.AddOp(txscript.OP_DROP)
	// push timeout key
	builder.AddData(ourPubKey[:])

	builder.AddOp(txscript.OP_ENDIF)

	// check whatever pubkey is left on the stack
	builder.AddOp(txscript.OP_CHECKSIG)

	// never any errors we care about here.
	s, _ := builder.Script()
	return s
}

// BigIntToEncodedBytes converts a big integer into its corresponding
// 32 byte little endian representation.
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

// Compute the predicted signature s*G
// it's just R - h(R||m)A
func DlcCalcOracleSignaturePubKey(msg []byte, oracleA, oracleR [33]byte) ([33]byte, error) {
	var sigPub [33]byte

	curve := btcec.S256()
	Pub, err := btcec.ParsePubKey(oracleA[:], curve)
	if err != nil {
		return sigPub, err
	}
	R, err := btcec.ParsePubKey(oracleR[:], curve)
	if err != nil {
		return sigPub, err
	}

	// h = Hash(R || m)
	Rpxb := BigIntToEncodedBytes(R.X)
	hashInput := make([]byte, 0, scalarSize*2)
	hashInput = append(hashInput, Rpxb[:]...)
	hashInput = append(hashInput, msg...)
	h := chainhash.HashB(hashInput)

	// h * A
	Pub.X, Pub.Y = curve.ScalarMult(Pub.X, Pub.Y, h)

	// this works?
	Pub.Y.Neg(Pub.Y)
	//	Pub.Y = Pub.Y.Neg()

	Pub.Y.Mod(Pub.Y, curve.P)

	sG := new(btcec.PublicKey)

	// Pub has been negated; add it to R
	sG.X, sG.Y = curve.Add(R.X, R.Y, Pub.X, Pub.Y)

	copy(sigPub[:], sG.SerializeCompressed())

	return sigPub, nil
}

// Ours = the one we generate & sign. Theirs (ours = false) = the one they generated, so we can use their sigs
func SettlementTx(c *DlcContract, d DlcContractDivision, contractInput wire.OutPoint, ours bool) (*wire.MsgTx, error) {

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
	oracleSigPub, err := DlcCalcOracleSignaturePubKey(buf.Bytes(), c.OracleA, c.OracleR)
	if err != nil {
		return nil, err
	}

	// Ours = the one we generate & sign. Theirs (ours = false) = the one they generated, so we can use their sigs
	if ours {
		if valueTheirs > 0 {
			tx.AddTxOut(DlcOutput(c.TheirPayoutPub, oracleSigPub, c.OurPayoutPub, valueTheirs-(totalFee-feeOurs)))
		}

		if valueOurs > 0 {
			var ourPayoutPKH [20]byte
			copy(ourPayoutPKH[:], btcutil.Hash160(c.OurPayoutPub[:]))

			tx.AddTxOut(wire.NewTxOut(valueOurs-feeOurs, DirectWPKHScriptFromPKH(ourPayoutPKH)))
		}
	} else {
		if valueOurs > 0 {
			tx.AddTxOut(DlcOutput(c.OurPayoutPub, oracleSigPub, c.TheirPayoutPub, valueOurs-(totalFee-feeTheirs)))
		}

		if valueTheirs > 0 {
			var theirPayoutPKH [20]byte
			copy(theirPayoutPKH[:], btcutil.Hash160(c.TheirPayoutPub[:]))
			tx.AddTxOut(wire.NewTxOut(valueTheirs-feeTheirs, DirectWPKHScriptFromPKH(theirPayoutPKH)))
		}
	}

	return tx, nil
}
