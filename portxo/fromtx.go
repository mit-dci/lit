package portxo

import (
	"fmt"

	"github.com/mit-dci/lit/wire"
)

// ExtractFromTx returns a portxo from a tx and index.
// It fills in what it can, but the keygen, sequence, and height can't
// be determined from just this and need to be put in separately.
func ExtractFromTx(tx *wire.MsgTx, idx uint32) (*PorTxo, error) {
	if tx == nil {
		return nil, fmt.Errorf("nil tx")
	}
	if int(idx) > len(tx.TxOut)-1 {
		return nil, fmt.Errorf("extract txo %d but tx has %d outputs",
			idx, len(tx.TxOut))
	}

	u := new(PorTxo)

	u.Op.Hash = tx.TxHash()
	u.Op.Index = idx

	u.Value = tx.TxOut[idx].Value

	u.Mode = TxoUnknownMode // default to unknown mode

	// check if mode can be determined from the pkscript
	u.Mode = TxoModeFromPkScript(tx.TxOut[idx].PkScript)

	// copy pkscript into portxo
	u.PkScript = tx.TxOut[idx].PkScript

	//done
	return u, nil
}

func TxoModeFromPkScript(script []byte) TxoMode {
	// start with unknown
	var mode TxoMode

	mode = TxoUnknownMode

	if script == nil {
		return mode
	}

	// check for p2pk
	if len(script) == 35 && script[0] == 0x21 && script[34] == 0xac {
		mode = TxoP2PKComp
	}

	// check for p2pkh
	if len(script) == 25 && script[0] == 0x76 && script[1] == 0xa9 &&
		script[2] == 0x14 && script[23] == 0x88 && script[24] == 0xac {

		mode = TxoP2PKHComp // assume compressed
	}

	// check for witness key hash
	if len(script) == 22 && script[0] == 0x00 && script[1] == 0x14 {
		mode = TxoP2WPKHComp // assume compressed
	}

	// check for witness script hash
	if len(script) == 34 && script[0] == 0x00 && script[1] == 0x20 {
		mode = TxoP2WSHComp // does compressed even mean anything for SH..?
	}

	// couldn't find anything, unknown
	return mode
}
