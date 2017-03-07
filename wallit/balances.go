package wallit

import (
	"fmt"

	"github.com/mit-dci/lit/portxo"
)

// HowMuchTotal returns the amount of all known current utxo.
// Returns -1 if there's an error.
func (w *Wallit) HowMuchTotal() int64 {
	allTxos, err := w.GetAllUtxos()
	if err != nil {
		return -1
	}

	var sum int64
	// iterate through utxos to figure out how much we have
	for _, u := range allTxos {
		sum += u.Value
	}
	return sum
}

// HowMuchWitConf returns the amount of confirmed, mature, witness utxo.
// (for building channels)
// Returns -1 if there's an error.
func (w *Wallit) HowMuchWitConf() int64 {
	currentHeight, err := w.GetDBSyncHeight()
	if err != nil {
		fmt.Printf(err.Error())
		return -1
	}

	allTxos, err := w.GetAllUtxos()
	if err != nil {
		return -1
	}
	var sum int64

	// iterate through utxos to figure out how much we have
	for _, u := range allTxos {
		// first, make sure it's witty
		if u.Mode&portxo.FlagTxoWitness != 0 {
			// then make sure it's confirmed, and any timeouts have passed
			if u.Height > 0 && u.Height+int32(u.Seq) <= currentHeight {
				sum += u.Value
			}
		}
	}

	return sum
}
