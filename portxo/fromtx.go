package portxo

import (
	"fmt"

	"github.com/btcsuite/btcd/wire"
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
	pks := tx.TxOut[idx].PkScript

	// check for p2pkh
	if len(pks) == 25 && pks[0] == 0x76 && pks[1] == 0xa9 && pks[2] == 0x14 &&
		pks[23] == 0x88 && pks[24] == 0xac {

		u.Mode = TxoP2PKHComp // assume compressed

	}

	// check for witness key hash
	if len(pks) == 22 && pks[0] == 0x00 && pks[1] == 0x14 {
		u.Mode = TxoP2WPKHComp // assume compressed
	}

	// check for witness script hash
	if len(pks) == 34 && pks[0] == 0x00 && pks[1] == 0x20 {
		u.Mode = TxoP2WSHComp // does compressed even mean anything for SH..?
	}

	// copy pkscript into portxo
	u.PkScript = tx.TxOut[idx].PkScript

	//done
	return u, nil
}
