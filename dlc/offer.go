package dlc

import (
	"github.com/mit-dci/lit/lnutil"
)

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
