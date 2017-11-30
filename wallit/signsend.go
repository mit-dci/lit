package wallit

import (
	"bytes"
	"fmt"
	"log"
	"sort"

	"github.com/adiabat/btcd/chaincfg/chainhash"
	"github.com/adiabat/btcd/txscript"
	"github.com/adiabat/btcd/wire"
	"github.com/adiabat/btcutil/txsort"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

// Build a tx, kindof like with SendCoins, but don't sign or broadcast.
// Segwit inputs only.  Freeze the utxos used so the tx can be signed and broadcast
// later.  Use only segwit utxos.  Return the txid, and indexes of where the txouts
// in the argument slice ended up in the final tx.
// Bunch of redundancy with SendMany, maybe move that to a shared function...
//NOTE this does not support multiple txouts with identical pkscripts in one tx.
// The code would be trivial; it's not supported on purpose.  Use unique pkscripts.

func (w *Wallit) SetRbf(rbf bool) {
	w.Rbf = rbf
	return
}

func (w *Wallit) MaybeSendRbf(txos []*wire.TxOut, ow bool) (*wire.MsgTx, error) {
	var err error
	var totalSend int64

	var outputByteSize int64
	// check for negative...?
	for _, txo := range txos {
		totalSend += txo.Value
		outputByteSize += 8 + int64(len(txo.PkScript))
	}
	var overshoot [6]int64
	var utxos []*portxo.PorTxo
	// overshoots vary for each fee value
	for i := 1; i < 6; i++ {
		utxos, overshoot[i], err =
			//w.PickUtxos(totalSend, outputByteSize, feePerByte, ow)
			w.PickUtxos(totalSend, outputByteSize, int64(20*i), ow)
		if err != nil {
			return nil, err
		}
	}
	// utxos configured to 100
	// overshoot values are stored
	log.Println("Utxos: ", utxos)
	log.Println("Overshoot:", overshoot)
	// Till here, nothing changes for all the Rbf guys
	padding := uint32(0)
	feePerByte := 20

	for loopVar := 1; loopVar <= 5; loopVar += 1 {
		// paste stuff Here
		log.Println("Setting fee per byte:", feePerByte)
		// log.Println("FreezeSet: ", w.FreezeSet)
		dustCutoff := int64(20000) // below this amount, just give to miners

		// make an initial txo copy so we can find where the outputs end up in final tx
		initTxos := make([]*wire.TxOut, len(txos))

		// change output (if needed)
		var changeOut *wire.TxOut

		ops := make([]*wire.OutPoint, len(txos))
		copy(initTxos, txos)

		// get inputs for this tx.  Only segwit if needed

		log.Printf("MaybeSend has overshoot %d, %d inputs\n", overshoot[loopVar], len(utxos))

		// changeOutSize is the extra vsize that a change output would add
		changeOutFee := 30 * feePerByte
		// add a change output if we have enough extra to do so
		if overshoot[loopVar] > dustCutoff+changeOutFee {
			changeOut, err = w.NewChangeOut(overshoot[loopVar] - changeOutFee)
			if err != nil {
				return nil, err
			}
		}

		if changeOut != nil {
			txos = append(txos, changeOut)
		}

		// make the tx (BuildDontSign stuff)
		tx := wire.NewMsgTx()
		// always make version 2 txs
		tx.Version = 2
		tx.LockTime = padding + uint32(w.CurrentHeight())
		// add all the txouts, direct from the argument slice
		for _, txo := range txos {
			if txo == nil || txo.PkScript == nil || txo.Value == 0 {
				return nil, fmt.Errorf("BuildAndSign arg invalid txo")
			}
			tx.AddTxOut(txo)
		}
		// add all the txins, first refenecing the prev outPoints
		for i, u := range utxos {
			tx.AddTxIn(wire.NewTxIn(&u.Op, nil, nil))
			// set sequence field if it's in the portxo
			if w.Rbf == true {
				u.Seq = 2
			} else {
				u.Seq = 1
			}
			tx.TxIn[i].Sequence = 4294967295 - u.Seq
		}
		// sort txouts in place before signing.  txins are already sorted from above
		txsort.InPlaceSort(tx)
		temp := tx.TxHash()
		// figure out where outputs ended up after adding the change output and sorting
		for i, initTxo := range initTxos {
			for j, finalTxo := range tx.TxOut {
				// If pkscripts match, this is where it ended up.
				// if you're sending different amounts to the same address, this
				// might not work!  Don't re-use addresses!
				if bytes.Equal(initTxo.PkScript, finalTxo.PkScript) {
					ops[i] = wire.NewOutPoint(&temp, uint32(j))
				}
			}
		}

		if changeOut != nil {
			txos = append(txos, changeOut)
		}

		// call a aprt of buildandsign over here
		if len(utxos) == 0 || len(txos) == 0 {
			return nil, fmt.Errorf("BuildAndSign args no utxos or txos")
		}
		// sort input utxos first.
		sort.Sort(portxo.TxoSliceByBip69(utxos))

		log.Println("tx:", tx.Version, tx.TxIn, tx.TxOut, tx.LockTime)
		padding += 10
		feePerByte += 20
	}
	return nil, nil
}

func (w *Wallit) MaybeSend(txos []*wire.TxOut, ow bool) ([]*wire.OutPoint, error) {
	var err error
	var totalSend int64
	dustCutoff := int64(20000) // below this amount, just give to miners
	// TODO: set dustCutoff dynamically acc to formula
	// dust = 3*(input + output)bytes * minrelaytxfee(sat/byte)
	// minrelaytxfee = 1 sat/byte acc to std. implementations (p2pkh=546sat)

	feePerByte := w.FeeRate

	// make an initial txo copy so we can find where the outputs end up in final tx

	initTxos := make([]*wire.TxOut, len(txos))

	// change output (if needed)
	var changeOut *wire.TxOut

	finalOutPoints := make([]*wire.OutPoint, len(txos))
	copy(initTxos, txos)

	var outputByteSize int64
	// check for negative...?
	for _, txo := range txos {
		totalSend += txo.Value
		outputByteSize += 8 + int64(len(txo.PkScript))
	}

	// start access to utxos
	w.FreezeMutex.Lock()
	defer w.FreezeMutex.Unlock()

	// get inputs for this tx.  Only segwit if needed
	utxos, overshoot, err :=
		w.PickUtxos(totalSend, outputByteSize, feePerByte, ow)
	if err != nil {
		return nil, err
	}

	log.Printf("MaybeSend has overshoot %d, %d inputs\n", overshoot, len(utxos))

	// changeOutSize is the extra vsize that a change output would add
	changeOutFee := 30 * feePerByte

	// add a change output if we have enough extra to do so
	if overshoot > dustCutoff+changeOutFee {
		changeOut, err = w.NewChangeOut(overshoot - changeOutFee)
		if err != nil {
			return nil, err
		}
	}

	// build frozen tx for later broadcast
	fTx := new(FrozenTx)
	fTx.Ins = utxos
	fTx.Outs = txos
	fTx.ChangeOut = changeOut

	if changeOut != nil {
		txos = append(txos, changeOut)
	}

	// BuildDontSign gets the txid.  Also sorts txin, txout slices in place
	tx, err := w.BuildDontSign(utxos, txos)
	if err != nil {
		return nil, err
	}

	// after building, store the locktime and txid
	fTx.Nlock = tx.LockTime
	fTx.Txid = tx.TxHash()

	for _, utxo := range utxos {
		w.FreezeSet[utxo.Op] = fTx
	}

	// figure out where outputs ended up after adding the change output and sorting
	for i, initTxo := range initTxos {
		for j, finalTxo := range tx.TxOut {
			// If pkscripts match, this is where it ended up.
			// if you're sending different amounts to the same address, this
			// might not work!  Don't re-use addresses!
			if bytes.Equal(initTxo.PkScript, finalTxo.PkScript) {
				finalOutPoints[i] = wire.NewOutPoint(&fTx.Txid, uint32(j))
			}
		}
	}

	return finalOutPoints, nil
}

// Sign and broadcast a tx previously built with MaybeSend.  This clears the freeze
// on the utxos but they're not utxos anymore anyway.
func (w *Wallit) ReallySend(txid *chainhash.Hash) error {
	log.Printf("Reallysend %s\n", txid.String())
	// start frozen set access
	w.FreezeMutex.Lock()
	defer w.FreezeMutex.Unlock()
	// get the transaction
	frozenTx, err := w.FindFreezeTx(txid) // doesn't return it if not at correct lock time
	if err != nil {
		return err
	}
	// delete inputs from frozen set (they're gone anyway, but just to clean it up)
	// I'm not going to delete those tx's with immature locktimes
	for _, txin := range frozenTx.Ins {
		log.Printf("\t remove %s from frozen outpoints\n", txin.Op.String())
		// rejected occurs over here
		delete(w.FreezeSet, txin.Op)
	}

	allOuts := frozenTx.Outs

	if frozenTx.ChangeOut != nil {
		allOuts = append(frozenTx.Outs, frozenTx.ChangeOut)
	}
	tx, err := w.BuildAndSign(frozenTx.Ins, allOuts, frozenTx.Nlock)
	if err != nil {
		return err
	}

	return w.NewOutgoingTx(tx)
}

// Cancel the hold on a tx previously built with MaybeSend.  Clears freeze on
// utxos so they can be used somewhere else.
// This function does nothing atm. Will be removed in upcoming commits
func (w *Wallit) NahDontSend(txid *chainhash.Hash) error {
	log.Printf("Nahdontsend %s\n", txid.String())
	// start frozen set access
	w.FreezeMutex.Lock()
	defer w.FreezeMutex.Unlock()
	// get the transaction
	frozenTx, err := w.FindFreezeTx(txid)
	if err != nil {
		return err
	}
	// go through all its inputs, and remove those outpoints from the frozen set
	for _, txin := range frozenTx.Ins {
		log.Printf("\t remove %s from frozen outpoints\n", txin.Op.String())
		delete(w.FreezeSet, txin.Op)
	}
	return nil
}

// FindFreezeTx looks through the frozen map to find a tx.  Error if it can't find it
func (w *Wallit) FindFreezeTx(txid *chainhash.Hash) (*FrozenTx, error) {
	// Here, what I do is pacakgge the tx and send it off without checking whether the
	// tx is confirmed at a specific height or not. I need to see if the transaction ahs been spent
	// Two ways to do this
	// 1. Query bitcoind
	// 2. Check the mem pool for that tx. If it is still present, means that the tx didn't go through.
	// So what I should do now is re-send th transasction with a higher fee but same input tx.
	// the later block will invalidate what I had earlier.
	for op := range w.FreezeSet {
		frozenTxid := w.FreezeSet[op].Txid
		if frozenTxid.IsEqual(txid) && (w.FreezeSet[op].Nlock == uint32(w.CurrentHeight())) {
			return w.FreezeSet[op], nil
		} else {
			// if its not set to the current lock time, don't return anything
			log.Println("Frozen tx will be unlocked at height", w.FreezeSet[op].Nlock)
			// weirdly enough, this nlock doesn't match the value passed to it.
			return w.FreezeSet[op], nil
		}
	}
	return nil, fmt.Errorf("couldn't find %s in frozen set", txid.String())
}

// GrabAll makes first-party justice txs.
func (w *Wallit) GrabAll() error {
	// no args, look through all utxos
	utxos, err := w.GetAllUtxos()
	if err != nil {
		return err
	}

	// currently grabs only confirmed txs.
	nothin := true
	for _, u := range utxos {
		if u.Seq == 1 && u.Height > 0 { // grabbable
			log.Printf("found %s to grab!\n", u.String())
			adr160, err := w.NewAdr160()
			if err != nil {
				return err
			}

			outScript := lnutil.DirectWPKHScriptFromPKH(adr160)

			tx, err := w.SendOne(*u, outScript)
			if err != nil {
				return err
			}
			err = w.NewOutgoingTx(tx)
			if err != nil {
				return err
			}
			nothin = false
		}
	}
	if nothin {
		log.Printf("Nothing to grab\n")
	}
	return nil
}

// Directly send out a tx.  For things that plug in to the uspv wallet.
func (w *Wallit) DirectSendTx(tx *wire.MsgTx) error {
	// don't ingest, just push out
	return w.Hook.PushTx(tx)
}

// NewOutgoingTx runs a tx though the db first, then sends it out to the network.
func (w *Wallit) NewOutgoingTx(tx *wire.MsgTx) error {
	_, err := w.Ingest(tx, 0) // our own tx; don't keep track of false positives
	if err != nil {
		return err
	}
	return w.Hook.PushTx(tx)
}

// PickUtxos Picks Utxos for spending.  Tell it how much money you want.
// It returns a tx-sortable utxoslice, and the overshoot amount.  Also errors.
// if "ow" is true, only gives witness utxos (for channel funding)
// The overshoot amount is *after* fees, so can be used directly for a
// change output.
func (w *Wallit) PickUtxos(
	amtWanted, outputByteSize, feePerByte int64,
	ow bool) (portxo.TxoSliceByBip69, int64, error) {

	curHeight, err := w.GetDBSyncHeight()
	if err != nil {
		return nil, 0, err
	}

	var allUtxos portxo.TxoSliceByAmt
	allUtxos, err = w.GetAllUtxos()
	if err != nil {
		return nil, 0, err
	}
	// remove frozen utxos from allUtxo slice.  Iterate backwards / trailing delete
	for i := len(allUtxos) - 1; i >= 0; i-- {
		_, frozen := w.FreezeSet[allUtxos[i].Op]
		if frozen {
			// faster than append, and we're sorting a few lines later anyway
			allUtxos[i] = allUtxos[len(allUtxos)-1] // redundant if at last index
			allUtxos = allUtxos[:len(allUtxos)-1]   // trim last element
		}
	}

	// start with utxos sorted by value and pop off utxos which are greater
	// than the send amount... as long as the next 2 are greater.
	// simple / straightforward coin selection optimization, which tends to make
	// 2 in 2 out

	// smallest and unconfirmed last (because it's reversed)
	sort.Sort(sort.Reverse(allUtxos))

	// guessing that txs won't be more than 10K here...
	maxFeeGuess := feePerByte * 10000

	// first pass of removing candidate utxos; if the next one is bigger than
	// we need, remove the top one.
	for len(allUtxos) > 1 &&
		allUtxos[1].Value > amtWanted+maxFeeGuess &&
		allUtxos[1].Height > 100 {
		allUtxos = allUtxos[1:]
	}

	// if we've got 2 or more confirmed utxos, and the next one is
	// more than enough, pop off the first one.
	// Note that there are probably all sorts of edge cases where this will
	// result in not being able to send money when you should be able to.
	// Thus the handwavey "maxFeeGuess"

	for len(allUtxos) > 2 &&
		allUtxos[2].Height > 100 && // since sorted, don't need to check [1]
		allUtxos[1].Mature(curHeight) &&
		allUtxos[2].Mature(curHeight) &&
		allUtxos[1].Value+allUtxos[2].Value > amtWanted+maxFeeGuess {
		log.Printf("remaining utxo list, in order:\n")
		for _, u := range allUtxos {
			log.Printf("\t h: %d amt: %d\n", u.Height, u.Value)
		}
		allUtxos = allUtxos[1:]
	}

	// coin selection is super complex, and we can definitely do a lot better
	// here!
	// TODO: anyone who wants to: implement more advanced coin selection algo

	// rSlice is the return slice of the utxos which are going into the tx
	var rSlice portxo.TxoSliceByBip69
	// add utxos until we've had enough
	remaining := amtWanted // remaining is how much is needed on input side
	sum := int64(0)
	for _, j := range allUtxos {
		sum += j.Value
		//log.Println("the value of this utxo is: ", j.Value)
	}
	for _, utxo := range allUtxos {
		// skip unconfirmed.  Or de-prioritize? Some option for this...
		//		if utxo.AtHeight == 0 {
		//			continue
		//		}
		if remaining > sum {
			log.Println("This sum of utxos is insufficient to make a tx. Please try again")
			return nil, 0, fmt.Errorf("wanted %d but only %d available.",
				remaining, sum)
		}
		log.Println("The value of the utxo that I get is: ", utxo.Value)
		if !utxo.Mature(curHeight) {
			continue // skip immature or unconfirmed time-locked sh outputs
		}
		if ow && utxo.Mode&portxo.FlagTxoWitness == 0 {
			continue // skip non-witness
		}
		// why are 0-value outputs a thing..?
		if utxo.Value < 1 {
			continue
		}
		// yeah, lets add this utxo!
		rSlice = append(rSlice, utxo)
		sum -= utxo.Value
		remaining -= utxo.Value
		// if remaining is positive, don't bother checking fee yet.
		// if remaining is negative, calculate needed fee
		if remaining <= 0 {
			fee := EstFee(rSlice, outputByteSize, feePerByte)
			// subtract fee from returned overshoot.
			// (remaining is negative here)
			remaining += fee

			// done adding utxos if remaining below negative est fee
			if remaining < -fee {
				break
			}
		}
	}

	if remaining > 0 {
		return nil, 0, fmt.Errorf("wanted %d but %d available.",
			amtWanted, amtWanted-remaining)
		// guy returns negative stuff sometimes
	}

	sort.Sort(rSlice) // send sorted.  This is probably redundant?
	return rSlice, -remaining, nil
}

// SendOne is for the sweep function, and doesn't do change.
// Probably can get rid of this for real txs.
func (w *Wallit) SendOne(u portxo.PorTxo, outScript []byte) (*wire.MsgTx, error) {

	w.FreezeMutex.Lock()
	defer w.FreezeMutex.Unlock()
	_, frozen := w.FreezeSet[u.Op]
	if frozen {
		return nil, fmt.Errorf("%s is frozen, can't spend", u.Op.String())
	}

	curHeight, err := w.GetDBSyncHeight()
	if err != nil {
		return nil, err
	}

	if u.Seq > 1 &&
		(u.Height < 100 || u.Height+int32(u.Seq) > curHeight) {
		// skip immature or unconfirmed time-locked sh outputs
		return nil, fmt.Errorf("Can't spend, immature")
	}
	// fixed fee
	fee := w.FeeRate * 200

	sendAmt := u.Value - fee
	// make user specified txout and add to tx
	txout := wire.NewTxOut(sendAmt, outScript)

	return w.BuildAndSign(
		[]*portxo.PorTxo{&u}, []*wire.TxOut{txout}, uint32(w.CurrentHeight()))
}

// Builds tx from inputs and outputs, returns tx.  Sorts.  Doesn't sign.
func (w *Wallit) BuildDontSign(
	utxos []*portxo.PorTxo, txos []*wire.TxOut) (*wire.MsgTx, error) {

	log.Println("RBF is set to", w.Rbf)
	// make the tx
	tx := wire.NewMsgTx()
	// set version 2, for op_csv
	tx.Version = 2
	// set the time, the way core does.
	tx.LockTime = uint32(w.CurrentHeight())

	// add all the txouts
	for _, txo := range txos {
		tx.AddTxOut(txo)
	}
	// add all the txins
	for i, u := range utxos {
		tx.AddTxIn(wire.NewTxIn(&u.Op, nil, nil))
		// set sequence field if it's in the portxo
		if w.Rbf == true {
			u.Seq = 2 // setting seq to 0xffffffff - 2 following electrum
		} else {
			u.Seq = 1
		}
		tx.TxIn[i].Sequence = 4294967295 - u.Seq // is there a nicer way?
	}
	// sort in place before signing
	txsort.InPlaceSort(tx)
	return tx, nil
}

func (w *Wallit) BuildSignRbf(txos []*wire.TxOut, ow bool) error {
	tx, err := w.MaybeSendRbf(txos, false)
	if err != nil {
		log.Println("Errored while sending rbf transaction. Sorry")
		return err
	}
	log.Println("tx2:", tx)
	return nil
}

// Build and sign builds a tx from a slice of utxos and txOuts.
// It then signs all the inputs and returns the tx.  Should
// pretty much always work for any inputs.
func (w *Wallit) BuildAndSign(
	utxos []*portxo.PorTxo, txos []*wire.TxOut, nlt uint32) (*wire.MsgTx, error) {
	var err error

	if len(utxos) == 0 || len(txos) == 0 {
		return nil, fmt.Errorf("BuildAndSign args no utxos or txos")
	}
	// sort input utxos first.
	sort.Sort(portxo.TxoSliceByBip69(utxos))

	// make the tx
	tx := wire.NewMsgTx()

	// always make version 2 txs
	tx.Version = 2
	tx.LockTime = nlt
	// add all the txouts, direct from the argument slice
	for _, txo := range txos {
		if txo == nil || txo.PkScript == nil || txo.Value == 0 {
			return nil, fmt.Errorf("BuildAndSign arg invalid txo")
		}
		tx.AddTxOut(txo)
	}
	// add all the txins, first refenecing the prev outPoints
	for i, u := range utxos {
		tx.AddTxIn(wire.NewTxIn(&u.Op, nil, nil))
		// set sequence field if it's in the portxo
		if w.Rbf == true {
			u.Seq = 2
		} else {
			u.Seq = 1
		}
		tx.TxIn[i].Sequence = 4294967295 - u.Seq
	}
	// sort txouts in place before signing.  txins are already sorted from above
	txsort.InPlaceSort(tx)

	// Signing stuff starts here!
	// generate tx-wide hashCache for segwit stuff
	// might not be needed (non-witness) but make it anyway
	hCache := txscript.NewTxSigHashes(tx)
	// make the stashes for signatures / witnesses
	sigStash := make([][]byte, len(utxos))
	witStash := make([][][]byte, len(utxos))

	for i, _ := range tx.TxIn {
		// get key
		priv := w.PathPrivkey(utxos[i].KeyGen)
		log.Printf("signing with privkey pub %x\n", priv.PubKey().SerializeCompressed())

		if priv == nil {
			return nil, fmt.Errorf("SendCoins: nil privkey")
		}

		// sign into stash.  3 possibilities:  legacy PKH, WPKH, WSH
		if utxos[i].Mode == portxo.TxoP2PKHComp { // legacy PKH
			sigStash[i], err = txscript.SignatureScript(tx, i,
				utxos[i].PkScript, txscript.SigHashAll, priv, true)
			if err != nil {
				return nil, err
			}
		}
		if utxos[i].Mode == portxo.TxoP2WPKHComp { // witness PKH
			witStash[i], err = txscript.WitnessScript(tx, hCache, i,
				utxos[i].Value, utxos[i].PkScript, txscript.SigHashAll, priv, true)
			if err != nil {
				return nil, err
			}
			//}
		}
		if utxos[i].Mode == portxo.TxoP2WSHComp { // witness script hash
			sig, err := txscript.RawTxInWitnessSignature(tx, hCache, i,
				utxos[i].Value, utxos[i].PkScript, txscript.SigHashAll, priv)
			if err != nil {
				return nil, err
			}
			// witness stack has the signature, items, then the previous full script
			witStash[i] = make([][]byte, 2+len(utxos[i].PreSigStack))

			// sig comes first (pushed to stack last)
			witStash[i][0] = sig

			// after stack comes PostSigStack items
			for j, element := range utxos[i].PreSigStack {
				witStash[i][j+1] = element
			}

			// last stack item is the pkscript
			witStash[i][len(witStash[i])-1] = utxos[i].PkScript
		}

	}
	// swap sigs into sigScripts in txins
	for i, txin := range tx.TxIn {
		if sigStash[i] != nil {
			txin.SignatureScript = sigStash[i]
		}
		if witStash[i] != nil {
			txin.Witness = witStash[i]
			txin.SignatureScript = nil
		}
	}

	log.Printf("tx: %s", TxToString(tx))
	return tx, nil
}

// EstFee gives a fee estimate based on a input / output set and a sat/Byte target.
// It guesses the final tx size based on:
// Txouts: 8 bytes + pkscript length
// Total guess on the p2wsh one, see if that's accurate
func EstFee(txins []*portxo.PorTxo, outputByteSize, spB int64) int64 {
	size := int64(40)      // around 40 bytes for a change output and nlock time
	size += outputByteSize // add the output size

	// iterate through txins, guessing size based on mode
	for _, txin := range txins {
		if txin == nil { // silently ignore nil txins; somebody else's problem
			continue
		}
		size += txin.EstSize()
	}

	log.Printf("%d spB, est vsize %d, fee %d\n", spB, size, size*spB)
	return size * spB
}
