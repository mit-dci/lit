package lnutil

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/adiabat/btcd/wire"
)

type DlcContractStatus int

const (
	ContractStatusDraft       DlcContractStatus = 0
	ContractStatusOfferedByMe DlcContractStatus = 1
	ContractStatusOfferedToMe DlcContractStatus = 2
	ContractStatusDeclined    DlcContractStatus = 3
	ContractStatusActive      DlcContractStatus = 4
	ContractStatusClosed      DlcContractStatus = 5
)

type DlcContract struct {
	Idx                                  uint64                    // Index of the contract for referencing in commands
	CoinType                             uint32                    // Coin type
	OracleA, OracleB, OracleQ            [33]byte                  // Pub keys of the oracle
	OracleDataFeed, OracleTimestamp      uint64                    // The data feed and time we use for contract settlement
	ValueAllOurs, ValueAllTheirs         uint64                    // The value of the datafeed based on which all money in the contract goes to either party
	OurFundingAmount, TheirFundingAmount uint64                    // The amounts either side are funding
	OurPayoutPKH, TheirPayoutPKH         [20]byte                  // PKH to which the contracts are supposed to pay out
	RemoteNodePub                        [33]byte                  // Pubkey of the peer we've offered the contract to or received the contract from
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

	var status int32
	_ = binary.Read(buf, binary.BigEndian, &status)
	fmt.Printf("Read byte for status: %d\n", status)

	c.Status = DlcContractStatus(status)

	var ourInputsLen uint32
	_ = binary.Read(buf, binary.BigEndian, &ourInputsLen)

	c.OurFundingInputs = make([]DlcContractFundingInput, ourInputsLen)
	var op [36]byte
	for i := uint32(0); i < ourInputsLen; i++ {
		copy(op[:], buf.Next(36))
		c.OurFundingInputs[i].Outpoint = *OutPointFromBytes(op)
		_ = binary.Read(buf, binary.BigEndian, &c.OurFundingInputs[i].Value)
	}

	var theirInputsLen uint32
	_ = binary.Read(buf, binary.BigEndian, &theirInputsLen)

	c.TheirFundingInputs = make([]DlcContractFundingInput, theirInputsLen)
	for i := uint32(0); i < ourInputsLen; i++ {
		copy(op[:], buf.Next(36))
		c.OurFundingInputs[i].Outpoint = *OutPointFromBytes(op)
		_ = binary.Read(buf, binary.BigEndian, &c.OurFundingInputs[i].Value)
	}

	_ = binary.Read(buf, binary.BigEndian, &c.CoinType)
	copy(c.RemoteNodePub[:], buf.Next(33))
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
	var status = int32(self.Status)
	fmt.Printf("Writing byte for status: %d\n", status)
	binary.Write(&buf, binary.BigEndian, status)

	ourInputsLen := uint32(len(self.OurFundingInputs))
	binary.Write(&buf, binary.BigEndian, ourInputsLen)

	for i := 0; i < len(self.OurFundingInputs); i++ {
		opArr := OutPointToBytes(self.OurFundingInputs[i].Outpoint)
		buf.Write(opArr[:])
		binary.Write(&buf, binary.BigEndian, self.OurFundingInputs[i].Value)
	}

	theirInputsLen := uint32(len(self.TheirFundingInputs))
	binary.Write(&buf, binary.BigEndian, theirInputsLen)

	for i := 0; i < len(self.TheirFundingInputs); i++ {
		opArr := OutPointToBytes(self.TheirFundingInputs[i].Outpoint)
		buf.Write(opArr[:])
		binary.Write(&buf, binary.BigEndian, self.TheirFundingInputs[i].Value)
	}

	binary.Write(&buf, binary.BigEndian, self.CoinType)

	buf.Write(self.RemoteNodePub[:])

	return buf.Bytes()
}
