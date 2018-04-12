package dlc

import (
	"github.com/mit-dci/lit/lnutil"
)

// Starts a new draft contract
func (mgr *DlcManager) AddContract() (*lnutil.DlcContract, error) {
	var err error

	c := new(lnutil.DlcContract)
	c.Status = lnutil.ContractStatusDraft
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

func (mgr *DlcManager) SetContractCoinType(cIdx uint64, cointype uint32) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	c.CoinType = cointype

	mgr.SaveContract(c)

	return nil
}
