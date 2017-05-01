package lnutil

import (
	"bytes"
	"fmt"

	"github.com/adiabat/btcd/txscript"
	"github.com/adiabat/btcd/wire"
)

// CommitScript is the script for 0.13.1: OP_CHECKSIG turned into OP_CHECSIGVERIFY
func CommitScript(RKey, TKey [33]byte, delay uint16) []byte {
	builder := txscript.NewScriptBuilder()

	// 1 for penalty / revoked, 0 for timeout
	// 1, so timeout
	builder.AddOp(txscript.OP_IF)

	// Just push revokable key
	builder.AddData(RKey[:])

	// 0, so revoked
	builder.AddOp(txscript.OP_ELSE)

	// CSV delay
	builder.AddInt64(int64(delay))
	// CSV check, fails here if too early
	builder.AddOp(txscript.OP_NOP3) // really OP_CHECKSEQUENCEVERIFY
	// Drop delay value
	builder.AddOp(txscript.OP_DROP)
	// push timeout key
	builder.AddData(TKey[:])

	builder.AddOp(txscript.OP_ENDIF)

	// check whatever pubkey is left on the stack
	builder.AddOp(txscript.OP_CHECKSIG)

	// never any errors we care about here.
	s, _ := builder.Script()
	return s
}

// FundMultiPre generates the non-p2sh'd multisig script for 2 of 2 pubkeys.
// useful for making transactions spending the fundtx.
// returns a bool which is true if swapping occurs.
func FundTxScript(aPub, bPub [33]byte) ([]byte, bool, error) {
	var swapped bool
	if bytes.Compare(aPub[:], bPub[:]) == -1 { // swap to sort pubkeys if needed
		aPub, bPub = bPub, aPub
		swapped = true
	}
	bldr := txscript.NewScriptBuilder()
	// Require 1 signatures, either key// so from both of the pubkeys
	bldr.AddOp(txscript.OP_2)
	// add both pubkeys (sorted)
	bldr.AddData(aPub[:])
	bldr.AddData(bPub[:])
	// 2 keys total.  In case that wasn't obvious.
	bldr.AddOp(txscript.OP_2)
	// Good ol OP_CHECKMULTISIG.  Don't forget the zero!
	bldr.AddOp(txscript.OP_CHECKMULTISIG)
	// get byte slice
	pre, err := bldr.Script()
	return pre, swapped, err
}

// FundTxOut creates a TxOut for the funding transaction.
// Give it the two pubkeys and it'll give you the p2sh'd txout.
// You don't have to remember the p2sh preimage, as long as you remember the
// pubkeys involved.
func FundTxOut(pubA, pubB [33]byte, amt int64) (*wire.TxOut, error) {
	if amt < 0 {
		return nil, fmt.Errorf("Can't create FundTx script with negative coins")
	}
	scriptBytes, _, err := FundTxScript(pubA, pubB)
	if err != nil {
		return nil, err
	}
	scriptBytes = P2WSHify(scriptBytes)

	return wire.NewTxOut(amt, scriptBytes), nil
}
