package lnutil

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	White = color.New(color.FgHiWhite).SprintFunc()
	Green = color.New(color.FgHiGreen).SprintFunc()
	Red   = color.New(color.FgHiRed).SprintFunc()

	Header   = color.New(color.FgHiCyan).SprintFunc()
	Prompt   = color.New(color.FgHiYellow).SprintFunc()
	OutPoint = color.New(color.FgYellow).SprintFunc()
	Address  = color.New(color.FgMagenta).SprintFunc()
	BTC      = color.New(color.FgHiWhite).SprintFunc()
	Satoshi  = color.New(color.Faint).SprintFunc()
	MicroBTC = color.New(color.FgYellow, color.Faint).SprintFunc()
)

func ReqColor(required ...interface{}) string {
	var s string
	for i := 0; i < len(required); i++ {
		s += " <"
		s += White(required[i])
		s += ">"
	}
	return s
}

func OptColor(optional ...interface{}) string {
	var s string
	var tail string
	for i := 0; i < len(optional); i++ {
		s += " [<"
		s += White(optional[i])
		s += ">"
		tail += "]"
	}
	return s + tail
}

func SatoshiColor(value int64) string {

	uBTC := value / 100
	mBTC := uBTC / 1000
	btc := mBTC / 1000

	sat := value

	if uBTC < 1 {
		return fmt.Sprintf("%s", Satoshi(sat))
	}

	uBTC -= (mBTC * 1000)
	sat -= (uBTC * 100)

	if mBTC < 1 {
		return fmt.Sprintf("%s%s", MicroBTC(uBTC), Satoshi(fmt.Sprintf("%02d", sat)))
	}

	mBTC -= (btc * 1000)
	sat -= (mBTC * 100000)

	if btc < 1 {
		return fmt.Sprintf("%d%s%s",
			mBTC,
			MicroBTC(fmt.Sprintf("%03d", uBTC)),
			Satoshi(fmt.Sprintf("%02d", sat)))
	}

	sat -= btc * 100000000

	return fmt.Sprintf("%s%03d%s%s",
		BTC(btc),
		mBTC,
		MicroBTC(fmt.Sprintf("%03d", uBTC)),
		Satoshi(fmt.Sprintf("%02d", sat)))
}
