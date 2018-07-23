package qln

import (
	"fmt"
	."github.com/mit-dci/lit/logs"

	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/lnutil"

	"github.com/mit-dci/lit/wire"
	"github.com/mit-dci/lit/btcutil/txsort"
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
		//		Log.Printf("sequence byte %x, locktime byte %x\n",
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

	fee := q.State.Fee // symmetric fee

	// make my output
	myScript := lnutil.DirectWPKHScript(q.MyRefundPub)
	myAmt := q.State.MyAmt - fee
	myOutput := wire.NewTxOut(myAmt, myScript)
	// make their output
	theirScript := lnutil.DirectWPKHScript(q.TheirRefundPub)
	theirAmt := (q.Value - q.State.MyAmt) - fee
	theirOutput := wire.NewTxOut(theirAmt, theirScript)

	// check output amounts (should never fail)
	if myAmt < consts.MinOutput {
		return nil, fmt.Errorf("SimpleCloseTx: my output amt %d too low", myAmt)
	}
	if theirAmt < consts.MinOutput {
		return nil, fmt.Errorf("SimpleCloseTx: their output amt %d too low", myAmt)
	}

	tx := wire.NewMsgTx()

	// make tx with these outputs
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
func (q *Qchan) BuildStateTx(mine bool) (*wire.MsgTx, error) {
	if q == nil {
		return nil, fmt.Errorf("BuildStateTx: nil chan")
	}
	// sanity checks
	s := q.State // use it a lot, make shorthand variable
	if s == nil {
		return nil, fmt.Errorf("channel (%d,%d) has no state", q.KeyGen.Step[3], q.KeyGen.Step[4])
	}

	var fancyAmt, pkhAmt, theirAmt int64 // output amounts
	var revPub, timePub [33]byte         // pubkeys
	var pkhPub [33]byte                  // the simple output's pub key hash

	fee := s.Fee // fixed fee for now

	theirAmt = q.Value - s.MyAmt

	// the PKH clear refund also has elkrem points added to mask the PKH.
	// this changes the txouts at each state to blind sorcerer better.
	if mine { // build MY tx (to verify) (unless breaking)
		// My tx that I store.  They get funds unencumbered. SH is mine eventually
		// SH pubkeys are base points combined with the elk point we give them
		// Create latest elkrem point (the one I create)
		curElk, err := q.ElkPoint(false, q.State.StateIdx)
		if err != nil {
			return nil, err
		}
		revPub = lnutil.CombinePubs(q.TheirHAKDBase, curElk)
		timePub = lnutil.AddPubsEZ(q.MyHAKDBase, curElk)

		pkhPub = q.TheirRefundPub

		// nonzero amts means build the output
		if theirAmt > 0 {
			pkhAmt = theirAmt - fee
		}
		if s.MyAmt > 0 {
			fancyAmt = s.MyAmt - fee
		}
	} else { // build THEIR tx (to sign)
		// Their tx that they store.  I get funds PKH.  SH is theirs eventually.
		Log.Infof("using elkpoint %x\n", s.ElkPoint)
		// SH pubkeys are our base points plus the received elk point
		revPub = lnutil.CombinePubs(q.MyHAKDBase, s.ElkPoint)
		timePub = lnutil.AddPubsEZ(q.TheirHAKDBase, s.ElkPoint)
		// PKH output
		pkhPub = q.MyRefundPub

		// nonzero amts means build the output
		if theirAmt > 0 {
			fancyAmt = theirAmt - fee
		}
		if s.MyAmt > 0 {
			pkhAmt = s.MyAmt - fee
		}
	}

	// check amounts.  Nonzero amounts below the minOutput is an error.
	// Shouldn't happen and means some checks in push/pull went wrong.
	if fancyAmt != 0 && fancyAmt < consts.MinOutput {
		return nil, fmt.Errorf("SH amt %d too low", fancyAmt)
	}
	if pkhAmt != 0 && pkhAmt < consts.MinOutput {
		return nil, fmt.Errorf("PKH amt %d too low", pkhAmt)
	}

	// now that everything is chosen, build fancy script and pkh script
	fancyScript := lnutil.CommitScript(revPub, timePub, q.Delay)
	pkhScript := lnutil.DirectWPKHScript(pkhPub) // p2wpkh-ify

	Log.Infof("> made SH script, state %d\n", s.StateIdx)
	Log.Infof("\t revPub %x timeout pub %x \n", revPub, timePub)
	Log.Infof("\t script %x ", fancyScript)

	fancyScript = lnutil.P2WSHify(fancyScript) // p2wsh-ify

	Log.Infof("\t scripthash %x\n", fancyScript)

	// create txouts by assigning amounts
	outFancy := wire.NewTxOut(fancyAmt, fancyScript)
	outPKH := wire.NewTxOut(pkhAmt, pkhScript)

	Log.Infof("\tcombined refund %x, pkh %x\n", pkhPub, outPKH.PkScript)

	// make a new tx
	tx := wire.NewMsgTx()
	// add txouts
	if fancyAmt != 0 {
		tx.AddTxOut(outFancy)
	}
	if pkhAmt != 0 {
		tx.AddTxOut(outPKH)
	}

	if len(tx.TxOut) < 1 {
		return nil, fmt.Errorf("No outputs, all below minOutput")
	}

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
