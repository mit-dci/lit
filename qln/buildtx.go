package qln

import (
	"bytes"
	"fmt"
	"log"

	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/lnutil"

	"github.com/mit-dci/lit/btcutil/txsort"
	"github.com/mit-dci/lit/wire"
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
		//		log.Printf("sequence byte %x, locktime byte %x\n",
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

// BuildStateTxs constructs and returns a state commitment tx and a list of HTLC
// success/failure txs.  As simple as I can make it.
// This func just makes the tx with data from State in ram, and HAKD key arg
func (q *Qchan) BuildStateTxs(mine bool) (*wire.MsgTx, []*wire.MsgTx, []*wire.TxOut, error) {
	if q == nil {
		return nil, nil, nil, fmt.Errorf("BuildStateTx: nil chan")
	}
	// sanity checks
	s := q.State // use it a lot, make shorthand variable
	if s == nil {
		return nil, nil, nil, fmt.Errorf("channel (%d,%d) has no state", q.KeyGen.Step[3], q.KeyGen.Step[4])
	}

	var fancyAmt, pkhAmt, theirAmt int64 // output amounts

	revPub, timePub, pkhPub, err := q.GetKeysFromState(mine)
	if err != nil {
		return nil, nil, nil, err
	}

	var revPKH [20]byte
	revPKHSlice := btcutil.Hash160(revPub[:])
	copy(revPKH[:], revPKHSlice[:20])

	fee := s.Fee // fixed fee for now

	value := q.Value

	if s.InProgHTLC != nil {
		value -= s.InProgHTLC.Amt
	}

	if s.CollidingHTLC != nil {
		value -= s.CollidingHTLC.Amt
	}

	for _, h := range s.HTLCs {
		if !h.Cleared && !h.Clearing {
			value -= h.Amt
		}
	}

	theirAmt = value - s.MyAmt

	log.Printf("Value: %d, MyAmt: %d, TheirAmt: %d", value, s.MyAmt, theirAmt)

	// the PKH clear refund also has elkrem points added to mask the PKH.
	// this changes the txouts at each state to blind sorcerer better.
	if mine { // build MY tx (to verify) (unless breaking)
		// nonzero amts means build the output
		if theirAmt > 0 {
			pkhAmt = theirAmt - fee
		}
		if s.MyAmt > 0 {
			fancyAmt = s.MyAmt - fee
		}
	} else { // build THEIR tx (to sign)
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
		return nil, nil, nil, fmt.Errorf("SH amt %d too low", fancyAmt)
	}
	if pkhAmt != 0 && pkhAmt < consts.MinOutput {
		return nil, nil, nil, fmt.Errorf("PKH amt %d too low", pkhAmt)
	}

	// now that everything is chosen, build fancy script and pkh script
	fancyScript := lnutil.CommitScript(revPub, timePub, q.Delay)
	pkhScript := lnutil.DirectWPKHScript(pkhPub) // p2wpkh-ify

	log.Printf("> made SH script, state %d\n", s.StateIdx)
	log.Printf("\t revPub %x timeout pub %x \n", revPub, timePub)
	log.Printf("\t script %x ", fancyScript)

	fancyScript = lnutil.P2WSHify(fancyScript) // p2wsh-ify

	log.Printf("\t scripthash %x\n", fancyScript)

	// create txouts by assigning amounts
	outFancy := wire.NewTxOut(fancyAmt, fancyScript)
	outPKH := wire.NewTxOut(pkhAmt, pkhScript)

	fmt.Printf("\tcombined refund %x, pkh %x, amt %d\n", pkhPub, outPKH.PkScript, pkhAmt)

	var HTLCTxOuts []*wire.TxOut

	// Generate new HTLC signatures
	for _, h := range s.HTLCs {
		if !h.Clearing && !h.Cleared {
			HTLCOut, err := q.GenHTLCOut(h, mine)
			if err != nil {
				return nil, nil, nil, err
			}
			HTLCTxOuts = append(HTLCTxOuts, HTLCOut)
		}
	}

	// There's an HTLC in progress
	if s.InProgHTLC != nil {
		HTLCOut, err := q.GenHTLCOut(*s.InProgHTLC, mine)
		if err != nil {
			return nil, nil, nil, err
		}
		HTLCTxOuts = append(HTLCTxOuts, HTLCOut)
	}

	// There's an colliding HTLC in progress
	if s.CollidingHTLC != nil {
		HTLCOut, err := q.GenHTLCOut(*s.CollidingHTLC, mine)
		if err != nil {
			return nil, nil, nil, err
		}
		HTLCTxOuts = append(HTLCTxOuts, HTLCOut)
	}

	// make a new tx
	tx := wire.NewMsgTx()
	// add txouts
	if fancyAmt != 0 {
		tx.AddTxOut(outFancy)
	}
	if pkhAmt != 0 {
		tx.AddTxOut(outPKH)
	}

	// Add HTLC outputs
	for _, out := range HTLCTxOuts {
		tx.AddTxOut(out)
	}

	if len(tx.TxOut) < 1 {
		return nil, nil, nil, fmt.Errorf("No outputs, all below minOutput")
	}

	// add unsigned txin
	tx.AddTxIn(wire.NewTxIn(&q.Op, nil, nil))
	// set index hints

	// state 0 and 1 can't use mask?  Think they can now.
	SetStateIdxBits(tx, s.StateIdx, q.GetChanHint(mine))

	// sort outputs
	txsort.InPlaceSort(tx)

	txHash := tx.TxHash()

	HTLCSpends := map[int]*wire.MsgTx{}

	for j, h := range HTLCTxOuts {
		amt := h.Value - fee
		if amt < consts.MinOutput {
			return nil, nil, nil, fmt.Errorf("HTLC amt %d too low (fee is %d)", amt, fee)
		}

		// But now they're sorted how do I know which outpoint to spend?
		// We can iterate over our HTLC list, then compare pkScripts to find
		// the right one
		// Which index is this HTLC output in the tx?
		var idx int
		for i, out := range tx.TxOut {
			if bytes.Compare(out.PkScript, h.PkScript) == 0 {
				idx = i
				break
			}
		}

		spendHTLCScript := lnutil.CommitScript(revPub, timePub, q.Delay)

		HTLCSpend := wire.NewMsgTx()

		HTLCOp := wire.NewOutPoint(&txHash, uint32(idx))

		in := wire.NewTxIn(HTLCOp, nil, nil)
		in.Sequence = 0

		HTLCSpend.AddTxIn(in)
		HTLCSpend.AddTxOut(wire.NewTxOut(amt, lnutil.P2WSHify(spendHTLCScript)))

		HTLCSpend.Version = 2

		/*
			!incoming & mine: my TX that they sign (HTLC-timeout)
			!incoming & !mine: their TX that I sign (HTLC-success)
			incoming & mine: my TX that they sign (HTLC-success)
			incoming & !mine: their TX that I sign (HTLC-timeout)
		*/

		var success bool
		var lt uint32

		if j == len(s.HTLCs) {
			success = s.InProgHTLC.Incoming == mine
			lt = s.InProgHTLC.Locktime
		} else if j == len(s.HTLCs)+1 {
			success = s.CollidingHTLC.Incoming == mine
			lt = s.CollidingHTLC.Locktime
		} else {
			success = s.HTLCs[j].Incoming == mine
			lt = s.HTLCs[j].Locktime
		}

		if success {
			// HTLC-success
			HTLCSpend.LockTime = 0
		} else {
			// HTLC-failure
			HTLCSpend.LockTime = lt
		}

		HTLCSpends[idx] = HTLCSpend
	}

	var HTLCSpendsArr []*wire.MsgTx

	for i := 0; i < len(HTLCSpends)+2; i++ {
		if s, ok := HTLCSpends[i]; ok {
			HTLCSpendsArr = append(HTLCSpendsArr, s)
		}
	}

	return tx, HTLCSpendsArr, HTLCTxOuts, nil
}

func (q *Qchan) GenHTLCScriptWithElkPointsAndRevPub(h HTLC, mine bool, theirElkPoint, myElkPoint, revPub [33]byte) ([]byte, error) {
	var remotePub, localPub [33]byte

	revPKHSlice := btcutil.Hash160(revPub[:])
	var revPKH [20]byte
	copy(revPKH[:], revPKHSlice[:20])

	if mine { // Generating OUR tx that WE save
		remotePub = lnutil.CombinePubs(h.TheirHTLCBase, theirElkPoint)
		localPub = lnutil.CombinePubs(h.MyHTLCBase, myElkPoint)
	} else { // Generating THEIR tx that THEY save
		remotePub = lnutil.CombinePubs(h.MyHTLCBase, myElkPoint)
		localPub = lnutil.CombinePubs(h.TheirHTLCBase, theirElkPoint)
	}

	var HTLCScript []byte

	/*
		incoming && mine = Receive
		incoming && !mine = Offer
		!incoming && mine = Offer
		!incoming && !mine = Receive
	*/
	if h.Incoming != mine {
		HTLCScript = lnutil.OfferHTLCScript(revPKH,
			remotePub, h.RHash, localPub)
	} else {
		HTLCScript = lnutil.ReceiveHTLCScript(revPKH,
			remotePub, h.RHash, localPub, h.Locktime)
	}

	log.Printf("HTLC %d, script: %x, myBase: %x, theirBase: %x, Incoming: %t, Amt: %d, RHash: %x",
		h.Idx, HTLCScript, h.MyHTLCBase, h.TheirHTLCBase, h.Incoming, h.Amt, h.RHash)

	return HTLCScript, nil

}

func (q *Qchan) GenHTLCScript(h HTLC, mine bool) ([]byte, error) {

	revPub, _, _, err := q.GetKeysFromState(mine)
	if err != nil {
		return nil, err
	}

	curElk, err := q.ElkPoint(false, q.State.StateIdx)
	if err != nil {
		return nil, err
	}
	return q.GenHTLCScriptWithElkPointsAndRevPub(h, mine, q.State.ElkPoint, curElk, revPub)
}

func (q *Qchan) GenHTLCOutWithElkPointsAndRevPub(h HTLC, mine bool, theirElkPoint, myElkPoint, revPub [33]byte) (*wire.TxOut, error) {
	HTLCScript, err := q.GenHTLCScriptWithElkPointsAndRevPub(h, mine, theirElkPoint, myElkPoint, revPub)
	if err != nil {
		return nil, err
	}

	witScript := lnutil.P2WSHify(HTLCScript)

	HTLCOut := wire.NewTxOut(h.Amt, witScript)

	return HTLCOut, nil
}

func (q *Qchan) GenHTLCOut(h HTLC, mine bool) (*wire.TxOut, error) {
	revPub, _, _, err := q.GetKeysFromState(mine)
	if err != nil {
		return nil, err
	}

	curElk, err := q.ElkPoint(false, q.State.StateIdx)
	if err != nil {
		return nil, err
	}

	return q.GenHTLCOutWithElkPointsAndRevPub(h, mine, q.State.ElkPoint, curElk, revPub)
}

// GetKeysFromState will inspect the channel state and return the revPub, timePub and pkhPub based on
// whether we're building our own or the remote transaction.
func (q *Qchan) GetKeysFromState(mine bool) (revPub, timePub, pkhPub [33]byte, err error) {

	// the PKH clear refund also has elkrem points added to mask the PKH.
	// this changes the txouts at each state to blind sorcerer better.
	if mine { // build MY tx (to verify) (unless breaking)
		var curElk [33]byte
		// My tx that I store.  They get funds unencumbered. SH is mine eventually
		// SH pubkeys are base points combined with the elk point we give them
		// Create latest elkrem point (the one I create)
		curElk, err = q.ElkPoint(false, q.State.StateIdx)
		if err != nil {
			return
		}
		revPub = lnutil.CombinePubs(q.TheirHAKDBase, curElk)
		timePub = lnutil.AddPubsEZ(q.MyHAKDBase, curElk)

		pkhPub = q.TheirRefundPub

	} else { // build THEIR tx (to sign)
		// Their tx that they store.  I get funds PKH.  SH is theirs eventually.
		log.Printf("using elkpoint %x\n", q.State.ElkPoint)
		// SH pubkeys are our base points plus the received elk point
		revPub = lnutil.CombinePubs(q.MyHAKDBase, q.State.ElkPoint)
		timePub = lnutil.AddPubsEZ(q.TheirHAKDBase, q.State.ElkPoint)
		// PKH output
		pkhPub = q.MyRefundPub
	}

	return
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
