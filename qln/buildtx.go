package qln

import (
	"fmt"

	"github.com/mit-dci/lit/lnutil"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/txsort"
)

// GetStateIdxFromTx returns the state index from a commitment transaction.
// No errors; returns 0 if there is no retrievable index.
// Takes the xor input X which is derived from the 0th elkrems.
func GetStateIdxFromTx(tx *wire.MsgTx, x uint64) uint64 {
	// no tx, so no index
	if tx == nil {
		return 0
	}
	// more than 1 input, so not a close tx
	if len(tx.TxIn) != 1 {
		return 0
	}
	// mask need two high bytes of 0s
	if x > 1<<48 {
		return 0
	}
	// check that indicating high bytes are correct.  If not, return 0
	if tx.TxIn[0].Sequence>>24 != 0xff || tx.LockTime>>24 != 0x21 {
		//		fmt.Printf("sequence byte %x, locktime byte %x\n",
		//			tx.TxIn[0].Sequence>>24, tx.LockTime>>24 != 0x21)
		return 0
	}
	// high 24 bits sequence, low 24 bits locktime
	seqBits := uint64(tx.TxIn[0].Sequence & 0x00ffffff)
	timeBits := uint64(tx.LockTime & 0x00ffffff)

	return (seqBits<<24 | timeBits) ^ x
}

// SetStateIdxBits modifies the tx in place, setting the sequence and locktime
// fields to indicate the given state index.
func SetStateIdxBits(tx *wire.MsgTx, idx, x uint64) error {
	if tx == nil {
		return fmt.Errorf("SetStateIdxBits: nil tx")
	}
	if len(tx.TxIn) != 1 {
		return fmt.Errorf("SetStateIdxBits: tx has %d inputs", len(tx.TxIn))
	}
	if idx >= 1<<48 {
		return fmt.Errorf(
			"SetStateIdxBits: index %d greater than max %d", idx, uint64(1<<48)-1)
	}

	idx = idx ^ x
	// high 24 bits sequence, low 24 bits locktime
	seqBits := uint32(idx >> 24)
	timeBits := uint32(idx & 0x00ffffff)

	tx.TxIn[0].Sequence = seqBits | seqMask
	tx.LockTime = timeBits | timeMask

	return nil
}

// SimpleCloseTx produces a close tx based on the current state.
// The PKH addresses are my refund base with their r-elkrem point, and
// their refund base with my r-elkrem point.  "Their" point means they have
// the point but not the scalar.
func (q *Qchan) SimpleCloseTx() (*wire.MsgTx, error) {
	// sanity checks
	if q == nil || q.State == nil {
		return nil, fmt.Errorf("SimpleCloseTx: nil chan / state")
	}
	fee := int64(5000) // fixed fee for now (on both sides)

	// make my output
	myScript := lnutil.DirectWPKHScript(q.MyRefundPub)
	myOutput := wire.NewTxOut(q.State.MyAmt-fee, myScript)
	// make their output
	theirScript := lnutil.DirectWPKHScript(q.TheirRefundPub)
	theirOutput := wire.NewTxOut((q.Value-q.State.MyAmt)-fee, theirScript)

	// make tx with these outputs
	tx := wire.NewMsgTx()
	tx.AddTxOut(myOutput)
	tx.AddTxOut(theirOutput)
	// add channel outpoint as txin
	tx.AddTxIn(wire.NewTxIn(&q.Op, nil, nil))
	// sort and return
	txsort.InPlaceSort(tx)
	return tx, nil
}

// BuildStateTx constructs and returns a state tx.  As simple as I can make it.
// This func just makes the tx with data from State in ram, and HAKD key arg
// Delta should always be 0 when making this tx.
// It decides whether to make THEIR tx or YOUR tx based on the HAKD pubkey given --
// if it's zero, then it makes their transaction (for signing onlu)
// If it's full, it makes your transaction (for verification in most cases,
// but also for signing when breaking the channel)
// Index is used to set nlocktime for state hints.
// fee and op_csv timeout are currently hardcoded, make those parameters later.
// also returns the script preimage for later spending.
func (q *Qchan) BuildStateTx(mine bool) (*wire.MsgTx, error) {
	if q == nil {
		return nil, fmt.Errorf("BuildStateTx: nil chan")
	}
	// sanity checks
	s := q.State // use it a lot, make shorthand variable
	if s == nil {
		return nil, fmt.Errorf("channel (%d,%d) has no state", q.KeyGen.Step[3], q.KeyGen.Step[4])
	}
	// if delta is non-zero, something is wrong.
	if s.Delta != 0 {
		return nil, fmt.Errorf(
			"BuildStateTx: delta is %d (expect 0)", s.Delta)
	}
	var fancyAmt, pkhAmt int64   // output amounts
	var revPub, timePub [33]byte // pubkeys
	var pkhPub [33]byte          // the simple output's pub key hash
	fee := int64(5000)           // fixed fee for now
	delay := uint16(5)           // fixed CSV delay for now
	// delay is super short for testing.

	// Both received and self-generated elkpoints are needed

	// Create latest elkrem point (the one I create)
	curElk, err := q.ElkPoint(false, q.State.StateIdx)
	if err != nil {
		return nil, err
	}

	// the PKH clear refund also has elkrem points added to mask the PKH.
	// this changes the txouts at each state to blind sorceror better.
	if mine { // build MY tx (to verify) (unless breaking)
		// My tx that I store.  They get funds unencumbered. SH is mine eventually
		// SH pubkeys are base points combined with the elk point we give them

		revPub = lnutil.CombinePubs(q.TheirHAKDBase, curElk)
		timePub = lnutil.AddPubsEZ(q.MyHAKDBase, curElk)

		pkhPub = q.TheirRefundPub
		pkhAmt = (q.Value - s.MyAmt) - fee
		fancyAmt = s.MyAmt - fee

		fmt.Printf("\t refund base %x, elkpointR %x\n", q.TheirRefundPub, s.ElkPoint)
	} else { // build THEIR tx (to sign)
		// Their tx that they store.  I get funds PKH.  SH is theirs eventually.

		// SH pubkeys are our base points plus the received elk point
		revPub = lnutil.CombinePubs(q.MyHAKDBase, s.ElkPoint)
		timePub = lnutil.AddPubsEZ(q.TheirHAKDBase, s.ElkPoint)

		fancyAmt = (q.Value - s.MyAmt) - fee

		// PKH output
		pkhPub = q.MyRefundPub
		pkhAmt = s.MyAmt - fee
		fmt.Printf("\trefund base %x, elkpoint %x\n", q.MyRefundPub, curElk)
	}

	// now that everything is chosen, build fancy script and pkh script
	fancyScript := lnutil.CommitScript(revPub, timePub, delay)
	pkhScript := lnutil.DirectWPKHScript(pkhPub) // p2wpkh-ify

	fmt.Printf("> made SH script, state %d\n", s.StateIdx)
	fmt.Printf("\t revPub %x timeout pub %x \n", revPub, timePub)
	fmt.Printf("\t script %x ", fancyScript)

	fancyScript = lnutil.P2WSHify(fancyScript) // p2wsh-ify

	fmt.Printf("\t scripthash %x\n", fancyScript)

	// create txouts by assigning amounts
	outFancy := wire.NewTxOut(fancyAmt, fancyScript)
	outPKH := wire.NewTxOut(pkhAmt, pkhScript)

	fmt.Printf("\tcombined refund %x, pkh %x\n", pkhPub, outPKH.PkScript)

	// make a new tx
	tx := wire.NewMsgTx()
	// add txouts
	tx.AddTxOut(outFancy)
	tx.AddTxOut(outPKH)
	// add unsigned txin
	tx.AddTxIn(wire.NewTxIn(&q.Op, nil, nil))
	// set index hints

	// state 0 and 1 can't use mask?  Think they can now.
	SetStateIdxBits(tx, s.StateIdx, q.GetChanHint(mine))

	// sort outputs
	txsort.InPlaceSort(tx)
	return tx, nil
}

// the scriptsig to put on a P2SH input.  Sigs need to be in order!
func SpendMultiSigWitStack(pre, sigA, sigB []byte) [][]byte {

	witStack := make([][]byte, 4)

	witStack[0] = nil // it's not an OP_0 !!!! argh!
	witStack[1] = sigA
	witStack[2] = sigB
	witStack[3] = pre

	return witStack
}
