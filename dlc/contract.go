package dlc

import (
	"bytes"
	"encoding/binary"

	"github.com/adiabat/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
)

type DlcContractStatus int

const (
	ContractStatusDraft DlcContractStatus = iota
	ContractStatusOfferedByMe
	ContractStatusOfferedToMe
	ContractStatusActive
	ContractStatusClosed
)

type DlcContract struct {
	Idx                                  uint64                    // Index of the contract for referencing in commands
	OracleA, OracleB, OracleQ            [33]byte                  // Pub keys of the oracle
	OracleDataFeed, OracleTimestamp      uint64                    // The data feed and time we use for contract settlement
	ValueAllOurs, ValueAllTheirs         uint64                    // The value of the datafeed based on which all money in the contract goes to either party
	OurFundingAmount, TheirFundingAmount uint64                    // The amounts either side are funding
	OurPayoutPKH, TheirPayoutPKH         [20]byte                  // PKH to which the contracts are supposed to pay out
	Status                               DlcContractStatus         // Status of the contract
	OurFundingInputs, TheirFundingInputs []DlcContractFundingInput // Outpoints used to fund the contract
}

type DlcContractFundingInput struct {
	Outpoint wire.OutPoint
	Value    uint64
}

func DlcContractFromBytes(b []byte) (*DlcContract, error) {
	buf := bytes.NewBuffer(b)
	c := new(DlcContract)

	copy(c.OracleA[:], buf.Next(33))
	copy(c.OracleB[:], buf.Next(33))
	copy(c.OracleQ[:], buf.Next(33))

	_ = binary.Read(buf, binary.BigEndian, &c.OracleDataFeed)
	_ = binary.Read(buf, binary.BigEndian, &c.OracleTimestamp)
	_ = binary.Read(buf, binary.BigEndian, &c.ValueAllOurs)
	_ = binary.Read(buf, binary.BigEndian, &c.ValueAllTheirs)
	_ = binary.Read(buf, binary.BigEndian, &c.OurFundingAmount)
	_ = binary.Read(buf, binary.BigEndian, &c.TheirFundingAmount)

	copy(c.OurPayoutPKH[:], buf.Next(20))
	copy(c.TheirPayoutPKH[:], buf.Next(20))

	_ = binary.Read(buf, binary.BigEndian, &c.Status)

	var ourInputsLen uint32
	_ = binary.Read(buf, binary.BigEndian, &ourInputsLen)

	c.OurFundingInputs = make([]DlcContractFundingInput, ourInputsLen)
	var op [36]byte
	for i := uint32(0); i < ourInputsLen; i++ {
		copy(op[:], buf.Next(36))
		c.OurFundingInputs[i].Outpoint = *lnutil.OutPointFromBytes(op)
		_ = binary.Read(buf, binary.BigEndian, &c.OurFundingInputs[i].Value)
	}

	var theirInputsLen uint32
	_ = binary.Read(buf, binary.BigEndian, &theirInputsLen)

	c.TheirFundingInputs = make([]DlcContractFundingInput, theirInputsLen)
	for i := uint32(0); i < ourInputsLen; i++ {
		copy(op[:], buf.Next(36))
		c.OurFundingInputs[i].Outpoint = *lnutil.OutPointFromBytes(op)
		_ = binary.Read(buf, binary.BigEndian, &c.OurFundingInputs[i].Value)
	}

	return c, nil
}

func (self *DlcContract) Bytes() []byte {
	var buf bytes.Buffer

	buf.Write(self.OracleA[:])
	buf.Write(self.OracleB[:])
	buf.Write(self.OracleQ[:])
	binary.Write(&buf, binary.BigEndian, self.OracleDataFeed)
	binary.Write(&buf, binary.BigEndian, self.OracleTimestamp)
	binary.Write(&buf, binary.BigEndian, self.ValueAllOurs)
	binary.Write(&buf, binary.BigEndian, self.ValueAllTheirs)
	binary.Write(&buf, binary.BigEndian, self.OurFundingAmount)
	binary.Write(&buf, binary.BigEndian, self.TheirFundingAmount)
	buf.Write(self.OurPayoutPKH[:])
	buf.Write(self.TheirPayoutPKH[:])
	binary.Write(&buf, binary.BigEndian, self.Status)

	ourInputsLen := uint32(len(self.OurFundingInputs))
	binary.Write(&buf, binary.BigEndian, ourInputsLen)

	for i := 0; i < len(self.OurFundingInputs); i++ {
		opArr := lnutil.OutPointToBytes(self.OurFundingInputs[i].Outpoint)
		buf.Write(opArr[:])
		binary.Write(&buf, binary.BigEndian, self.OurFundingInputs[i].Value)
	}

	theirInputsLen := uint32(len(self.TheirFundingInputs))
	binary.Write(&buf, binary.BigEndian, theirInputsLen)

	for i := 0; i < len(self.TheirFundingInputs); i++ {
		opArr := lnutil.OutPointToBytes(self.TheirFundingInputs[i].Outpoint)
		buf.Write(opArr[:])
		binary.Write(&buf, binary.BigEndian, self.TheirFundingInputs[i].Value)
	}

	return buf.Bytes()
}

// Starts a new draft contract
func (mgr *DlcManager) AddContract() (*DlcContract, error) {
	var err error

	c := new(DlcContract)
	c.Status = ContractStatusDraft
	err = mgr.SaveContract(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (mgr *DlcManager) SetContractOracle(cIdx, oIdx uint64) error {
	o, err := mgr.LoadOracle(oIdx)
	if err != nil {
		return err
	}

	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	c.OracleA = o.A
	c.OracleB = o.B
	c.OracleQ = o.Q

	mgr.SaveContract(c)

	return nil
}

func (mgr *DlcManager) SetContractSettlementTime(cIdx, time uint64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	c.OracleTimestamp = time

	mgr.SaveContract(c)

	return nil
}

func (mgr *DlcManager) SetContractDatafeed(cIdx, feed uint64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	c.OracleDataFeed = feed

	mgr.SaveContract(c)

	return nil
}

func (mgr *DlcManager) SetContractFunding(cIdx, ourAmount, theirAmount uint64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	c.OurFundingAmount = ourAmount
	c.TheirFundingAmount = theirAmount

	mgr.SaveContract(c)

	return nil
}

func (mgr *DlcManager) SetContractSettlementDivision(cIdx, valueAllOurs, valueAllTheirs uint64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	c.ValueAllOurs = valueAllOurs
	c.ValueAllTheirs = valueAllTheirs

	mgr.SaveContract(c)

	return nil
}
