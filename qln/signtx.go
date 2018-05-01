package qln

import (
	"fmt"

	"github.com/adiabat/btcd/btcec"
	"github.com/adiabat/btcd/txscript"
	"github.com/adiabat/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/sig64"
)

// SignBreak signs YOUR tx, which you already have a sig for
func (nd *LitNode) SignBreakTx(q *Qchan) (*wire.MsgTx, error) {
	tx, err := q.BuildStateTx(true)
	if err != nil {
		return nil, err
	}

	// make hash cache for this tx
	hCache := txscript.NewTxSigHashes(tx)

	// generate script preimage (keep track of key order)
	pre, swap, err := lnutil.FundTxScript(q.MyPub, q.TheirPub)
	if err != nil {
		return nil, err
	}

	// get private signing key
	priv := nd.SubWallet[q.Coin()].GetPriv(q.KeyGen)
	// generate sig.
	mySig, err := txscript.RawTxInWitnessSignature(
		tx, hCache, 0, q.Value, pre, txscript.SigHashAll, priv)

	theirSig := sig64.SigDecompress(q.State.sig)
	// put the sighash all byte on the end of their signature
	theirSig = append(theirSig, byte(txscript.SigHashAll))

	fmt.Printf("made mysig: %x theirsig: %x\n", mySig, theirSig)
	// add sigs to the witness stack
	if swap {
		tx.TxIn[0].Witness = SpendMultiSigWitStack(pre, theirSig, mySig)
	} else {
		tx.TxIn[0].Witness = SpendMultiSigWitStack(pre, mySig, theirSig)
	}

	// save channel state as closed
	q.CloseData.Closed = true
	q.CloseData.CloseTxid = tx.TxHash()
	err = nd.SaveQchanUtxoData(q)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// SignSimpleClose signs the given simpleClose tx, given the other signature
// Tx is modified in place.
func (nd *LitNode) SignSimpleClose(q *Qchan, tx *wire.MsgTx) ([64]byte, error) {

	var sig [64]byte
	// make hash cache
	hCache := txscript.NewTxSigHashes(tx)

	// generate script preimage for signing (ignore key order)
	pre, _, err := lnutil.FundTxScript(q.MyPub, q.TheirPub)
	if err != nil {
		return sig, err
	}
	// get private signing key
	priv := nd.SubWallet[q.Coin()].GetPriv(q.KeyGen)
	// generate sig
	mySig, err := txscript.RawTxInWitnessSignature(
		tx, hCache, 0, q.Value, pre, txscript.SigHashAll, priv)
	if err != nil {
		return sig, err
	}
	// truncate sig (last byte is sighash type, always sighashAll)
	mySig = mySig[:len(mySig)-1]
	return sig64.SigCompress(mySig)
}

// SignSettlementTx signs the given settlement tx based on the passed contract
// using the passed private key. Tx is modified in place.
func (nd *LitNode) SignSettlementTx(c *lnutil.DlcContract, tx *wire.MsgTx,
	priv *btcec.PrivateKey) ([64]byte, error) {

	var sig [64]byte
	// make hash cache
	hCache := txscript.NewTxSigHashes(tx)

	// generate script preimage for signing (ignore key order)
	pre, _, err := lnutil.FundTxScript(c.OurFundMultisigPub,
		c.TheirFundMultisigPub)

	if err != nil {
		return sig, err
	}
	// generate sig
	mySig, err := txscript.RawTxInWitnessSignature(
		tx, hCache, 0, c.TheirFundingAmount+c.OurFundingAmount,
		pre, txscript.SigHashAll, priv)

	if err != nil {
		return sig, err
	}
	// truncate sig (last byte is sighash type, always sighashAll)
	mySig = mySig[:len(mySig)-1]
	return sig64.SigCompress(mySig)
}

// SignClaimTx signs the given claim tx based on the passed preimage and value
// using the passed private key. Tx is modified in place. timeout=false means
// it's a regular claim, timeout=true means we're claiming an output that has
// expired (for instance if someone) published the wrong settlement TX, we can
// claim this output back to our wallet after the timelock expired.
func (nd *LitNode) SignClaimTx(claimTx *wire.MsgTx, value int64, pre []byte,
	priv *btcec.PrivateKey, timeout bool) error {

	// make hash cache
	hCache := txscript.NewTxSigHashes(claimTx)

	// generate sig
	mySig, err := txscript.RawTxInWitnessSignature(
		claimTx, hCache, 0, value, pre, txscript.SigHashAll, priv)
	if err != nil {
		return err
	}

	witStash := make([][]byte, 3)
	witStash[0] = mySig
	if timeout {
		witStash[1] = nil
	} else {
		witStash[1] = []byte{0x01}
	}
	witStash[2] = pre
	claimTx.TxIn[0].Witness = witStash
	return nil
}

// SignNextState generates your signature for their state.
func (nd *LitNode) SignState(q *Qchan) ([64]byte, error) {

	var sig [64]byte

	// make sure channel exists, and wallet is present on node
	if q == nil {
		return sig, fmt.Errorf("SignState nil channel")
	}
	_, ok := nd.SubWallet[q.Coin()]
	if !ok {
		return sig, fmt.Errorf("SignState no wallet for cointype %d", q.Coin())
	}
	// build transaction for next state
	tx, err := q.BuildStateTx(false) // their tx, as I'm signing
	if err != nil {
		return sig, err
	}

	// make hash cache for this tx
	hCache := txscript.NewTxSigHashes(tx)

	// generate script preimage (ignore key order)
	pre, _, err := lnutil.FundTxScript(q.MyPub, q.TheirPub)
	if err != nil {
		return sig, err
	}

	// get private signing key
	priv := nd.SubWallet[q.Coin()].GetPriv(q.KeyGen)

	// generate sig.
	bigSig, err := txscript.RawTxInWitnessSignature(
		tx, hCache, 0, q.Value, pre, txscript.SigHashAll, priv)
	// truncate sig (last byte is sighash type, always sighashAll)
	bigSig = bigSig[:len(bigSig)-1]

	sig, err = sig64.SigCompress(bigSig)
	if err != nil {
		return sig, err
	}

	fmt.Printf("____ sig creation for channel (%d,%d):\n", q.Peer(), q.Idx())
	fmt.Printf("\tinput %s\n", tx.TxIn[0].PreviousOutPoint.String())
	for i, txout := range tx.TxOut {
		fmt.Printf("\toutput %d: %x %d\n", i, txout.PkScript, txout.Value)
	}
	fmt.Printf("\tstate %d myamt: %d theiramt: %d\n", q.State.StateIdx, q.State.MyAmt, q.Value-q.State.MyAmt)

	return sig, nil
}

// VerifySig verifies their signature for your next state.
// it also saves the sig if it's good.
// do bool, error or just error?  Bad sig is an error I guess.
// for verifying signature, always use theirHAKDpub, so generate & populate within
// this function.
func (q *Qchan) VerifySig(sig [64]byte) error {

	bigSig := sig64.SigDecompress(sig)
	// my tx when I'm verifying.
	tx, err := q.BuildStateTx(true)
	if err != nil {
		return err
	}

	// generate fund output script preimage (ignore key order)
	pre, _, err := lnutil.FundTxScript(q.MyPub, q.TheirPub)
	if err != nil {
		return err
	}

	hCache := txscript.NewTxSigHashes(tx)

	parsed, err := txscript.ParseScript(pre)
	if err != nil {
		return err
	}
	// always sighash all
	hash := txscript.CalcWitnessSignatureHash(
		parsed, hCache, txscript.SigHashAll, tx, 0, q.Value)

	// sig is pre-truncated; last byte for sighashtype is always sighashAll
	pSig, err := btcec.ParseDERSignature(bigSig, btcec.S256())
	if err != nil {
		return err
	}
	theirPubKey, err := btcec.ParsePubKey(q.TheirPub[:], btcec.S256())
	if err != nil {
		return err
	}
	fmt.Printf("____ sig verification for channel (%d,%d):\n", q.Peer(), q.Idx())
	fmt.Printf("\tinput %s\n", tx.TxIn[0].PreviousOutPoint.String())
	for i, txout := range tx.TxOut {
		fmt.Printf("\toutput %d: %x %d\n", i, txout.PkScript, txout.Value)
	}
	fmt.Printf("\tstate %d myamt: %d theiramt: %d\n", q.State.StateIdx, q.State.MyAmt, q.Value-q.State.MyAmt)
	fmt.Printf("\tsig: %x\n", sig)

	worked := pSig.Verify(hash, theirPubKey)
	if !worked {
		return fmt.Errorf("Invalid signature on chan %d state %d",
			q.Idx(), q.State.StateIdx)
	}

	// copy signature, overwriting old signature.
	q.State.sig = sig

	return nil
}
