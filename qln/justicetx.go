package qln

import (
	"bytes"
	"fmt"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/sig64"
)

/*
functions relating to the "justice transaction" (aka penalty transaction)


because we're using the sipa/schnorr delinearization, we don't need to vary the PKH
anymore.  We can hand over 1 point per commit & figure everything out from that.
*/

// BuildWatchTxidSig builds the partial txid and signature pair which can
// be exported to the watchtower.
// This get a channel that is 1 state old.  So we can produce a signature.
func (nd *LnNode) BuildWatchTxidSig(q *Qchan) ([80]byte, error) {
	var parTxidSig [80]byte // 16 byte txid and 64 byte signature stuck together

	// in this function, "bad" refers to the hypothetical transaction spending the
	// com tx.  "justice" is the tx spending the bad tx

	fee := int64(5000) // fixed fee for now

	// build the bad tx
	badTx, err := q.BuildStateTx(false)

	// re-build the script.  redundant as we just did this in BuildStateTx, but
	// we need the preimage.

	// first we need the keys in the bad script.  Start by getting the elk-scalar
	// we should have it at the "current" state number
	elk, err := q.ElkRcv.AtIndex(q.State.StateIdx)
	if err != nil {
		return parTxidSig, err
	}
	// get elk scalar
	elkScalar := ElkScalar(elk)
	// get elk point
	elkPoint := ElkPointFromHash(&elkScalar)

	// build script to store in porTxo, make pubkeys
	badTimeoutPub := lnutil.AddPubsEZ(q.MyHAKDBase, elkPoint)
	badRevokePub := lnutil.CombinePubs(q.TheirHAKDBase, elkPoint)
	script := lnutil.P2WSHify(
		lnutil.CommitScript(badRevokePub, badTimeoutPub, q.TimeOut))

	var badAmt int64
	badIdx := uint32(len(badTx.TxOut) + 1)

	// figure out which output to bring justice to
	for i, out := range badTx.TxOut {
		if bytes.Equal(out.PkScript, script) {
			badIdx = uint32(i)
			badAmt = out.Value
			break
		}
	}
	if badIdx > uint32(len(badTx.TxOut)) {
		return parTxidSig, fmt.Errorf("BuildWatchTxidSig couldn't find revocable SH output")
	}

	// make a keygen to get the private HAKD base scalar
	kg := q.KeyGen
	kg.Step[2] = UseChannelHAKDBase
	// get HAKD base scalar
	privBase := nd.BaseWallet.GetPriv(kg)
	// combine elk & HAKD base to make signing key
	combinedPrivKey := lnutil.CombinePrivKeyWithBytes(privBase, elkScalar[:])

	// get badtxid
	badTxid := badTx.TxHash()
	// make bad outpoint
	badOP := wire.NewOutPoint(&badTxid, badIdx)
	// make the justice txin, empty sig / witness
	justiceIn := wire.NewTxIn(badOP, nil, nil)
	// make justice output script
	justiceScript := lnutil.DirectWPKHScriptFromPKH(q.WatchRefundAdr)
	// make justice txout
	justiceOut := wire.NewTxOut(badAmt-fee, justiceScript)

	justiceTx := wire.NewMsgTx()
	// set to version 2, though might not matter as no CSV is used
	justiceTx.Version = 2

	// add inputs and outputs
	justiceTx.AddTxIn(justiceIn)
	justiceTx.AddTxOut(justiceOut)

	// get hashcache for signing
	hCache := txscript.NewTxSigHashes(justiceTx)

	// sign with combined key.  Justice txs always have only 1 input, so txin is 0
	bigSig, err := txscript.RawTxInWitnessSignature(
		justiceTx, hCache, 0, q.Value, script, txscript.SigHashAll, combinedPrivKey)
	// truncate sig (last byte is sighash type, always sighashAll)
	bigSig = bigSig[:len(bigSig)-1]

	sig, err := sig64.SigCompress(bigSig)
	if err != nil {
		return parTxidSig, err
	}

	copy(parTxidSig[:16], badTxid[:16])
	copy(parTxidSig[16:], sig[:])

	return parTxidSig, nil
}
