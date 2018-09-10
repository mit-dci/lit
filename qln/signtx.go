package qln

import (
	"bytes"
	"fmt"
	"log"

	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/txscript"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/sig64"
	"github.com/mit-dci/lit/wire"
)

// SignBreak signs YOUR tx, which you already have a sig for
func (nd *LitNode) SignBreakTx(q *Qchan) (*wire.MsgTx, error) {
	// TODO: we probably have to do something with the HTLCs here

	tx, _, _, err := q.BuildStateTxs(true)
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
	priv, err := nd.SubWallet[q.Coin()].GetPriv(q.KeyGen)
	if err != nil {
		return nil, err
	}
	// generate sig.
	mySig, err := txscript.RawTxInWitnessSignature(
		tx, hCache, 0, q.Value, pre, txscript.SigHashAll, priv)

	theirSig := sig64.SigDecompress(q.State.sig)
	// put the sighash all byte on the end of their signature
	theirSig = append(theirSig, byte(txscript.SigHashAll))

	log.Printf("made mysig: %x theirsig: %x\n", mySig, theirSig)
	// add sigs to the witness stack
	if swap {
		tx.TxIn[0].Witness = SpendMultiSigWitStack(pre, theirSig, mySig)
	} else {
		tx.TxIn[0].Witness = SpendMultiSigWitStack(pre, mySig, theirSig)
	}

	// save channel state as closed
	// Removed - this is already done in the calling function - and is killing
	// the ability to just print the TX.
	// q.CloseData.Closed = true
	// q.CloseData.CloseTxid = tx.TxHash()
	// err = nd.SaveQchanUtxoData(q)
	// if err != nil {
	//	return nil, err
	// }

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
	priv, err := nd.SubWallet[q.Coin()].GetPriv(q.KeyGen)
	if err != nil {
		return sig, err
	}
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
func (nd *LitNode) SignState(q *Qchan) ([64]byte, [][64]byte, error) {
	var sig [64]byte

	// make sure channel exists, and wallet is present on node
	if q == nil {
		return sig, nil, fmt.Errorf("SignState nil channel")
	}
	_, ok := nd.SubWallet[q.Coin()]
	if !ok {
		return sig, nil, fmt.Errorf("SignState no wallet for cointype %d", q.Coin())
	}
	// build transaction for next state
	commitmentTx, spendHTLCTxs, HTLCTxOuts, err := q.BuildStateTxs(false) // their tx, as I'm signing
	if err != nil {
		return sig, nil, err
	}

	log.Printf("Signing state with Elk [%x] NextElk [%x] N2Elk [%x]\n", q.State.ElkPoint, q.State.NextElkPoint, q.State.N2ElkPoint)

	// make hash cache for this tx
	hCache := txscript.NewTxSigHashes(commitmentTx)

	// generate script preimage (ignore key order)
	pre, _, err := lnutil.FundTxScript(q.MyPub, q.TheirPub)
	if err != nil {
		return sig, nil, err
	}

	// get private signing key
	priv, err := nd.SubWallet[q.Coin()].GetPriv(q.KeyGen)
	if err != nil {
		return sig, nil, err
	}

	// generate sig.
	bigSig, err := txscript.RawTxInWitnessSignature(
		commitmentTx, hCache, 0, q.Value, pre, txscript.SigHashAll, priv)
	// truncate sig (last byte is sighash type, always sighashAll)
	bigSig = bigSig[:len(bigSig)-1]

	sig, err = sig64.SigCompress(bigSig)
	if err != nil {
		return sig, nil, err
	}

	fmt.Printf("____ sig creation for channel (%d,%d):\n", q.Peer(), q.Idx())
	fmt.Printf("\tinput %s\n", commitmentTx.TxIn[0].PreviousOutPoint.String())
	for i, txout := range commitmentTx.TxOut {
		fmt.Printf("\toutput %d: %x %d\n", i, txout.PkScript, txout.Value)
	}

	log.Printf("\tstate %d myamt: %d theiramt: %d\n", q.State.StateIdx, q.State.MyAmt, q.Value-q.State.MyAmt)

	// Generate signatures for HTLC-success/failure transactions
	spendHTLCSigs := map[int][64]byte{}

	curElk, err := q.ElkSnd.AtIndex(q.State.StateIdx)
	if err != nil {
		return sig, nil, err
	}
	elkScalar := lnutil.ElkScalar(curElk)

	ep := lnutil.ElkPointFromHash(curElk)

	log.Printf("Using elkpoint %x to sign HTLC txs", ep)

	for idx, h := range HTLCTxOuts {
		// Find out which vout this HTLC is in the commitment tx since BIP69
		// potentially reordered them
		var where uint32
		for i, o := range commitmentTx.TxOut {
			if bytes.Compare(o.PkScript, h.PkScript) == 0 {
				where = uint32(i)
				break
			}
		}

		var HTLCPrivBase *btcec.PrivateKey
		if idx == len(q.State.HTLCs) {
			HTLCPrivBase, err = nd.SubWallet[q.Coin()].GetPriv(q.State.InProgHTLC.KeyGen)
		} else if idx == len(q.State.HTLCs)+1 {
			HTLCPrivBase, err = nd.SubWallet[q.Coin()].GetPriv(q.State.CollidingHTLC.KeyGen)
		} else {
			HTLCPrivBase, err = nd.SubWallet[q.Coin()].GetPriv(q.State.HTLCs[idx].KeyGen)
		}

		if err != nil {
			return sig, nil, err
		}

		HTLCPriv := lnutil.CombinePrivKeyWithBytes(HTLCPrivBase, elkScalar[:])

		// Find the tx we need to sign. (this would all be much easier if we
		// didn't use BIP69)
		var spendTx *wire.MsgTx
		var which int
		for i, t := range spendHTLCTxs {
			if t.TxIn[0].PreviousOutPoint.Index == where {
				spendTx = t
				which = i
				break
			}
		}

		hc := txscript.NewTxSigHashes(spendTx)
		var HTLCScript []byte

		if idx == len(q.State.HTLCs) {
			HTLCScript, err = q.GenHTLCScript(*q.State.InProgHTLC, false)
		} else if idx == len(q.State.HTLCs)+1 {
			HTLCScript, err = q.GenHTLCScript(*q.State.CollidingHTLC, false)
		} else {
			HTLCScript, err = q.GenHTLCScript(q.State.HTLCs[idx], false)
		}
		if err != nil {
			return sig, nil, err
		}

		HTLCparsed, err := txscript.ParseScript(HTLCScript)
		if err != nil {
			return sig, nil, err
		}

		spendHTLCHash := txscript.CalcWitnessSignatureHash(
			HTLCparsed, hc, txscript.SigHashAll, spendTx, 0, h.Value)

		log.Printf("Signing HTLC hash: %x, with pubkey: %x", spendHTLCHash, HTLCPriv.PubKey().SerializeCompressed())

		mySig, err := HTLCPriv.Sign(spendHTLCHash)
		if err != nil {
			return sig, nil, err
		}

		HTLCSig := mySig.Serialize()
		s, err := sig64.SigCompress(HTLCSig)
		if err != nil {
			return sig, nil, err
		}

		spendHTLCSigs[which] = s
	}

	// Get the sigs in the same order as the HTLCs in the tx
	var spendHTLCSigsArr [][64]byte
	for i := 0; i < len(spendHTLCSigs)+2; i++ {
		if s, ok := spendHTLCSigs[i]; ok {
			spendHTLCSigsArr = append(spendHTLCSigsArr, s)
		}
	}

	return sig, spendHTLCSigsArr, err
}

// VerifySig verifies their signature for your next state.
// it also saves the sig if it's good.
// do bool, error or just error?  Bad sig is an error I guess.
// for verifying signature, always use theirHAKDpub, so generate & populate within
// this function.
func (q *Qchan) VerifySigs(sig [64]byte, HTLCSigs [][64]byte) error {

	bigSig := sig64.SigDecompress(sig)
	// my tx when I'm verifying.
	commitmentTx, spendHTLCTxs, HTLCTxOuts, err := q.BuildStateTxs(true)
	if err != nil {
		return err
	}

	log.Printf("Verifying signatures with Elk [%x] NextElk [%x] N2Elk [%x]\n", q.State.ElkPoint, q.State.NextElkPoint, q.State.N2ElkPoint)

	// generate fund output script preimage (ignore key order)
	pre, _, err := lnutil.FundTxScript(q.MyPub, q.TheirPub)
	if err != nil {
		return err
	}

	hCache := txscript.NewTxSigHashes(commitmentTx)

	parsed, err := txscript.ParseScript(pre)
	if err != nil {
		return err
	}
	// always sighash all
	hash := txscript.CalcWitnessSignatureHash(
		parsed, hCache, txscript.SigHashAll, commitmentTx, 0, q.Value)

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
	fmt.Printf("\tinput %s\n", commitmentTx.TxIn[0].PreviousOutPoint.String())
	for i, txout := range commitmentTx.TxOut {
		fmt.Printf("\toutput %d: %x %d\n", i, txout.PkScript, txout.Value)
	}
	log.Printf("\tstate %d myamt: %d theiramt: %d\n", q.State.StateIdx, q.State.MyAmt, q.Value-q.State.MyAmt)
	log.Printf("\tsig: %x\n", sig)

	worked := pSig.Verify(hash, theirPubKey)
	if !worked {
		return fmt.Errorf("Invalid signature on chan %d state %d",
			q.Idx(), q.State.StateIdx)
	}

	// Verify HTLC-success/failure signatures

	if len(HTLCSigs) != len(spendHTLCTxs) {
		return fmt.Errorf("Wrong number of signatures provided for HTLCs in channel. Got %d expected %d.",
			len(HTLCSigs), len(spendHTLCTxs))
	}

	// Map HTLC index to signature index
	sigIndex := map[uint32]uint32{}

	log.Printf("Using elkpoint %x to verify HTLC txs", q.State.NextElkPoint)

	for idx, h := range HTLCTxOuts {
		// Find out which vout this HTLC is in the commitment tx since BIP69
		// potentially reordered them
		var where uint32
		for i, o := range commitmentTx.TxOut {
			if bytes.Compare(o.PkScript, h.PkScript) == 0 {
				where = uint32(i)
				break
			}
		}

		// Find the tx we need to verify. (this would all be much easier if we
		// didn't use BIP69)
		var spendTx *wire.MsgTx
		var which int
		for i, t := range spendHTLCTxs {
			if t.TxIn[0].PreviousOutPoint.Index == where {
				spendTx = t
				which = i
				sigIndex[uint32(idx)] = uint32(which)
				break
			}
		}

		hc := txscript.NewTxSigHashes(spendTx)
		var HTLCScript []byte
		if idx == len(q.State.HTLCs) {
			HTLCScript, err = q.GenHTLCScript(*q.State.InProgHTLC, true)
		} else if idx == len(q.State.HTLCs)+1 {
			HTLCScript, err = q.GenHTLCScript(*q.State.CollidingHTLC, true)
		} else {
			HTLCScript, err = q.GenHTLCScript(q.State.HTLCs[idx], true)
		}
		if err != nil {
			return err
		}

		HTLCparsed, err := txscript.ParseScript(HTLCScript)
		if err != nil {
			return err
		}
		// always sighash all
		spendHTLCHash := txscript.CalcWitnessSignatureHash(
			HTLCparsed, hc, txscript.SigHashAll, spendTx, 0, h.Value)

		// sig is pre-truncated; last byte for sighashtype is always sighashAll
		HTLCSig, err := btcec.ParseDERSignature(sig64.SigDecompress(HTLCSigs[which]), btcec.S256())
		if err != nil {
			return err
		}

		var theirHTLCPub [33]byte
		if idx == len(q.State.HTLCs) {
			theirHTLCPub = lnutil.CombinePubs(q.State.InProgHTLC.TheirHTLCBase, q.State.NextElkPoint)
		} else if idx == len(q.State.HTLCs)+1 {
			theirHTLCPub = lnutil.CombinePubs(q.State.CollidingHTLC.TheirHTLCBase, q.State.NextElkPoint)
		} else {
			theirHTLCPub = lnutil.CombinePubs(q.State.HTLCs[idx].TheirHTLCBase, q.State.NextElkPoint)
		}

		theirHTLCPubKey, err := btcec.ParsePubKey(theirHTLCPub[:], btcec.S256())
		if err != nil {
			return err
		}

		log.Printf("Verifying HTLC hash: %x, with pubkey: %x", spendHTLCHash, theirHTLCPub)

		sigValid := HTLCSig.Verify(spendHTLCHash, theirHTLCPubKey)
		if !sigValid {
			return fmt.Errorf("Invalid signature HTLC on chan %d state %d HTLC %d",
				q.Idx(), q.State.StateIdx, idx)
		}
	}

	// copy signature, overwriting old signature.
	q.State.sig = sig

	// copy HTLC-success/failure signatures
	for i, s := range sigIndex {
		if int(i) == len(q.State.HTLCs) {
			q.State.InProgHTLC.Sig = HTLCSigs[s]
		} else if int(i) == len(q.State.HTLCs)+1 {
			q.State.CollidingHTLC.Sig = HTLCSigs[s]
		} else {
			q.State.HTLCs[i].Sig = HTLCSigs[s]
		}
	}

	return nil
}
