package lnutil

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"

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

	_ = binary.Read(buf, binary.BigEndian, &c.PeerIdx)
	_ = binary.Read(buf, binary.BigEndian, &c.CoinType)
	_ = binary.Read(buf, binary.BigEndian, &c.OracleTimestamp)
	_ = binary.Read(buf, binary.BigEndian, &c.OurFundingAmount)
	_ = binary.Read(buf, binary.BigEndian, &c.TheirFundingAmount)

	copy(c.OurChangePKH[:], buf.Next(20))
	copy(c.TheirChangePKH[:], buf.Next(20))

	copy(c.OurFundMultisigPub[:], buf.Next(33))
	copy(c.TheirFundMultisigPub[:], buf.Next(33))

	copy(c.OurPayoutPub[:], buf.Next(33))
	copy(c.TheirPayoutPub[:], buf.Next(33))

	var status int32
	_ = binary.Read(buf, binary.BigEndian, &status)

	c.Status = DlcContractStatus(status)

	var ourInputsLen uint32
	_ = binary.Read(buf, binary.BigEndian, &ourInputsLen)
	fmt.Printf("[R] Our input len: [%d]\n", ourInputsLen)

	c.OurFundingInputs = make([]DlcContractFundingInput, ourInputsLen)
	var op [36]byte
	for i := uint32(0); i < ourInputsLen; i++ {
		copy(op[:], buf.Next(36))
		c.OurFundingInputs[i].Outpoint = *OutPointFromBytes(op)
		_ = binary.Read(buf, binary.BigEndian, &c.OurFundingInputs[i].Value)
	}

	var theirInputsLen uint32
	_ = binary.Read(buf, binary.BigEndian, &theirInputsLen)
	fmt.Printf("[R] Their input len: [%d]\n", theirInputsLen)

	c.TheirFundingInputs = make([]DlcContractFundingInput, theirInputsLen)
	for i := uint32(0); i < theirInputsLen; i++ {
		copy(op[:], buf.Next(36))
		c.TheirFundingInputs[i].Outpoint = *OutPointFromBytes(op)
		_ = binary.Read(buf, binary.BigEndian, &c.TheirFundingInputs[i].Value)
	}

	var divisionLen uint32
	_ = binary.Read(buf, binary.BigEndian, &divisionLen)
	c.Division = make([]DlcContractDivision, divisionLen)
	for i := uint32(0); i < divisionLen; i++ {
		_ = binary.Read(buf, binary.BigEndian, &c.Division[i].OracleValue)
		_ = binary.Read(buf, binary.BigEndian, &c.Division[i].ValueOurs)
	}

	var theirSigCount uint32
	_ = binary.Read(buf, binary.BigEndian, &theirSigCount)
	c.TheirSettlementSignatures = make([]DlcContractSettlementSignature, theirSigCount)

	var sigLen uint32
	for i := uint32(0); i < theirSigCount; i++ {
		_ = binary.Read(buf, binary.BigEndian, &c.TheirSettlementSignatures[i].Outcome)
		_ = binary.Read(buf, binary.BigEndian, &sigLen)
		c.TheirSettlementSignatures[i].Signature = make([]byte, sigLen)
		copy(c.TheirSettlementSignatures[i].Signature, buf.Next(int(sigLen)))
	}

	return c, nil
}

func (self *DlcContract) Bytes() []byte {
	var buf bytes.Buffer

	buf.Write(self.PubKey[:])
	buf.Write(self.OracleA[:])
	buf.Write(self.OracleR[:])
	binary.Write(&buf, binary.BigEndian, self.PeerIdx)
	binary.Write(&buf, binary.BigEndian, self.CoinType)
	binary.Write(&buf, binary.BigEndian, self.OracleTimestamp)
	binary.Write(&buf, binary.BigEndian, self.OurFundingAmount)
	binary.Write(&buf, binary.BigEndian, self.TheirFundingAmount)

	buf.Write(self.OurChangePKH[:])
	buf.Write(self.TheirChangePKH[:])
	buf.Write(self.OurFundMultisigPub[:])
	buf.Write(self.TheirFundMultisigPub[:])
	buf.Write(self.OurPayoutPub[:])
	buf.Write(self.TheirPayoutPub[:])

	var status = int32(self.Status)
	binary.Write(&buf, binary.BigEndian, status)

	ourInputsLen := uint32(len(self.OurFundingInputs))
	fmt.Printf("[W] Our input len: [%d]\n", ourInputsLen)
	binary.Write(&buf, binary.BigEndian, ourInputsLen)

	for i := 0; i < len(self.OurFundingInputs); i++ {
		opArr := OutPointToBytes(self.OurFundingInputs[i].Outpoint)
		buf.Write(opArr[:])
		binary.Write(&buf, binary.BigEndian, self.OurFundingInputs[i].Value)
	}

	theirInputsLen := uint32(len(self.TheirFundingInputs))
	fmt.Printf("[W] Their input len: [%d]\n", theirInputsLen)
	binary.Write(&buf, binary.BigEndian, theirInputsLen)

	for i := 0; i < len(self.TheirFundingInputs); i++ {
		opArr := OutPointToBytes(self.TheirFundingInputs[i].Outpoint)
		buf.Write(opArr[:])
		binary.Write(&buf, binary.BigEndian, self.TheirFundingInputs[i].Value)
	}

	divisionLen := uint32(len(self.Division))
	binary.Write(&buf, binary.BigEndian, divisionLen)

	for i := 0; i < len(self.Division); i++ {
		binary.Write(&buf, binary.BigEndian, self.Division[i].OracleValue)
		binary.Write(&buf, binary.BigEndian, self.Division[i].ValueOurs)
	}

	theirSigLen := uint32(len(self.TheirSettlementSignatures))
	binary.Write(&buf, binary.BigEndian, theirSigLen)

	for i := 0; i < len(self.TheirSettlementSignatures); i++ {
		binary.Write(&buf, binary.BigEndian, self.TheirSettlementSignatures[i].Outcome)
		binary.Write(&buf, binary.BigEndian, uint32(len(self.TheirSettlementSignatures[i].Signature)))
		buf.Write(self.TheirSettlementSignatures[i].Signature)
	}

	fmt.Printf("Serialized contract: %x\n", buf.Bytes())

	return buf.Bytes()
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
