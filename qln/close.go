package qln

import (
	"bytes"
	"fmt"
	"log"

	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/btcutil/txscript"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/sig64"
	"github.com/mit-dci/lit/wire"
)

/* CloseChannel --- cooperative close
This is the simplified close which sends to the same outputs as a break tx,
just with no timeouts.

Users might want a more advanced close function which allows multiple outputs.
They can exchange txouts and sigs.  That could be "fancyClose", but this is
just close, so only a signature is sent by the initiator, and the receiver
doesn't reply, as the channel is closed.

*/

// CoopClose requests a cooperative close of the channel
func (nd *LitNode) CoopClose(q *Qchan) error {

	nd.RemoteMtx.Lock()
	_, ok := nd.RemoteCons[q.Peer()]
	nd.RemoteMtx.Unlock()
	if !ok {
		return fmt.Errorf("not connected to peer %d ", q.Peer())
	}

	if q.CloseData.Closed {
		return fmt.Errorf("can't close (%d,%d): already closed",
			q.KeyGen.Step[3]&0x7fffffff, q.KeyGen.Step[4]&0x7fffffff)
	}

	for _, h := range q.State.HTLCs {
		if !h.Cleared {
			return fmt.Errorf("can't close (%d,%d): there are uncleared HTLCs",
				q.KeyGen.Step[3]&0x7fffffff, q.KeyGen.Step[4]&0x7fffffff)
		}
	}

	tx, err := q.SimpleCloseTx()
	if err != nil {
		return err
	}

	sig, err := nd.SignSimpleClose(q, tx)
	if err != nil {
		return err
	}

	// Save something, just so the UI marks it as closed, and
	// we don't accept payments on this channel anymore.

	// save channel state as closed.  We know the txid... even though that
	// txid may not actually happen.
	q.CloseData.Closed = true
	q.CloseData.CloseTxid = tx.TxHash()
	err = nd.SaveQchanUtxoData(q)
	if err != nil {
		return err
	}

	var signature [64]byte
	copy(signature[:], sig[:])

	// Save something to db... TODO
	// Should save something, just so the UI marks it as closed, and
	// we don't accept payments on this channel anymore.

	outMsg := lnutil.NewCloseReqMsg(q.Peer(), q.Op, signature)

	nd.OmniOut <- outMsg
	return nil
}

// CloseReqHandler takes in a close request from a remote host, signs and
// responds with a close response.  Obviously later there will be some judgment
// over what to do, but for now it just signs whatever it's requested to.

func (nd *LitNode) CloseReqHandler(msg lnutil.CloseReqMsg) {
	opArr := lnutil.OutPointToBytes(msg.Outpoint)

	// get channel
	q, err := nd.GetQchan(opArr)
	if err != nil {
		log.Printf("CloseReqHandler GetQchan err %s", err.Error())
		return
	}

	if nd.SubWallet[q.Coin()] == nil {
		log.Printf("Not connected to coin type %d\n", q.Coin())
	}

	for _, h := range q.State.HTLCs {
		if !h.Cleared {
			log.Printf("can't close (%d,%d): there are uncleared HTLCs",
				q.KeyGen.Step[3]&0x7fffffff, q.KeyGen.Step[4]&0x7fffffff)
			return
		}
	}

	// verify their sig?  should do that before signing our side just to be safe
	// TODO -- yeah we need to verify their sig

	// build close tx
	tx, err := q.SimpleCloseTx()
	if err != nil {
		log.Printf("CloseReqHandler SimpleCloseTx err %s", err.Error())
		return
	}

	hCache := txscript.NewTxSigHashes(tx)

	pre, _, err := lnutil.FundTxScript(q.MyPub, q.TheirPub)
	if err != nil {
		log.Printf("CloseReqHandler Sig err %s", err.Error())
		return
	}

	parsed, err := txscript.ParseScript(pre)
	if err != nil {
		log.Printf("CloseReqHandler Sig err %s", err.Error())
		return
	}
	// always sighash all
	hash := txscript.CalcWitnessSignatureHash(
		parsed, hCache, txscript.SigHashAll, tx, 0, q.Value)

	theirBigSig := sig64.SigDecompress(msg.Signature)

	// sig is pre-truncated; last byte for sighashtype is always sighashAll
	pSig, err := btcec.ParseDERSignature(theirBigSig, btcec.S256())
	if err != nil {
		log.Printf("CloseReqHandler Sig err %s", err.Error())
		return
	}
	theirPubKey, err := btcec.ParsePubKey(q.TheirPub[:], btcec.S256())
	if err != nil {
		log.Printf("CloseReqHandler Sig err %s", err.Error())
		return
	}

	worked := pSig.Verify(hash, theirPubKey)
	if !worked {
		log.Printf("CloseReqHandler Sig err invalid signature on close tx %s", err.Error())
		return
	}

	// sign close
	mySig, err := nd.SignSimpleClose(q, tx)
	if err != nil {
		log.Printf("CloseReqHandler SignSimpleClose err %s", err.Error())
		return
	}

	myBigSig := sig64.SigDecompress(mySig)

	// put the sighash all byte on the end of both signatures
	myBigSig = append(myBigSig, byte(txscript.SigHashAll))
	theirBigSig = append(theirBigSig, byte(txscript.SigHashAll))

	pre, swap, err := lnutil.FundTxScript(q.MyPub, q.TheirPub)
	if err != nil {
		log.Printf("CloseReqHandler FundTxScript err %s", err.Error())
		return
	}

	// swap if needed
	if swap {
		tx.TxIn[0].Witness = SpendMultiSigWitStack(pre, theirBigSig, myBigSig)
	} else {
		tx.TxIn[0].Witness = SpendMultiSigWitStack(pre, myBigSig, theirBigSig)
	}
	log.Printf(lnutil.TxToString(tx))

	// save channel state to db as closed.
	q.CloseData.Closed = true
	q.CloseData.CloseTxid = tx.TxHash()
	err = nd.SaveQchanUtxoData(q)
	if err != nil {
		log.Printf("CloseReqHandler SaveQchanUtxoData err %s", err.Error())
		return
	}

	// broadcast
	err = nd.SubWallet[q.Coin()].PushTx(tx)
	if err != nil {
		log.Printf("CloseReqHandler NewOutgoingTx err %s", err.Error())
		return
	}

	return
}

func (q *Qchan) GetHtlcTxosWithElkPointsAndRevPub(tx *wire.MsgTx, mine bool, theirElkPoint, myElkPoint, revPub [33]byte) ([]*wire.TxOut, []uint32, error) {
	htlcOutsInTx := make([]*wire.TxOut, 0)
	htlcOutIndexesInTx := make([]uint32, 0)
	htlcOuts := make([]*wire.TxOut, 0)
	for _, h := range q.State.HTLCs {
		txOut, err := q.GenHTLCOutWithElkPointsAndRevPub(h, mine, theirElkPoint, myElkPoint, revPub)
		if err != nil {
			return nil, nil, err
		}
		htlcOuts = append(htlcOuts, txOut)
	}

	for i, out := range tx.TxOut {
		htlcOut := false
		for _, hOut := range htlcOuts {
			if out.Value == hOut.Value && bytes.Equal(out.PkScript, hOut.PkScript) {
				// This is an HTLC output
				log.Printf("Found HTLC output at index %d", i)
				htlcOut = true
				break
			}
		}
		if htlcOut {
			htlcOutsInTx = append(htlcOutsInTx, out)
			htlcOutIndexesInTx = append(htlcOutIndexesInTx, uint32(i))
		}
	}

	return htlcOutsInTx, htlcOutIndexesInTx, nil
}

func (q *Qchan) GetHtlcTxos(tx *wire.MsgTx, mine bool) ([]*wire.TxOut, []uint32, error) {
	revPub, _, _, err := q.GetKeysFromState(mine)
	if err != nil {
		return nil, nil, err
	}

	curElk, err := q.ElkPoint(false, q.State.StateIdx)
	if err != nil {
		return nil, nil, err
	}

	return q.GetHtlcTxosWithElkPointsAndRevPub(tx, mine, q.State.ElkPoint, curElk, revPub)
}

// GetCloseTxos takes in a tx and sets the QcloseTXO fields based on the tx.
// It also returns the spendable (u)txos generated by the close.
// TODO way too long.  Need to split up.
// TODO problem with collisions, and insufficiently high elk receiver...?
func (q *Qchan) GetCloseTxos(tx *wire.MsgTx) ([]portxo.PorTxo, error) {
	if tx == nil {
		return nil, fmt.Errorf("IngesGetCloseTxostCloseTx: nil tx")
	}
	txid := tx.TxHash()
	// double check -- does this tx actually close the channel?
	if !(len(tx.TxIn) == 1 && lnutil.OutPointsEqual(tx.TxIn[0].PreviousOutPoint, q.Op)) {
		return nil, fmt.Errorf("tx %s doesn't spend channel outpoint %s",
			txid.String(), q.Op.String())
	}
	var shIdx, pkhIdx uint32
	var pkhIsMine bool
	cTxos := make([]portxo.PorTxo, 1)
	myPKHPkSript := lnutil.DirectWPKHScript(q.MyRefundPub)

	htlcOutsInTx, htlcOutIndexesInTx, err := q.GetHtlcTxos(tx, false)
	if err != nil {
		return nil, err
	}
	htlcOutsInOurTx, htlcOutIndexesInOurTx, err := q.GetHtlcTxos(tx, true)
	if err != nil {
		return nil, err
	}
	htlcOutsInTx = append(htlcOutsInTx, htlcOutsInOurTx...)
	htlcOutIndexesInTx = append(htlcOutIndexesInTx, htlcOutIndexesInOurTx...)

	shIdx = 999 // set high here to detect if there's no SH output
	// Classify outputs. If output is an HTLC, do nothing, since there is a
	// separate function for that
	for i, out := range tx.TxOut {
		if len(out.PkScript) == 34 {
			htlcOut := false
			for _, idx := range htlcOutIndexesInTx {
				if uint32(i) == idx {
					htlcOut = true
					break
				}
			}

			// There should be only one other script output other than HTLCs which
			// is the closing script with timelock
			if !htlcOut {
				shIdx = uint32(i)
			}
		} else if bytes.Equal(myPKHPkSript, out.PkScript) {
			pkhIdx = uint32(i)
			pkhIsMine = true
		}
	}

	// if pkh is mine, grab it.
	if pkhIsMine {
		log.Printf("got PKH output [%d] from channel close", pkhIdx)
		var pkhTxo portxo.PorTxo // create new utxo and copy into it

		pkhTxo.Op.Hash = txid
		pkhTxo.Op.Index = pkhIdx
		pkhTxo.Height = q.CloseData.CloseHeight
		// keypath same, use different
		pkhTxo.KeyGen = q.KeyGen
		// same keygen as underlying channel, but use is refund
		pkhTxo.KeyGen.Step[2] = UseChannelRefund

		pkhTxo.Mode = portxo.TxoP2WPKHComp
		pkhTxo.Value = tx.TxOut[pkhIdx].Value
		// PKH, could omit this
		pkhTxo.PkScript = tx.TxOut[pkhIdx].PkScript
		cTxos[0] = pkhTxo
	}

	// get state hint based on pkh match.  If pkh is mine, that's their TX & hint.
	// if there's no PKH output for me, the TX is mine, so use my hint.
	var comNum uint64
	if pkhIsMine {
		comNum = GetStateIdxFromTx(tx, q.GetChanHint(false))
	} else {
		comNum = GetStateIdxFromTx(tx, q.GetChanHint(true))
	}
	if comNum > q.State.StateIdx { // future state, uhoh.  Crash for now.
		log.Printf("indicated state %d but we know up to %d",
			comNum, q.State.StateIdx)
		return cTxos, nil
	}

	// if we didn't get the pkh, and the comNum is current, we get the SH output.
	// also we probably closed ourselves.  Regular timeout
	if !pkhIsMine && shIdx < 999 && comNum != 0 && comNum == q.State.StateIdx {
		theirElkPoint, err := q.ElkPoint(false, comNum)
		if err != nil {
			return nil, err
		}

		// build script to store in porTxo, make pubkeys
		timeoutPub := lnutil.AddPubsEZ(q.MyHAKDBase, theirElkPoint)
		revokePub := lnutil.CombinePubs(q.TheirHAKDBase, theirElkPoint)

		script := lnutil.CommitScript(revokePub, timeoutPub, q.Delay)
		// script check.  redundant / just in case
		genSH := fastsha256.Sum256(script)
		if !bytes.Equal(genSH[:], tx.TxOut[shIdx].PkScript[2:34]) {
			log.Printf("got different observed and generated SH scripts.\n")
			log.Printf("in %s:%d, see %x\n", txid, shIdx, tx.TxOut[shIdx].PkScript)
			log.Printf("generated %x \n", genSH)
			log.Printf("revokable pub %x\ntimeout pub %x\n", revokePub, timeoutPub)
		}

		// create the ScriptHash, timeout portxo.
		var shTxo portxo.PorTxo // create new utxo and copy into it
		// use txidx's elkrem as it may not be most recent
		elk, err := q.ElkSnd.AtIndex(comNum)
		if err != nil {
			return nil, err
		}
		// keypath is the same, except for use
		shTxo.KeyGen = q.KeyGen

		shTxo.Op.Hash = txid
		shTxo.Op.Index = shIdx
		shTxo.Height = q.CloseData.CloseHeight

		shTxo.KeyGen.Step[2] = UseChannelHAKDBase

		elkpoint := lnutil.ElkPointFromHash(elk)
		addhash := chainhash.DoubleHashH(append(elkpoint[:], q.MyHAKDBase[:]...))

		shTxo.PrivKey = addhash

		shTxo.Mode = portxo.TxoP2WSHComp
		shTxo.Value = tx.TxOut[shIdx].Value
		shTxo.Seq = uint32(q.Delay)
		shTxo.PreSigStack = make([][]byte, 1) // revoke SH has one presig item
		shTxo.PreSigStack[0] = nil            // and that item is a nil (timeout)

		shTxo.PkScript = script
		cTxos[0] = shTxo
	}

	// if we got the pkh, and the comNum is too old, we can get the SH.  Justice.
	if pkhIsMine && comNum != 0 && comNum < q.State.StateIdx {
		log.Printf("Executing Justice!")

		// ---------- revoked SH is mine
		// invalid previous state, can be grabbed!
		// make MY elk points
		myElkPoint, err := q.ElkPoint(true, comNum)
		if err != nil {
			return nil, err
		}

		theirElkPoint, err := q.ElkPoint(false, comNum)
		if err != nil {
			return nil, err
		}

		timeoutPub := lnutil.AddPubsEZ(q.TheirHAKDBase, myElkPoint)
		revokePub := lnutil.CombinePubs(q.MyHAKDBase, myElkPoint)
		script := lnutil.CommitScript(revokePub, timeoutPub, q.Delay)

		htlcOutsInTx, htlcOutIndexesInTx, err := q.GetHtlcTxosWithElkPointsAndRevPub(tx, false, myElkPoint, theirElkPoint, revokePub)
		if err != nil {
			return nil, err
		}

		// Do this search again with the HTLC TXOs using the correct elkpoints
		// There's probably a better solution for this.

		shIdx = 999 // set high here to detect if there's no SH output
		// Classify outputs. If output is an HTLC, do nothing, since there is a
		// separate function for that
		for i, out := range tx.TxOut {
			if len(out.PkScript) == 34 {
				htlcOut := false
				for _, idx := range htlcOutIndexesInTx {
					if uint32(i) == idx {
						htlcOut = true
						break
					}
				}

				// There should be only one other script output other than HTLCs which
				// is the closing script with timelock
				if !htlcOut {
					shIdx = uint32(i)
				}
			}
		}

		log.Printf("P2SH output from channel (non-HTLC output) is at %d", shIdx)

		// script check
		wshScript := lnutil.P2WSHify(script)
		if !bytes.Equal(wshScript[:], tx.TxOut[shIdx].PkScript) {
			log.Printf("got different observed and generated SH scripts.\n")
			log.Printf("in %s:%d, see %x\n", txid, shIdx, tx.TxOut[shIdx].PkScript)
			log.Printf("generated %x \n", wshScript)
			log.Printf("revokable pub %x\ntimeout pub %x\n", revokePub, timeoutPub)
		}

		// myElkHashR added to HAKD private key
		elk, err := q.ElkRcv.AtIndex(comNum)
		if err != nil {
			return nil, err
		}

		var shTxo portxo.PorTxo // create new utxo and copy into it
		shTxo.KeyGen = q.KeyGen
		shTxo.Op.Hash = txid
		shTxo.Op.Index = shIdx
		shTxo.Height = q.CloseData.CloseHeight

		shTxo.KeyGen.Step[2] = UseChannelHAKDBase

		shTxo.PrivKey = lnutil.ElkScalar(elk)

		// just return the elkScalar and let
		// something modify it before export due to the seq=1 flag.

		shTxo.PkScript = script
		shTxo.Value = tx.TxOut[shIdx].Value
		shTxo.Mode = portxo.TxoP2WSHComp
		shTxo.Seq = 1                         // 1 means grab immediately
		shTxo.PreSigStack = make([][]byte, 1) // timeout SH has one presig item
		shTxo.PreSigStack[0] = []byte{0x01}   // and that item is a 1 (justice)
		cTxos = append(cTxos, shTxo)

		log.Printf("There are %d HTLC Outs to do justice on in this transaction\n", len(htlcOutsInTx))
		// Also grab HTLCs. They are mine now too :)
		for i, txo := range htlcOutsInTx {
			log.Printf("Executing Justice on HTLC TXO!")

			// script check
			htlcScript, err := q.GenHTLCScriptWithElkPointsAndRevPub(q.State.HTLCs[i], false, theirElkPoint, myElkPoint, revokePub)
			if err != nil {
				return nil, err
			}
			wshHTLCScript := lnutil.P2WSHify(htlcScript)

			if !bytes.Equal(wshHTLCScript[:], txo.PkScript) {
				log.Printf("got different observed and generated HTLC scripts.\n")
				log.Printf("in %s:%d, see %x\n", txid, htlcOutIndexesInTx[i], txo.PkScript)
				log.Printf("generated %x \n", wshHTLCScript)
				log.Printf("revokable pub %x\ntimeout pub %x\n", revokePub, timeoutPub)
			}

			var htlcTxo portxo.PorTxo // create new utxo and copy into it
			htlcTxo.KeyGen = q.KeyGen
			htlcTxo.Op.Hash = txid
			htlcTxo.Op.Index = htlcOutIndexesInTx[i]
			htlcTxo.Height = q.CloseData.CloseHeight

			htlcTxo.KeyGen.Step[2] = UseChannelHAKDBase

			htlcTxo.PrivKey = lnutil.ElkScalar(elk)

			// just return the elkScalar and let
			// something modify it before export due to the seq=1 flag.

			htlcTxo.PkScript = script
			htlcTxo.Value = txo.Value
			htlcTxo.Mode = portxo.TxoP2WSHComp
			htlcTxo.Seq = 1                         // 1 means grab immediately
			htlcTxo.PreSigStack = make([][]byte, 1) // timeout SH has one presig item
			htlcTxo.PreSigStack[0] = []byte{0x01}   // and that item is a 1 (justice)
			cTxos = append(cTxos, htlcTxo)
		}
	}
	log.Printf("Returning [%d] cTxos", len(cTxos))
	return cTxos, nil
}
