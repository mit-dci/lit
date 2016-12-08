package qln

import (
	"bytes"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/sig64"
	"github.com/mit-dci/lit/watchtower"
)

/*
functions relating to the "justice transaction" (aka penalty transaction)


because we're using the sipa/schnorr delinearization, we don't need to vary the PKH
anymore.  We can hand over 1 point per commit & figure everything out from that.
*/

// BuildWatchTxidSig builds the partial txid and signature pair which can
// be exported to the watchtower.
// This get a channel that is 1 state old.  So we can produce a signature.
func (nd *LnNode) BuildWatchTxidSig(q *Qchan) error {
	var parTxidSig [80]byte // 16 byte txid and 64 byte signature stuck together

	// in this function, "bad" refers to the hypothetical transaction spending the
	// com tx.  "justice" is the tx spending the bad tx

	fee := int64(5000) // fixed fee for now

	// first we need the keys in the bad script.  Start by getting the elk-scalar
	// we should have it at the "current" state number
	elk, err := q.ElkRcv.AtIndex(q.State.StateIdx)
	if err != nil {
		return err
	}
	// build elkpoint, and rewind the channel's remote elkpoint by one state
	// get elk scalar
	elkScalar := ElkScalar(elk)
	// get elk point
	elkPoint := ElkPointFromHash(elk)
	// overwrite remote elkpoint in channel state
	q.State.ElkPoint = elkPoint

	// make pubkeys, build script
	badRevokePub := lnutil.CombinePubs(q.MyHAKDBase, elkPoint)
	badTimeoutPub := lnutil.AddPubsEZ(q.TheirHAKDBase, elkPoint)
	script := lnutil.CommitScript(badRevokePub, badTimeoutPub, q.TimeOut)
	scriptHashOutScript := lnutil.P2WSHify(script)

	// build the bad tx (redundant as we just build most of it...
	badTx, err := q.BuildStateTx(false)

	var badAmt int64
	badIdx := uint32(len(badTx.TxOut) + 1)

	fmt.Printf("made revpub %x timeout pub %x\nscript:%x\nhash %x\n",
		badRevokePub[:], badTimeoutPub[:], script, scriptHashOutScript)
	// figure out which output to bring justice to
	for i, out := range badTx.TxOut {
		fmt.Printf("txout %d pkscript %x\n", i, out.PkScript)
		if bytes.Equal(out.PkScript, scriptHashOutScript) {
			badIdx = uint32(i)
			badAmt = out.Value
			break
		}
	}
	if badIdx > uint32(len(badTx.TxOut)) {
		return fmt.Errorf("BuildWatchTxidSig couldn't find revocable SH output")
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
		return err
	}

	copy(parTxidSig[:16], badTxid[:16])
	copy(parTxidSig[16:], sig[:])

	return nd.SaveJusticeSig(q.State.StateIdx, q.WatchRefundAdr, parTxidSig)
}

// SaveJusticeSig save the txid/sig of a justice transaction to the db.  Pretty
// straightforward
func (nd *LnNode) SaveJusticeSig(comnum uint64, pkh [20]byte, txidsig [80]byte) error {
	return nd.LnDB.Update(func(btx *bolt.Tx) error {
		sigs := btx.Bucket(BKTWatch)
		if sigs == nil {
			return fmt.Errorf("no justice bucket")
		}
		// one bucket per refund PKH
		justBkt, err := sigs.CreateBucketIfNotExists(pkh[:])
		if err != nil {
			return err
		}

		return justBkt.Put(lnutil.U64tB(comnum), txidsig[:])
	})
}

func (nd *LnNode) LoadJusticeSig(comnum uint64, pkh [20]byte) ([80]byte, error) {
	var txidsig [80]byte

	err := nd.LnDB.View(func(btx *bolt.Tx) error {
		sigs := btx.Bucket(BKTWatch)
		if sigs == nil {
			return fmt.Errorf("no justice bucket")
		}
		// one bucket per refund PKH
		justBkt := sigs.Bucket(pkh[:])
		if justBkt == nil {
			return fmt.Errorf("pkh %x not in justice bucket", pkh)
		}
		sigbytes := justBkt.Get(lnutil.U64tB(comnum))
		if sigbytes == nil {
			return fmt.Errorf("state %d not in db under pkh %x", comnum, pkh)
		}
		copy(txidsig[:], sigbytes)
		return nil
	})
	return txidsig, err
}

func (nd *LnNode) ShowJusticeDB() (string, error) {
	var s string

	err := nd.LnDB.View(func(btx *bolt.Tx) error {
		sigs := btx.Bucket(BKTWatch)
		if sigs == nil {
			return fmt.Errorf("no justice bucket")
		}

		// go through all pkh buckets
		return sigs.ForEach(func(k, _ []byte) error {
			s += fmt.Sprintf("Channel refunding to pkh %x\n", k)
			pkhBucket := sigs.Bucket(k)
			if pkhBucket == nil {
				return fmt.Errorf("%x not a bucket", k)
			}
			return pkhBucket.ForEach(func(idx, txidsig []byte) error {
				s += fmt.Sprintf("\tidx %x\t txidsig: %x\n", idx, txidsig)
				return nil
			})
		})
	})
	return s, err
}

//
func (nd *LnNode) SendWatchDesc(qc *Qchan) error {

	// if watchUpTo isn't 2 behind the state number, there's nothing to send
	// kindof confusing inequality: can't send state 0 info to watcher when at
	// state 1, but otherwise makes sense.
	if qc.State.WatchUpTo+2 > qc.State.StateIdx {
		return fmt.Errorf("Channel at state %d, up to %d exported, nothing to do",
			qc.State.StateIdx, qc.State.WatchUpTo)
	}
	// send initial description if we haven't sent anything yet
	if qc.State.WatchUpTo == 0 {

	}
	// send messages to get up to 1 less than current state
	for qc.State.WatchUpTo < qc.State.StateIdx-1 {
		// increment watchupto number
		qc.State.WatchUpTo++
		// retreive the sig data from db
		txidsig, err := nd.LoadJusticeSig(qc.State.WatchUpTo, qc.WatchRefundAdr)
		if err != nil {
			return err
		}
		// get the elkrem
		elk, err := qc.ElkRcv.AtIndex(qc.State.WatchUpTo)
		if err != nil {
			return err
		}
		commsg := new(watchtower.ComMsg)
		commsg.DestPKH = qc.WatchRefundAdr
		commsg.Elk = *elk
		copy(commsg.ParTxid[:], txidsig[:16])
		copy(commsg.Sig[:], txidsig[16:])
		serializedComMsg := commsg.ToBytes()

		// stash to send all?  or just send here

		_, err = nd.WatchCon.Write(
			append([]byte{watchtower.MSGID_WATCH_COMMSG}, serializedComMsg[:]...))
		if err != nil {
			return err
		}
	}

	return nil

}
