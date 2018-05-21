package dlc

import (
	"fmt"

	"github.com/mit-dci/lit/lnutil"
)

const COINTYPE_NOT_SET = ^uint32(0) // Max Uint

// AddContract starts a new draft contract
func (mgr *DlcManager) AddContract() (*lnutil.DlcContract, error) {
	var err error

	c := new(lnutil.DlcContract)
	c.Status = lnutil.ContractStatusDraft
	c.CoinType = COINTYPE_NOT_SET
	err = mgr.SaveContract(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// SetContractOracle assigns a particular oracle to a contract - used for
// determining which pubkey A to use and can also allow for fetching R-points
// automatically when the oracle was imported from a REST api
func (mgr *DlcManager) SetContractOracle(cIdx, oIdx uint64) error {

	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the oracle unless the" +
			" contract is in Draft state")
	}

	o, err := mgr.LoadOracle(oIdx)
	if err != nil {
		return err
	}

	c.OracleA = o.A

	// Reset the R point when changing the oracle
	c.OracleR = [33]byte{}

	mgr.SaveContract(c)

	return nil
}


// SetContractDatafeed will automatically fetch the R-point from the REST API,
// if an oracle is imported from a REST API. You need to set the settlement time
// first, becuase the R point is a key unique for the time and feed
func (mgr *DlcManager) SetContractDatafeed(cIdx, feed uint64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the Datafeed unless the" +
			" contract is in Draft state")
	}

	if c.OracleTimestamp == 0 {
		return fmt.Errorf("You need to set the settlement timestamp first," +
			" otherwise no R point can be retrieved for the feed")
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

// SetContractRPoint allows you to manually set the R-point key if an oracle is
// not imported from a REST API
func (mgr *DlcManager) SetContractRPoint(cIdx uint64, rPoint [33]byte) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the R-point unless the" +
			" contract is in Draft state")
	}

	c.OracleR = rPoint

	err = mgr.SaveContract(c)
	if err != nil {
		return err
	}

	return nil
}

// SetContractFundingAndDivision sets the funding and division parameters
// to the contract. It will specify how much
// we (the offering party) are willing to fund, as well as what they are.
// It also sets the division of the contract settlement ie if the
// oraclized value is valueAllOurs, then the entire value of the contract is
// payable to us. If the oraclized value is valueAllTheirs, then the entire
// value is paid to our peer. Between those, the value is divided linearly.

func (mgr *DlcManager) SetContractFundingAndDivision(cIdx uint64, our, their,
	valueAllOurs, valueAllTheirs int64, time uint64, cointype uint32) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}
	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the funding unless the" +
			" contract is in Draft state")
	}

	c.CoinType = cointype
	c.OurFundingAmount = our
	c.TheirFundingAmount = their
	c.OracleTimestamp = time
	c.OracleR = [33]byte{} // Reset the R point
	c.Division = nil // If the funding changes, the division needs to be re-set.

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

	totalContractValue := c.OurFundingAmount + c.TheirFundingAmount
	fTotal := float64(totalContractValue)
	fRange := float64(valueAllOurs - valueAllTheirs)
	if !oursHighest {
		fRange = float64(valueAllTheirs - valueAllOurs)
	}
	c.Division = make([]lnutil.DlcContractDivision, rangeMax-rangeMin+1)
	for i := rangeMin; i <= rangeMax; i++ {
		c.Division[i-rangeMin].OracleValue = i

		if (oursHighest && i >= valueAllOurs) ||
			(!oursHighest && i <= valueAllOurs) {
			c.Division[i-rangeMin].ValueOurs = totalContractValue
			continue
		}

		if (oursHighest && i <= valueAllTheirs) ||
			(!oursHighest && i >= valueAllTheirs) {
			c.Division[i-rangeMin].ValueOurs = 0
			continue
		}

		idx := i - rangeMin
		if oursHighest {
			fCurInRange := float64(i - valueAllTheirs)
			c.Division[idx].ValueOurs = int64(fTotal / fRange * fCurInRange)
		} else {
			fCurInRange := float64(i - valueAllOurs)
			c.Division[idx].ValueOurs = int64(totalContractValue)
			c.Division[idx].ValueOurs -= int64(fTotal / fRange * fCurInRange)

		}

	}

	mgr.SaveContract(c)
	return nil
}
