package dlc

import (
	"bytes"
	"crypto/rand"
	"fmt"

	"github.com/mit-dci/lit/lnutil"
)

// AddContract starts a new draft contract
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

// FindContractByKey finds a contract by its generated PubKey
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

// SetContractOracle assigns a particular oracle to a contract - used for determining which pubkey A to use and can also
// allow for fetching R-points automatically when the oracle was imported from a REST api
func (mgr *DlcManager) SetContractOracle(cIdx, oIdx uint64) error {

	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the oracle unless the contract is in Draft state")
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

// SetContractSettlementTime sets the unix epoch at which the oracle will sign a message containing the value the contract
// pays out on.
func (mgr *DlcManager) SetContractSettlementTime(cIdx, time uint64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the settlement time unless the contract is in Draft state")
	}

	c.OracleTimestamp = time

	// Reset the R point
	c.OracleR = [33]byte{}

	mgr.SaveContract(c)

	return nil
}

// SetContractDatafeed will automatically fetch the R-point from the REST API, if an oracle is imported from a REST API.
// You need to set the settlement time first, becuase the R point is a key unique for the time and feed
func (mgr *DlcManager) SetContractDatafeed(cIdx, feed uint64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the Datafeed unless the contract is in Draft state")
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

// SetContractRPoint allows you to manually set the R-point key if an oracle is not imported from a REST API
func (mgr *DlcManager) SetContractRPoint(cIdx uint64, rPoint [33]byte) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the R-point unless the contract is in Draft state")
	}

	c.OracleR = rPoint

	err = mgr.SaveContract(c)
	if err != nil {
		return err
	}

	return nil
}

// SetContractFunding sets the funding to the contract. It will specify how much we (the offering party) are
// funding, as well as
func (mgr *DlcManager) SetContractFunding(cIdx uint64, ourAmount, theirAmount int64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the funding unless the contract is in Draft state")
	}

	c.OurFundingAmount = ourAmount
	c.TheirFundingAmount = theirAmount

	// If the funding changes, the division needs to be re-set.
	c.Division = nil

	mgr.SaveContract(c)

	return nil
}

// SetContractSettlementDivision sets the division of the contract settlement. If the oraclized value is valueAllOurs, then the entire
// value of the contract is payable to us. If the oraclized value is valueAllTheirs, then the entire value is paid to
// our peer. Between those, the value is divided linearly.
func (mgr *DlcManager) SetContractSettlementDivision(cIdx uint64, valueAllOurs, valueAllTheirs int64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the division unless the contract is in Draft state")
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

// SetContractCoinType sets the cointype for a particular contract
func (mgr *DlcManager) SetContractCoinType(cIdx uint64, cointype uint32) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the coin type unless the contract is in Draft state")
	}

	c.CoinType = cointype

	mgr.SaveContract(c)

	return nil
}
