package qln

import (
	"bytes"
	"fmt"
	"log"

	"github.com/mit-dci/lit/btcutil/txscript"
	"github.com/mit-dci/lit/wire"
	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/sig64"
)

/*
functions relating to the "justice transaction" (aka penalty transaction)


because we're using the sipa/schnorr delinearization, we don't need to vary the PKH
anymore.  We can hand over 1 point per commit & figure everything out from that.
*/

type JusticeTx struct {
	Sig  [64]byte
	Txid [16]byte
	Amt  int64
	Data [32]byte
	Pkh  [20]byte
	Idx  uint64
}

func (jte *JusticeTx) ToBytes() ([]byte, error) {
	var buf bytes.Buffer

	// write the sig
	_, err := buf.Write(jte.Sig[:])
	if err != nil {
		return nil, err
	}

	// write tx id of the bad tx
	_, err = buf.Write(jte.Txid[:])
	if err != nil {
		return nil, err
	}
	// write the delta for this tx
	_, err = buf.Write(lnutil.I64tB(jte.Amt)[:])
	if err != nil {
		return nil, err
	}

	// then the data
	_, err = buf.Write(jte.Data[:])
	if err != nil {
		return nil, err
	}

	// done
	return buf.Bytes(), nil
}

func JusticeTxFromBytes(jte []byte) (JusticeTx, error) {
	var r JusticeTx
	if len(jte) < 120 || len(jte) > 120 {
		return r, fmt.Errorf("JusticeTx data %d bytes, expect 116", len(jte))
	}

	copy(r.Sig[:], jte[:64])
	copy(r.Txid[:], jte[64:80])
	r.Amt = lnutil.BtI64(jte[80:88])
	copy(r.Data[:], jte[88:])

	return r, nil
}

// BuildWatchTxidSig builds the partial txid and signature pair which can
// be exported to the watchtower.
// This get a channel that is 1 state old.  So we can produce a signature.
func (nd *LitNode) BuildJusticeSig(q *Qchan) error {

	if nd.SubWallet[q.Coin()] == nil {
		return fmt.Errorf("Not connected to coin type %d\n", q.Coin())
	}

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
	elkScalar := lnutil.ElkScalar(elk)
	// get elk point
	elkPoint := lnutil.ElkPointFromHash(elk)
	// overwrite remote elkpoint in channel state
	q.State.ElkPoint = elkPoint

	// make pubkeys, build script
	badRevokePub := lnutil.CombinePubs(q.MyHAKDBase, elkPoint)
	badTimeoutPub := lnutil.AddPubsEZ(q.TheirHAKDBase, elkPoint)
	script := lnutil.CommitScript(badRevokePub, badTimeoutPub, q.Delay)
	scriptHashOutScript := lnutil.P2WSHify(script)

	// TODO: we have to build justics txs for each of the HTLCs too

	// build the bad tx (redundant as we just build most of it...
	badTx, _, _, err := q.BuildStateTxs(false)
	if err != nil {
		return err
	}

	var badAmt int64
	badIdx := uint32(len(badTx.TxOut) + 1)

	log.Printf("made revpub %x timeout pub %x\nscript:%x\nhash %x\n",
		badRevokePub[:], badTimeoutPub[:], script, scriptHashOutScript)
	// figure out which output to bring justice to
	for i, out := range badTx.TxOut {
		log.Printf("txout %d pkscript %x\n", i, out.PkScript)
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
	privBase, err := nd.SubWallet[q.Coin()].GetPriv(kg)
	if err != nil {
		return err
	}
	// combine elk & HAKD base to make signing key
	combinedPrivKey := lnutil.CombinePrivKeyWithBytes(privBase, elkScalar[:])

	// get badtxid
	badTxid := badTx.TxHash()
	// make bad outpoint
	badOP := wire.NewOutPoint(&badTxid, badIdx)
	// make the justice txin, empty sig / witness
	justiceIn := wire.NewTxIn(badOP, nil, nil)
	justiceIn.Sequence = 1
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

	jtxid := justiceTx.TxHash()
	log.Printf("made justice tx %s\n", jtxid.String())
	// get hashcache for signing
	hCache := txscript.NewTxSigHashes(justiceTx)

	// sign with combined key.  Justice txs always have only 1 input, so txin is 0
	bigSig, err := txscript.RawTxInWitnessSignature(
		justiceTx, hCache, 0, badAmt, script, txscript.SigHashAll, combinedPrivKey)
	// truncate sig (last byte is sighash type, always sighashAll)
	bigSig = bigSig[:len(bigSig)-1]

	sig, err := sig64.SigCompress(bigSig)
	if err != nil {
		return err
	}

	var jte JusticeTx
	copy(jte.Sig[:], sig[:])
	copy(jte.Txid[:], badTxid[:16])
	jte.Data = q.State.Data
	jte.Amt = q.State.MyAmt

	justiceBytes, err := jte.ToBytes()
	if err != nil {
		return err
	}

	var justiceBytesFixed [120]byte
	copy(justiceBytesFixed[:], justiceBytes[:120])

	return nd.SaveJusticeSig(q.State.StateIdx, q.WatchRefundAdr, justiceBytesFixed)
}

// SaveJusticeSig save the txid/sig of a justice transaction to the db.  Pretty
// straightforward
func (nd *LitNode) SaveJusticeSig(comnum uint64, pkh [20]byte, txidsig [120]byte) error {
	return nd.LitDB.Update(func(btx *bolt.Tx) error {
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

func (nd *LitNode) LoadJusticeSig(comnum uint64, pkh [20]byte) (JusticeTx, error) {
	var txidsig JusticeTx

	err := nd.LitDB.View(func(btx *bolt.Tx) error {
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

		var err error
		txidsig, err = JusticeTxFromBytes(sigbytes)
		if err != nil {
			return err
		}

		return nil
	})

	return txidsig, err
}

func (nd *LitNode) DumpJusticeDB() ([]JusticeTx, error) {
	var txs []JusticeTx

	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		sigs := btx.Bucket(BKTWatch)
		if sigs == nil {
			return fmt.Errorf("no justice bucket")
		}

		// go through all pkh buckets
		return sigs.ForEach(func(k, _ []byte) error {
			pkhBucket := sigs.Bucket(k)
			if pkhBucket == nil {
				return fmt.Errorf("%x not a bucket", k)
			}
			return pkhBucket.ForEach(func(idx, txidsig []byte) error {
				var jtx JusticeTx
				jtx, err := JusticeTxFromBytes(txidsig)
				if err != nil {
					return err
				}

				copy(jtx.Pkh[:], k[:20])
				jtx.Idx = lnutil.BtU64(idx)

				txs = append(txs, jtx)

				return nil
			})
		})
	})
	return txs, err
}

func (nd *LitNode) ShowJusticeDB() (string, error) {
	var s string

	err := nd.LitDB.View(func(btx *bolt.Tx) error {
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
				s += fmt.Sprintf("\tidx %x\t txidsig: %x\n", idx, txidsig[:80])
				return nil
			})
		})
	})
	return s, err
}

// SendWatch syncs up the remote watchtower with all justice signatures
func (nd *LitNode) SyncWatch(qc *Qchan, watchPeer uint32) error {

	if !nd.ConnectedToPeer(watchPeer) {
		return fmt.Errorf("SyncWatch: not connected to peer %d", watchPeer)
	}
	// if watchUpTo isn't 2 behind the state number, there's nothing to send
	// kindof confusing inequality: can't send state 0 info to watcher when at
	// state 1.  State 0 needs special handling.
	if qc.State.WatchUpTo+2 > qc.State.StateIdx || qc.State.StateIdx < 2 {
		return fmt.Errorf("Channel at state %d, up to %d exported, nothing to do",
			qc.State.StateIdx, qc.State.WatchUpTo)
	}
	// send initial description if we haven't sent anything yet
	if qc.State.WatchUpTo == 0 {
		desc := lnutil.NewWatchDescMsg(watchPeer, qc.Coin(),
			qc.WatchRefundAdr, qc.Delay, 5000, qc.TheirHAKDBase, qc.MyHAKDBase)

		nd.OmniOut <- desc
		// after sending description, must send at least states 0 and 1.
		err := nd.SendWatchComMsg(qc, 0, watchPeer)
		if err != nil {
			return err
		}
		err = nd.SendWatchComMsg(qc, 1, watchPeer)
		if err != nil {
			return err
		}
		qc.State.WatchUpTo = 1
	}
	// send messages to get up to 1 less than current state
	for qc.State.WatchUpTo < qc.State.StateIdx-1 {
		// increment watchupto number
		qc.State.WatchUpTo++
		err := nd.SendWatchComMsg(qc, qc.State.WatchUpTo, watchPeer)
		if err != nil {
			return err
		}
	}
	// save updated WatchUpTo number
	return nd.SaveQchanState(qc)
}

// send WatchComMsg generates and sends the ComMsg to a watchtower
func (nd *LitNode) SendWatchComMsg(qc *Qchan, idx uint64, watchPeer uint32) error {
	// retrieve the sig data from db
	txidsig, err := nd.LoadJusticeSig(idx, qc.WatchRefundAdr)
	if err != nil {
		return err
	}
	// get the elkrem
	elk, err := qc.ElkRcv.AtIndex(idx)
	if err != nil {
		return err
	}

	comMsg := lnutil.NewComMsg(
		watchPeer, qc.Coin(), qc.WatchRefundAdr, *elk, txidsig.Txid, txidsig.Sig)

	nd.OmniOut <- comMsg
	return err
}
