package lnutil

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/adiabat/btcd/btcec"
	"github.com/adiabat/btcd/chaincfg/chainhash"

	"github.com/adiabat/btcd/wire"
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
)

// scalarSize is the size of an encoded big endian scalar.
const scalarSize = 32

const (
	OFFERTYPE_FORWARD = 0x01
)

// DlcOffer is a generic interface for offers that all specific offer types follow.
// It is a proposed contract that has not yet been accepted or funded
type DlcOffer interface {
	// Type of offer
	OfferType() uint8
	// Index of the contract for referencing in commands
	Idx() uint64
	SetIdx(idx uint64)
	// Index of the contract on the other peer
	TheirIdx() uint64

	Peer() uint32
	// Returns the serialized offer
	Bytes() []byte

	IsAccepted() bool
	SetAccepted()
	// Creates a new contract from this offer
	CreateContract() *DlcContract
	// Checks if this offer is equal to the provided contract in terms
	// of payout, oracle keys and settlement time. Used for auto-accepting
	EqualsContract(c *DlcContract) bool
}

func DlcOfferFromBytes(b []byte) (DlcOffer, error) {
	offerType := b[0] // first byte signifies what type of message is

	switch offerType {
	case OFFERTYPE_FORWARD:
		return DlcFwdOfferFromBytes(b[1:])
	default:
		return nil, fmt.Errorf("Unknown offer of type %d ", offerType)
	}
}

// DlcFwdOffer is an offer for a specific contract template: it is a bitcoin (or other
// coin) settled forward, which is symmetrically funded
type DlcFwdOffer struct {
	// Convenience definition for serialization from RPC
	OType uint8
	// Index of the offer
	OIdx uint64
	// Index of the offer on the other peer
	TheirOIdx uint64
	// Index of the peer offering to / from
	PeerIdx uint32
	// Coin type
	CoinType uint32
	// Pub keys of the oracle and the R point used in the contract
	OracleA, OracleR [33]byte
	// time of expected settlement
	SettlementTime uint64
	// amount of funding, in sats, each party contributes
	FundAmt int64
	// slice of my payouts for given oracle prices
	Payouts []DlcContractDivision

	// if true, I'm the 'buyer' of the foward asset (and I'm short bitcoin)
	ImBuyer bool

	// amount of asset to be delivered at settlement time
	// note that initial price is FundAmt / AssetQuantity
	AssetQuantity int64

	// Stores if the offer was accepted. When receiving a matching
	// Contract draft, we can accept that too.
	Accepted bool
}

func DlcFwdOfferFromBytes(b []byte) (*DlcFwdOffer, error) {
	buf := bytes.NewBuffer(b)
	o := new(DlcFwdOffer)
	var err error

	o.OIdx, err = wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	o.TheirOIdx, err = wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	peerIdx, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	o.PeerIdx = uint32(peerIdx)

	coinType, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	o.CoinType = uint32(coinType)

	copy(o.OracleA[:], buf.Next(33))
	copy(o.OracleR[:], buf.Next(33))

	o.SettlementTime, err = wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	fundAmt, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	o.FundAmt = int64(fundAmt)

	if bytes.Equal(buf.Next(1), []byte{1}) {
		o.ImBuyer = true
	}

	if bytes.Equal(buf.Next(1), []byte{1}) {
		o.Accepted = true
	}

	assetQty, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		return nil, err
	}

	o.AssetQuantity = int64(assetQty)
	o.OType = OFFERTYPE_FORWARD
	return o, nil
}

func (o *DlcFwdOffer) Bytes() []byte {
	var buf bytes.Buffer

	buf.Write([]byte{OFFERTYPE_FORWARD})
	wire.WriteVarInt(&buf, 0, uint64(o.OIdx))
	wire.WriteVarInt(&buf, 0, uint64(o.TheirOIdx))
	wire.WriteVarInt(&buf, 0, uint64(o.PeerIdx))
	wire.WriteVarInt(&buf, 0, uint64(o.CoinType))
	buf.Write(o.OracleA[:])
	buf.Write(o.OracleR[:])
	wire.WriteVarInt(&buf, 0, uint64(o.SettlementTime))
	wire.WriteVarInt(&buf, 0, uint64(o.FundAmt))
	if o.ImBuyer {
		buf.Write([]byte{1})
	} else {
		buf.Write([]byte{0})
	}
	if o.Accepted {
		buf.Write([]byte{1})
	} else {
		buf.Write([]byte{0})
	}
	wire.WriteVarInt(&buf, 0, uint64(o.AssetQuantity))

	return buf.Bytes()
}
func (o *DlcFwdOffer) OfferType() uint8  { return OFFERTYPE_FORWARD }
func (o *DlcFwdOffer) Idx() uint64       { return o.OIdx }
func (o *DlcFwdOffer) SetIdx(idx uint64) { o.OIdx = idx }
func (o *DlcFwdOffer) TheirIdx() uint64  { return o.TheirOIdx }
func (o *DlcFwdOffer) Peer() uint32      { return o.PeerIdx }
func (o *DlcFwdOffer) IsAccepted() bool  { return o.Accepted }
func (o *DlcFwdOffer) SetAccepted()      { o.Accepted = true }
func (o *DlcFwdOffer) CreateContract() *DlcContract {
	c := new(DlcContract)

	c.CoinType = o.CoinType
	c.OracleA = o.OracleA
	c.OracleR = o.OracleR
	c.OracleTimestamp = o.SettlementTime
	BuildPayouts(o)
	c.Division = o.Payouts
	c.Status = ContractStatusDraft
	c.OurFundingAmount = o.FundAmt
	c.TheirFundingAmount = o.FundAmt
	return c
}
func (o *DlcFwdOffer) EqualsContract(c *DlcContract) bool {
	return true
}

// Build payouts populates the payout schedule for a forward offer
func BuildPayouts(o *DlcFwdOffer) error {

	var price, maxPrice, oracleStep int64

	// clear out payout schedule just in case one's already there
	o.Payouts = make([]DlcContractDivision, 0)

	// use a coarse oracle that rounds asset prices to the neares 100 satoshis
	// the oracle must also use the same price stepping
	oracleStep = 100
	// max out at 100K (corresponds to asset price of 1000 per bitcoin)
	// this is very ugly to hard-code here but we can leave it until we implement
	// a more complex oracle price signing, such as base/mantissa.
	maxPrice = 100000

	for price = 0; price <= maxPrice; price += oracleStep {
		var div DlcContractDivision
		div.OracleValue = price
		// generally, the buyer gets the asset quantity times the oracle's price
		div.ValueOurs = o.AssetQuantity * price
		// if that exceeds total contract funds, they get everything
		if div.ValueOurs > o.FundAmt*2 {
			div.ValueOurs = o.FundAmt * 2
		}
		if !o.ImBuyer {
			// if I am the seller, instead I get whatever is left (which could be 0)
			div.ValueOurs = (o.FundAmt * 2) - div.ValueOurs
		}
		o.Payouts = append(o.Payouts, div)
	}

	return nil
}

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
		fmt.Println("Error while deserializing varint for theirIdx")
		return nil, err
	}
	c.Idx = ourIdx

	theirIdx, err := wire.ReadVarInt(buf, 0)
	if err != nil {
		fmt.Println("Error while deserializing varint for theirIdx")
		return nil, err
	}
	c.TheirIdx = theirIdx

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

	copy(c.OurPayoutBase[:], buf.Next(33))
	copy(c.TheirPayoutBase[:], buf.Next(33))

	copy(c.OurPayoutPKH[:], buf.Next(20))
	copy(c.TheirPayoutPKH[:], buf.Next(20))

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
	fmt.Printf("%x\n", buf.Bytes())
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
	curve := btcec.S256()

	A, err := btcec.ParsePubKey(pubA[:], curve)
	if err != nil {
		return returnValue, err
	}

	R, err := btcec.ParsePubKey(pubR[:], curve)
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

	P := new(btcec.PublicKey)

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

	totalFee := int64(1000) // TODO: Calculate
	feeEach := int64(float64(totalFee) / float64(2))
	feeOurs := feeEach
	feeTheirs := feeEach
	valueOurs := d.ValueOurs
	// We don't have enough to pay for a fee. We get 0, our contract partner
	// pays the rest of the fee
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
