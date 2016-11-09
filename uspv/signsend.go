package uspv

import (
	"bytes"
	"fmt"
	"log"
	"sort"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/txsort"
	"github.com/mit-dci/lit/portxo"
)

// Build a tx, kindof like with SendCoins, but don't sign or broadcast.
// Segwit inputs only.  Freeze the utxos used so the tx can be signed and broadcast
// later.  Use only segwit utxos.  Return the txid, and indexes of where the txouts
// in the argument slice ended up in the final tx.
// Bunch of redundancy with SendMany, maybe move that to a shared function...
//NOTE this does not support multiple txouts with identical pkscripts in one tx.
// The code would be trivial; it's not supported on purpose.  Use unique pkscripts.
func (s *SPVCon) MaybeSend(txos []*wire.TxOut) ([]*wire.OutPoint, error) {
	var err error
	var totalSend int64
	dustCutoff := int64(20000) // below this amount, just give to miners
	satPerByte := int64(80)    // satoshis per byte fee; have as arg later

	// make an initial txo copy so we can find where the outputs end up in final tx

	initTxos := make([]*wire.TxOut, len(txos))

	// change output (if needed)
	var changeOut *wire.TxOut

	finalOutPoints := make([]*wire.OutPoint, len(txos))
	copy(initTxos, txos)

	// check for negative...?
	for _, txo := range txos {
		totalSend += txo.Value
	}

	// start access to utxos
	s.TS.FreezeMutex.Lock()
	defer s.TS.FreezeMutex.Unlock()
	// get inputs for this tx.  Only segwit
	// This might not be enough for the fee if the inputs line up right...
	utxos, overshoot, err := s.TS.PickUtxos(totalSend, true)
	if err != nil {
		return nil, err
	}

	// estimate fee with outputs, see if change should be truncated
	fee := EstFee(utxos, txos, satPerByte)

	// add a change output if we have enough extra
	if overshoot-fee > dustCutoff {
		changeOut, err = s.TS.NewChangeOut(overshoot - fee)
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
	txid, err := s.TS.BuildDontSign(utxos, txos)
	if err != nil {
		return nil, err
	}

	fTx.Txid = txid

	for _, utxo := range utxos {
		s.TS.FreezeSet[utxo.Op] = fTx
	}

	// figure out where outputs ended up after adding the change output and sorting
	for i, initTxo := range initTxos {
		for j, finalTxo := range txos {
			// If pkscripts match, this is where it ended up.
			// if you're sending different amounts to the same address, this
			// might not work!  Don't re-use addresses!
			if bytes.Equal(initTxo.PkScript, finalTxo.PkScript) {
				finalOutPoints[i] = wire.NewOutPoint(&txid, uint32(j))
			}
		}
	}

	return finalOutPoints, nil
}

// Sign and broadcast a tx previously built with MaybeSend.  This clears the freeze
// on the utxos but they're not utxos anymore anyway.
func (s *SPVCon) ReallySend(txid *chainhash.Hash) error {
	fmt.Printf("Reallysend %s\n", txid.String())
	// start frozen set access
	s.TS.FreezeMutex.Lock()
	defer s.TS.FreezeMutex.Unlock()
	// get the transaction
	frozenTx, err := s.TS.FindFreezeTx(txid)
	if err != nil {
		return err
	}
	// delete inputs from frozen set (they're gone anyway, but just to clean it up)
	for _, txin := range frozenTx.Ins {
		fmt.Printf("\t remove %s from frozen outpoints\n", txin.Op.String())
		delete(s.TS.FreezeSet, txin.Op)
	}

	allOuts := append(frozenTx.Outs, frozenTx.ChangeOut)

	tx, err := s.TS.BuildAndSign(frozenTx.Ins, allOuts)
	if err != nil {
		return err
	}

	// All non-change outputs of the frozenTx are registered as watch-outpoints
	for i, txout := range tx.TxOut {
		if !bytes.Equal(frozenTx.ChangeOut.PkScript, txout.PkScript) {
			// not the change output, so register it as a watched OP
			var op wire.OutPoint
			op.Hash = tx.TxHash()
			op.Index = uint32(i)
			err = s.TS.RegisterWatchOP(op)
			if err != nil {
				return err
			}
		}
	}

	return s.NewOutgoingTx(tx)
}

// Cancel the hold on a tx previously built with MaybeSend.  Clears freeze on
// utxos so they can be used somewhere else.
func (s *SPVCon) NahDontSend(txid *chainhash.Hash) error {
	fmt.Printf("Nahdontsend %s\n", txid.String())
	// start frozen set access
	s.TS.FreezeMutex.Lock()
	defer s.TS.FreezeMutex.Unlock()
	// get the transaction
	frozenTx, err := s.TS.FindFreezeTx(txid)
	if err != nil {
		return err
	}
	// go through all its inputs, and remove those outpoints from the frozen set
	for _, txin := range frozenTx.Ins {
		fmt.Printf("\t remove %s from frozen outpoints\n", txin.Op.String())
		delete(s.TS.FreezeSet, txin.Op)
	}
	return nil
}

func (s *SPVCon) WatchThis(op wire.OutPoint) error {
	err := s.TS.RegisterWatchOP(op)
	if err != nil {
		return err
	}
	// make new filter
	filt, err := s.TS.GimmeFilter()
	if err != nil {
		return err
	}
	// send filter
	s.Refilter(filt)
	return nil
}

// FindFreezeTx looks through the frozen map to find a tx.  Error if it can't find it
func (ts *TxStore) FindFreezeTx(txid *chainhash.Hash) (*FrozenTx, error) {
	for op := range ts.FreezeSet {
		frozenTxid := ts.FreezeSet[op].Txid
		if frozenTxid.IsEqual(txid) {
			return ts.FreezeSet[op], nil
		}
	}
	return nil, fmt.Errorf("couldn't find %s in frozen set", txid.String())
}

// Rebroadcast sends an inv message of all the unconfirmed txs the db is
// aware of.  This is called after every sync.  Only txids so hopefully not
// too annoying for nodes.
func (s *SPVCon) Rebroadcast() {
	// get all unconfirmed txs
	invMsg, err := s.TS.GetPendingInv()
	if err != nil {
		log.Printf("Rebroadcast GetPendingInv error: %s", err.Error())
		return
	}
	if len(invMsg.InvList) == 0 { // nothing to broadcast, so don't
		return
	}
	s.outMsgQueue <- invMsg
	return
}

func (s *SPVCon) GrabAll() error {
	// no args, look through all utxos
	utxos, err := s.TS.GetAllUtxos()
	if err != nil {
		return err
	}

	// currently grabs only confirmed txs.
	nothin := true
	for _, u := range utxos {
		if u.Seq == 1 && u.Height > 0 { // grabbable
			fmt.Printf("found %s to grab!\n", u.String())
			adr160, err := s.TS.NewAdr160()
			if err != nil {
				return err
			}
			nAdr, err := btcutil.NewAddressWitnessPubKeyHash(
				adr160, s.TS.Param)
			tx, err := s.TS.SendOne(*u, nAdr)
			if err != nil {
				return err
			}
			err = s.NewOutgoingTx(tx)
			if err != nil {
				return err
			}
			nothin = false
		}
	}
	if nothin {
		fmt.Printf("Nothing to grab\n")
	}
	return nil
}

// NewOutgoingTx runs a tx though the db first, then sends it out to the network.
func (s *SPVCon) NewOutgoingTx(tx *wire.MsgTx) error {
	txid := tx.TxHash()
	// assign height of zero for txs we create
	err := s.OKTxid(&txid, 0)
	if err != nil {
		return err
	}
	_, err = s.TS.Ingest(tx, 0) // our own tx; don't keep track of false positives
	if err != nil {
		return err
	}
	// make an inv message instead of a tx message to be polite
	iv1 := wire.NewInvVect(wire.InvTypeWitnessTx, &txid)
	invMsg := wire.NewMsgInv()
	err = invMsg.AddInvVect(iv1)
	if err != nil {
		return err
	}
	s.outMsgQueue <- invMsg
	return nil
}

// PickUtxos Picks Utxos for spending.  Tell it how much money you want.
// It returns a tx-sortable utxoslice, and the overshoot amount.  Also errors.
// if "ow" is true, only gives witness utxos (for channel funding)
func (ts *TxStore) PickUtxos(
	amtWanted int64, ow bool) (portxo.TxoSliceByBip69, int64, error) {
	satPerByte := int64(80) // satoshis per byte fee; have as arg later
	curHeight, err := ts.GetDBSyncHeight()
	if err != nil {
		return nil, 0, err
	}

	var allUtxos portxo.TxoSliceByAmt
	allUtxos, err = ts.GetAllUtxos()
	if err != nil {
		return nil, 0, err
	}

	// remove frozen utxos from allUtxo slice.  Iterate backwards / trailing delete
	for i := len(allUtxos) - 1; i >= 0; i-- {
		_, frozen := ts.FreezeSet[allUtxos[i].Op]
		if frozen {
			// faster than append, and we're sorting a few lines later anyway
			allUtxos[i] = allUtxos[len(allUtxos)-1] // redundant if at last index
			allUtxos = allUtxos[:len(allUtxos)-1]   // trim last element
		}
	}

	// start with utxos sorted by value.
	// smallest and unconfirmed last (because it's reversed)
	sort.Sort(sort.Reverse(allUtxos))

	var rSlice portxo.TxoSliceByBip69
	// add utxos until we've had enough
	nokori := amtWanted // nokori is how much is needed on input side
	for _, utxo := range allUtxos {
		// skip unconfirmed.  Or de-prioritize? Some option for this...
		//		if utxo.AtHeight == 0 {
		//			continue
		//		}
		if utxo.Seq > 1 &&
			(utxo.Height < 100 || utxo.Height+int32(utxo.Seq) > curHeight) {
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
		nokori -= utxo.Value
		// if nokori is positive, don't bother checking fee yet.
		if nokori < 0 {
			var byteSize int64
			for _, txo := range rSlice {
				if txo.Mode&portxo.FlagTxoWitness != 0 {
					byteSize += 70 // vsize of wit inputs is ~68ish
				} else {
					byteSize += 130 // vsize of non-wit input is ~130
				}
			}
			fee := byteSize * satPerByte
			if nokori < -fee { // done adding utxos: nokori below negative est fee
				break
			}
		}
	}
	if nokori > 0 {
		return nil, 0, fmt.Errorf("wanted %d but %d available.",
			amtWanted, amtWanted-nokori)
	}

	sort.Sort(rSlice) // send sorted.  This is probably redundant?
	return rSlice, -nokori, nil
}

// sendDrop is broken, try to fix it for better spamming.

// SendDrop sends 2 chained transactions; one to a 2drop script, and then
// one spending that to an address.
// Note that this is completely insecure for any purpose, and
// all it does is waste space.  Kindof useless.
// Returns the 2nd, large tx's txid.
// Probably doesn't work with time-locked.  Doesn't really matter.
/*
func (ts *TxStore) SendDrop(
	u portxo.PorTxo, adr btcutil.Address) (*wire.MsgTx, *wire.MsgTx, error) {
	var err error
	// fixed fee
	fee := int64(5000)

	sendAmt := u.Value - fee
	tx := wire.NewMsgTx() // make new tx

	// add single dropdrop output
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_2DROP)
	builder.AddOp(txscript.OP_2DROP)
	builder.AddOp(txscript.OP_2DROP)
	builder.AddOp(txscript.OP_2DROP)
	builder.AddOp(txscript.OP_2DROP)
	builder.AddOp(txscript.OP_2DROP)
	builder.AddOp(txscript.OP_2DROP)
	builder.AddOp(txscript.OP_2DROP)
	builder.AddOp(txscript.OP_1)
	outpre, _ := builder.Script()

	txout := wire.NewTxOut(sendAmt, lnutil.P2WSHify(outpre))
	tx.AddTxOut(txout)

	// build input
	var prevPKs []byte
	if u.Mode&portxo.FlagTxoWitness != 0 {
		wa, err := btcutil.NewAddressWitnessPubKeyHash(
			ts.Adrs[u.KeyGen.Step[4]].PkhAdr.ScriptAddress(), ts.Param)
		prevPKs, err = txscript.PayToAddrScript(wa)
		if err != nil {
			return nil, nil, err
		}
	} else { // otherwise generate directly
		prevPKs, err = txscript.PayToAddrScript(
			ts.Adrs[u.KeyGen.Step[4]].PkhAdr)
		if err != nil {
			return nil, nil, err
		}
	}
	tx.AddTxIn(wire.NewTxIn(&u.Op, prevPKs, nil))

	var sig []byte
	var wit [][]byte
	hCache := txscript.NewTxSigHashes(tx)

	priv := ts.PathPrivkey(u.KeyGen)
	if priv == nil {
		return nil, nil, fmt.Errorf("SendDrop: nil privkey")
	}

	// This is where witness based sighash types need to happen
	// sign into stash
	if u.Mode&portxo.FlagTxoWitness != 0 {
		wit, err = txscript.WitnessScript(
			tx, hCache, 0, u.Value, tx.TxIn[0].SignatureScript,
			txscript.SigHashAll, priv, true)
		if err != nil {
			return nil, nil, err
		}
	} else {
		sig, err = txscript.SignatureScript(
			tx, 0, tx.TxIn[0].SignatureScript,
			txscript.SigHashAll, priv, true)
		if err != nil {
			return nil, nil, err
		}
	}

	// swap sigs into sigScripts in txins
	if sig != nil {
		tx.TxIn[0].SignatureScript = sig
	}
	if wit != nil {
		tx.TxIn[0].Witness = wit
		tx.TxIn[0].SignatureScript = nil
	}

	tx1id := tx.TxSha()
	sendAmt2 := sendAmt - fee
	tx2 := wire.NewMsgTx() // make new tx

	// now build a NEW tx spending that one!
	// add single output
	outAdrScript, err := txscript.PayToAddrScript(adr)
	if err != nil {
		return nil, nil, err
	}

	txout2 := wire.NewTxOut(sendAmt2, outAdrScript)
	tx2.AddTxOut(txout2)

	dropIn := wire.NewTxIn(wire.NewOutPoint(&tx1id, 0), nil, nil)
	dropIn.Witness = make([][]byte, 17)

	for i, _ := range dropIn.Witness {
		dropIn.Witness[i] = make([]byte, 512)
		_, err := rand.Read(dropIn.Witness[i])
		if err != nil {
			return nil, nil, err
		}
	}
	dropIn.Witness[16] = outpre
	tx2.AddTxIn(dropIn)

	return tx, tx2, nil
}
*/

// SendOne is for the sweep function, and doesn't do change.
// Probably can get rid of this for real txs.
func (ts *TxStore) SendOne(u portxo.PorTxo, adr btcutil.Address) (*wire.MsgTx, error) {

	ts.FreezeMutex.Lock()
	defer ts.FreezeMutex.Unlock()
	_, frozen := ts.FreezeSet[u.Op]
	if frozen {
		return nil, fmt.Errorf("%s is frozen, can't spend", u.Op.String())
	}

	curHeight, err := ts.GetDBSyncHeight()
	if err != nil {
		return nil, err
	}

	if u.Seq > 1 &&
		(u.Height < 100 || u.Height+int32(u.Seq) > curHeight) {
		// skip immature or unconfirmed time-locked sh outputs
		return nil, fmt.Errorf("Can't spend, immature")
	}
	// fixed fee
	fee := int64(5000)

	sendAmt := u.Value - fee

	// add single output
	outAdrScript, err := txscript.PayToAddrScript(adr)
	if err != nil {
		return nil, err
	}
	// make user specified txout and add to tx
	txout := wire.NewTxOut(sendAmt, outAdrScript)

	return ts.BuildAndSign([]*portxo.PorTxo{&u}, []*wire.TxOut{txout})
}

// Builds tx from inputs and outputs, returns txid.  Sorts.  Doesn't sign.
func (ts *TxStore) BuildDontSign(
	utxos []*portxo.PorTxo, txos []*wire.TxOut) (chainhash.Hash, error) {

	// make the tx
	tx := wire.NewMsgTx()
	tx.Version = 2
	// add all the txouts
	for _, txo := range txos {
		tx.AddTxOut(txo)
	}
	// add all the txins
	for i, u := range utxos {
		tx.AddTxIn(wire.NewTxIn(&u.Op, nil, nil))
		// set sequence field if it's in the portxo
		if u.Seq > 1 {
			tx.TxIn[i].Sequence = u.Seq
		}
	}
	// sort in place before signing
	txsort.InPlaceSort(tx)
	return tx.TxHash(), nil
}

// Build and sign builds a tx from a slice of utxos and txOuts.
// It then signs all the inputs and returns the tx.  Should
// pretty much always work for any inputs.
func (ts *TxStore) BuildAndSign(
	utxos []*portxo.PorTxo, txos []*wire.TxOut) (*wire.MsgTx, error) {
	var err error

	// sort utxos first.  I think this works.
	sort.Sort(portxo.TxoSliceByBip69(utxos))

	// make the tx
	tx := wire.NewMsgTx()

	// always make version 2 txs
	tx.Version = 2

	// add all the txouts, direct from the argument slice
	for _, txo := range txos {
		tx.AddTxOut(txo)
	}
	// add all the txins, first refenecing the prev outPoints
	for i, u := range utxos {
		tx.AddTxIn(wire.NewTxIn(&u.Op, nil, nil))
		// set sequence field if it's in the portxo
		if u.Seq > 1 {
			tx.TxIn[i].Sequence = u.Seq
		}
	}
	// sort txouts in place before signing.  txins are already sorted from above
	txsort.InPlaceSort(tx)

	// generate tx-wide hashCache for segwit stuff
	// might not be needed (non-witness) but make it anyway
	hCache := txscript.NewTxSigHashes(tx)
	// make the stashes for signatures / witnesses
	sigStash := make([][]byte, len(utxos))
	witStash := make([][][]byte, len(utxos))

	for i, _ := range tx.TxIn {
		// get key
		priv := ts.PathPrivkey(utxos[i].KeyGen)

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

	fmt.Printf("tx: %s", TxToString(tx))
	return tx, nil
}

// SendCoins sends coins.
func (ts *TxStore) SendCoins(
	adrs []btcutil.Address, sendAmts []int64) (*wire.MsgTx, error) {

	if len(adrs) != len(sendAmts) {
		return nil, fmt.Errorf(
			"%d addresses and %d amounts", len(adrs), len(sendAmts))
	}
	var err error
	var txos []*wire.TxOut
	var totalSend int64
	dustCutoff := int64(20000) // below this amount, just give to miners
	satPerByte := int64(80)    // satoshis per byte fee; have as arg later

	for _, amt := range sendAmts {
		totalSend += amt
	}

	// add non-change (arg) outputs
	for i, adr := range adrs {
		// make address script 76a914...88ac or 0014...
		outAdrScript, err := txscript.PayToAddrScript(adr)
		if err != nil {
			return nil, err
		}
		// make user specified txout and add to tx
		txout := wire.NewTxOut(sendAmts[i], outAdrScript)
		txos = append(txos, txout)
	}

	// get inputs for this tx
	// This might not be enough for the fee if the inputs line up right...
	utxos, overshoot, err := ts.PickUtxos(totalSend, false)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Overshot by %d, can make change output\n", overshoot)

	// estimate fee with outputs, see if change should be truncated
	fee := EstFee(utxos, txos, satPerByte)

	// add a change output if we have enough extra
	if overshoot-fee > dustCutoff {
		changeOut, err := ts.NewChangeOut(overshoot - fee)
		if err != nil {
			return nil, err
		}
		txos = append(txos, changeOut)
	}

	return ts.BuildAndSign(utxos, txos)
}

// EstFee gives a fee estimate based on a input / output set and a sat/Byte target.
// It guesses the final tx size based on:
// Txouts: 8 bytes + pkscript length
// Txins by mode:
// P2 PKH is op,seq (40) + pub(33) + sig(71) = 144
// P2 WPKH is op,seq(40) + [(33+71 / 4) = 26] = 66
// P2 WSH is op,seq(40) + [75(script) + 71]/4 (36) = 76
// Total guess on the p2wsh one, see if that's accurate
func EstFee(txins []*portxo.PorTxo, txouts []*wire.TxOut, spB int64) int64 {
	size := int64(40) // around 40 bytes for a change output and nlock time
	// iterate through txins, guessing size based on mode
	for _, txin := range txins {
		switch txin.Mode {
		case portxo.TxoP2PKHComp: // non witness is about 150 bytes
			size += 144
		case portxo.TxoP2WPKHComp:
			size += 66
		case portxo.TxoP2WSHComp:
			size += 76
		default:
			size += 150 // huh?
		}
	}
	for _, txout := range txouts {
		size += 8 + int64(len(txout.PkScript))
	}
	fmt.Printf("%d spB, est vsize %d, fee %d\n", spB, size, size*spB)
	return size * spB
}
