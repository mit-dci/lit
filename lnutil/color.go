package lnutil

import (
    "fmt"

    "github.com/fatih/color"
)

var (
    White = color.New(color.FgHiWhite).SprintFunc()
    Cyan = color.New(color.FgHiCyan).SprintFunc()
    Yellow = color.New(color.FgHiYellow).SprintFunc()
    Red = color.New(color.FgHiRed).SprintFunc()
    Green = color.New(color.FgHiGreen).SprintFunc()
    WhiteUL = color.New(color.FgHiWhite).Add(color.Underline).SprintFunc()
    Faint = color.New(color.Faint).SprintFunc()
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
    var mBTC = value / 100000
    if mBTC < 1 {
        return Faint(value)
    }
    var sat = value - (mBTC * 100000)
    var btc = mBTC / 1000
    mBTC -= (btc * 1000)
    if btc < 1 {
        return fmt.Sprintf("%d%s", mBTC, Faint(fmt.Sprintf("%05d", sat)))
    }

    return fmt.Sprintf("%s%03d%s", WhiteUL(btc), mBTC, Faint(fmt.Sprintf("%05d", sat)))
}
