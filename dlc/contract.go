package dlc

import (
	"bytes"
	"crypto/rand"
	"fmt"

	"github.com/mit-dci/lit/lnutil"
)

// Starts a new draft contract
func (mgr *DlcManager) AddContract() (*lnutil.DlcContract, error) {
	var err error

	c := new(lnutil.DlcContract)
	c.Status = lnutil.ContractStatusDraft
	rand.Read(c.PubKey[:])
	err = mgr.SaveContract(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (mgr *DlcManager) FindContractByKey(key [33]byte) (*lnutil.DlcContract, error) {
	contracts, err := mgr.ListContracts()
	if err != nil {
		return nil, err
	}

	for _, c := range contracts {
		if bytes.Equal(c.PubKey[:], key[:]) {
			return c, nil
		}
	}

	return nil, fmt.Errorf("Contract not found")
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

	if c.OracleTimestamp == 0 {
		return fmt.Errorf("You need to set the settlement timestamp first, otherwise no R point can be retrieved for the feed")
	}

	o, err := mgr.FindOracleByKey(c.OracleA)
	if err != nil {
		return err
	}

	c.OracleR, err = o.FetchRPoint(feed, c.OracleTimestamp)
	if err != nil {
		return err
	}

	err = mgr.SaveContract(c)
	if err != nil {
		return err
	}
	return nil
}

func (mgr *DlcManager) SetContractRPoint(cIdx uint64, rPoint [33]byte) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	c.OracleR = rPoint

	err = mgr.SaveContract(c)
	if err != nil {
		return err
	}

	return nil
}

func (mgr *DlcManager) SetContractFunding(cIdx uint64, ourAmount, theirAmount int64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	c.OurFundingAmount = ourAmount
	c.TheirFundingAmount = theirAmount

	mgr.SaveContract(c)

	return nil
}

func (mgr *DlcManager) SetContractSettlementDivision(cIdx uint64, valueAllOurs, valueAllTheirs int64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	rangeMin := valueAllTheirs - (valueAllOurs - valueAllTheirs)
	rangeMax := valueAllOurs + (valueAllOurs - valueAllTheirs)
	oursHighest := true
	if valueAllTheirs > valueAllOurs {
		oursHighest = false
		rangeMin = valueAllOurs - (valueAllTheirs - valueAllOurs)
		rangeMax = valueAllTheirs + (valueAllTheirs - valueAllOurs)
	}
	if rangeMin < 0 {
		rangeMin = 0
	}

	fmt.Printf("Creating division from %d to %d\n", rangeMin, rangeMax)

	totalContractValue := c.OurFundingAmount + c.TheirFundingAmount

	c.Division = make([]lnutil.DlcContractDivision, rangeMax-rangeMin+1)
	for i := rangeMin; i <= rangeMax; i++ {
		c.Division[i-rangeMin].OracleValue = i

		if (oursHighest && i >= valueAllOurs) || (!oursHighest && i <= valueAllOurs) {
			c.Division[i-rangeMin].ValueOurs = totalContractValue
			continue
		}

		if (oursHighest && i <= valueAllTheirs) || (!oursHighest && i >= valueAllTheirs) {
			c.Division[i-rangeMin].ValueOurs = 0
			continue
		}

		if oursHighest {
			c.Division[i-rangeMin].ValueOurs = int64(float64(totalContractValue) / float64(valueAllOurs-valueAllTheirs) * float64(i-valueAllTheirs))
		} else {
			c.Division[i-rangeMin].ValueOurs = int64(totalContractValue) - int64(float64(totalContractValue)/float64(valueAllTheirs-valueAllOurs)*float64(i-valueAllOurs))
		}
	}
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
