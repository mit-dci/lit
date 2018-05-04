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

// SetContractSettlementTime sets the unix epoch at which the oracle will sign a
// message containing the value the contract pays out on.
func (mgr *DlcManager) SetContractSettlementTime(cIdx, time uint64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the settlement time" +
			" unless the contract is in Draft state")
	}

	c.OracleTimestamp = time

	// Reset the R point
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

// SetContractFunding sets the funding to the contract. It will specify how much
// we (the offering party) are funding, as well as
func (mgr *DlcManager) SetContractFunding(cIdx uint64, our, their int64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the funding unless the" +
			" contract is in Draft state")
	}

	c.OurFundingAmount = our
	c.TheirFundingAmount = their

	// If the funding changes, the division needs to be re-set.
	c.Division = nil

	mgr.SaveContract(c)

	return nil
}

// Build payouts populates the payout schedule for a forward offer
func BuildPayouts(o *lnutil.DlcFwdOffer) error {

	var price, maxPrice, oracleStep int64

	// clear out payout schedule just in case one's already there
	o.Payouts = make([]lnutil.DlcContractDivision, 0)

	// use a coarse oracle that rounds asset prices to the neares 100 satoshis
	// the oracle must also use the same price stepping
	oracleStep = 100
	// max out at 100K (corresponds to asset price of 1000 per bitcoin)
	// this is very ugly to hard-code here but we can leave it until we implement
	// a more complex oracle price signing, such as base/mantissa.
	maxPrice = 100000

	for price = 0; price <= maxPrice; price += oracleStep {
		var div lnutil.DlcContractDivision
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

// SetContractDivision sets the division of the contract settlement. If the
// oraclized value is valueAllOurs, then the entire value of the contract is
// payable to us. If the oraclized value is valueAllTheirs, then the entire
// value is paid to our peer. Between those, the value is divided linearly.
func (mgr *DlcManager) SetContractDivision(cIdx uint64,
	valueAllOurs, valueAllTheirs int64) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the division unless" +
			" the contract is in Draft state")
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

// SetContractCoinType sets the cointype for a particular contract
func (mgr *DlcManager) SetContractCoinType(cIdx uint64, cointype uint32) error {
	c, err := mgr.LoadContract(cIdx)
	if err != nil {
		return err
	}

	if c.Status != lnutil.ContractStatusDraft {
		return fmt.Errorf("You cannot change or set the coin type unless" +
			" the contract is in Draft state")
	}

	c.CoinType = cointype

	mgr.SaveContract(c)

	return nil
}
