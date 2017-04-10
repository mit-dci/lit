package wallit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

// const strings for db usage
var (
	// storage of all utxos. top level is outpoints.
	BKToutpoint = []byte("DuffelBag")
	// storage of all addresses being watched.  top level is pkscripts
	BKTadr = []byte("adr")

	BKTStxos = []byte("SpentTxs")  // for bookkeeping / not sure
	BKTTxns  = []byte("Txns")      // all txs we care about, for replays
	BKTState = []byte("MiscState") // misc states of DB

	//	BKTWatch = []byte("watch") // outpoints we're watching for someone else
	// these are in the state bucket
	KEYNumKeys = []byte("NumKeys") // number of p2pkh keys used

	KEYTipHeight = []byte("TipHeight") // height synced to
)

// make a new change output.  I guess this is supposed to be on a different
// branch than regular addresses...
func (w *Wallit) NewChangeOut(amt int64) (*wire.TxOut, error) {
	change160, err := w.NewAdr160() // change is always witnessy
	if err != nil {
		return nil, err
	}
	changeAdr, err := btcutil.NewAddressWitnessPubKeyHash(
		change160, w.Param)
	if err != nil {
		return nil, err
	}
	changeScript, err := txscript.PayToAddrScript(changeAdr)
	if err != nil {
		return nil, err
	}
	changeOut := wire.NewTxOut(amt, changeScript)
	return changeOut, nil
}

// AddPorTxoAdr adds an externally sourced address to the db.  Looks at the keygen
// to derive hash160.
func (w *Wallit) AddPorTxoAdr(kg portxo.KeyGen) error {
	// write to db file
	return w.StateDB.Update(func(btx *bolt.Tx) error {
		adrb := btx.Bucket(BKTadr)
		if adrb == nil {
			return fmt.Errorf("no adr bucket")
		}

		adr160 := w.PathPubHash160(kg)
		log.Printf("adding addr %x\n", adr160)
		// add the 20-byte key-hash into the db
		return adrb.Put(adr160, kg.Bytes())
	})
}

// AdrDump returns all the addresses in the wallit.
// currently returns non-segwit p2pkh addresses, which
// can then be converted somewhere else into bech32 addresses.
func (w *Wallit) AdrDump() ([]btcutil.Address, error) {
	var i, last uint32 // number of addresses made so far
	var adrSlice []btcutil.Address

	err := w.StateDB.View(func(btx *bolt.Tx) error {
		sta := btx.Bucket(BKTState)
		if sta == nil {
			return fmt.Errorf("no state bucket")
		}

		oldNBytes := sta.Get(KEYNumKeys)
		last = lnutil.BtU32(oldNBytes)
		// update the db with number of created keys
		return nil
	})
	if err != nil {
		return nil, err
	}

	if last > 1<<20 {
		return nil, fmt.Errorf("Got %d keys stored, expect something reasonable", last)
	}

	for i = 0; i < last; i++ {
		nKg := GetWalletKeygen(i)
		nAdr160 := w.PathPubHash160(nKg)
		if nAdr160 == nil {
			return nil, fmt.Errorf("NewAdr error: got nil h160")
		}

		wa, err := btcutil.NewAddressPubKeyHash(nAdr160, w.Param)
		if err != nil {
			return nil, err
		}
		adrSlice = append(adrSlice, btcutil.Address(wa))
	}
	return adrSlice, nil
}

// NewAdr creates a new, never before seen address, and increments the
// DB counter, and returns the hash160 of the pubkey.
func (w *Wallit) NewAdr160() ([]byte, error) {
	var err error
	if w.Param == nil {
		return nil, fmt.Errorf("NewAdr error: nil param")
	}

	var n uint32 // number of addresses made so far

	err = w.StateDB.View(func(btx *bolt.Tx) error {
		sta := btx.Bucket(BKTState)
		if sta == nil {
			return fmt.Errorf("no state bucket")
		}

		oldNBytes := sta.Get(KEYNumKeys)
		n = lnutil.BtU32(oldNBytes)
		// update the db with number of created keys
		return nil
	})
	if n > 1<<30 {
		return nil, fmt.Errorf("Got %d keys stored, expect something reasonable", n)
	}

	nKg := GetWalletKeygen(n)
	nAdr160 := w.PathPubHash160(nKg)
	if nAdr160 == nil {
		return nil, fmt.Errorf("NewAdr error: got nil h160")
	}
	log.Printf("adr %d hash is %x\n", n, nAdr160)

	kgBytes := nKg.Bytes()

	// total number of keys (now +1) into 4 bytes
	nKeyNumBytes := lnutil.U32tB(n + 1)

	// write to db file
	err = w.StateDB.Update(func(btx *bolt.Tx) error {
		adrb := btx.Bucket(BKTadr)
		if adrb == nil {
			return fmt.Errorf("no adr bucket")
		}
		sta := btx.Bucket(BKTState)
		if sta == nil {
			return fmt.Errorf("no state bucket")
		}

		// add the 20-byte key-hash into the db
		err = adrb.Put(nAdr160, kgBytes)
		if err != nil {
			return err
		}

		// update the db with number of created keys
		return sta.Put(KEYNumKeys, nKeyNumBytes)
	})
	if err != nil {
		return nil, err
	}
	var adr20 [20]byte
	copy(adr20[:], nAdr160)
	err = w.Hook.RegisterAddress(adr20)
	if err != nil {
		return nil, err
	}

	return nAdr160, nil
}

// SetDBSyncHeight sets sync height of the db, indicated the latest block
// of which it has ingested all the transactions.
func (w *Wallit) SetDBSyncHeight(n int32) error {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, n)

	return w.StateDB.Update(func(btx *bolt.Tx) error {
		sta := btx.Bucket(BKTState)
		return sta.Put(KEYTipHeight, buf.Bytes())
	})
}

// SyncHeight returns the chain height to which the db has synced
func (w *Wallit) GetDBSyncHeight() (int32, error) {
	var n int32
	err := w.StateDB.View(func(btx *bolt.Tx) error {
		sta := btx.Bucket(BKTState)
		if sta == nil {
			return fmt.Errorf("no state")
		}
		t := sta.Get(KEYTipHeight)

		if t == nil { // no height written, so 0
			return nil
		}

		// read 4 byte tip height to n
		err := binary.Read(bytes.NewBuffer(t), binary.BigEndian, &n)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

// SaveTx unconditionally saves a tx in the DB, usually for sending out to nodes
func (w *Wallit) SaveTx(tx *wire.MsgTx) error {
	// open db
	return w.StateDB.Update(func(btx *bolt.Tx) error {
		// get the outpoint watch bucket
		txbkt := btx.Bucket(BKTTxns)
		if txbkt == nil {
			return fmt.Errorf("tx bucket not in db")
		}
		var buf bytes.Buffer
		tx.Serialize(&buf)
		txid := tx.TxHash()
		return txbkt.Put(txid[:], buf.Bytes())
	})
}

func (w *Wallit) UtxoDump() ([]*portxo.PorTxo, error) {
	return w.GetAllUtxos()
}

// GetAllUtxos returns a slice of all portxos in the db. empty slice is OK.
// Doesn't return watch only outpoints
func (w *Wallit) GetAllUtxos() ([]*portxo.PorTxo, error) {
	var utxos []*portxo.PorTxo
	err := w.StateDB.View(func(btx *bolt.Tx) error {
		dufb := btx.Bucket(BKToutpoint)
		if dufb == nil {
			return fmt.Errorf("no duffel bag")
		}
		return dufb.ForEach(func(k, v []byte) error {

			// 0 len v means it's a watch-only utxo, not spendable
			if len(v) == 0 {
				// fmt.Printf("not nil, 0 len slice\n")
				return nil
			}

			// have to copy k and v here, otherwise append will crash it.
			// not quite sure why but append does weird stuff I guess.
			// create a new utxo
			x := make([]byte, len(k)+len(v))
			copy(x, k)
			copy(x[len(k):], v)
			newU, err := portxo.PorTxoFromBytes(x)
			if err != nil {
				return err
			}
			// and add it to ram
			utxos = append(utxos, newU)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return utxos, nil
}

// RegisterWatchOP registers an outpoint to watch.  Called from ReallySend()
func (w *Wallit) RegisterWatchOP(op wire.OutPoint) error {
	opArr := lnutil.OutPointToBytes(op)
	// open db
	return w.StateDB.Update(func(btx *bolt.Tx) error {
		// get the outpoint watch bucket
		dufb := btx.Bucket(BKToutpoint)
		if dufb == nil {
			return fmt.Errorf("watch bucket not in db")
		}
		return dufb.Put(opArr[:], nil)
	})
}

// GainUtxo registers the utxo in the duffel bag
// don't register address; they shouldn't be re-used ever anyway.
func (w *Wallit) GainUtxo(u portxo.PorTxo) error {
	log.Printf("gaining exported utxo %s at height %d\n",
		u.Op.String(), u.Height)
	// serialize porTxo
	utxoBytes, err := u.Bytes()
	if err != nil {
		return err
	}

	// open db
	return w.StateDB.Update(func(btx *bolt.Tx) error {
		// get the outpoint watch bucket
		dufb := btx.Bucket(BKToutpoint)
		if dufb == nil {
			return fmt.Errorf("duffel bag not in db")
		}

		// add utxo itself
		return dufb.Put(utxoBytes[:36], utxoBytes[36:])
	})
}

func NewPorTxo(tx *wire.MsgTx, idx uint32, height int32,
	kg portxo.KeyGen) (*portxo.PorTxo, error) {
	// extract base portxo from tx
	ptxo, err := portxo.ExtractFromTx(tx, idx)
	if err != nil {
		return nil, err
	}

	ptxo.Height = height
	ptxo.KeyGen = kg

	return ptxo, nil
}

// NewPorTxoBytesFromKGBytes just calls NewPorTxo() and de/serializes
// quick shortcut for ingest()
func NewPorTxoBytesFromKGBytes(
	tx *wire.MsgTx, idx uint32, height int32, kgb []byte) ([]byte, error) {

	if len(kgb) != 53 {
		return nil, fmt.Errorf("keygen %d bytes, expect 53", len(kgb))
	}

	var kgarr [53]byte
	copy(kgarr[:], kgb)

	kg := portxo.KeyGenFromBytes(kgarr)

	ptxo, err := NewPorTxo(tx, idx, height, kg)
	if err != nil {
		return nil, err
	}
	return ptxo.Bytes()
}

// Ingest -- take in a tx from the ChainHook
func (w *Wallit) Ingest(tx *wire.MsgTx, height int32) (uint32, error) {
	return w.IngestMany([]*wire.MsgTx{tx}, height)
}

//TODO !!!!!!!!!!!!!!!111
// IngestMany puts txs into the DB atomically.  This can result in a
// gain, a loss, or no result.

// This should always work; there shouldn't be false positives getting to here,
// as those should be handled on the ChainHook level.
// IngestMany can probably work OK even if the txs are out of order.
// But don't do that, that's weird and untested.
// also it'll error if you give it more than 1M txs, so don't.
func (w *Wallit) IngestMany(txs []*wire.MsgTx, height int32) (uint32, error) {
	var hits uint32
	var err error

	cachedShas := make([]*chainhash.Hash, len(txs)) // cache every txid
	hitTxs := make([]bool, len(txs))                // keep track of which txs to store

	// not worth making a struct but these 2 go together

	// spentOPs are all the outpoints being spent by this
	// batch of txs, serialized into 36 byte arrays
	spentOPs := make([][36]byte, 0, len(txs)) // at least 1 txin per tx
	// spendTxIdx tells which tx (in the txs slice) the utxo loss came from
	spentTxIdx := make([]uint32, 0, len(txs))

	if len(txs) < 1 || len(txs) > 1000000 {
		return 0, fmt.Errorf("tried to ingest %d txs, expect 1 to 1M", len(txs))
	}

	// initial in-ram work on all txs.
	for i, tx := range txs {
		// tx has been OK'd by SPV; check tx sanity
		utilTx := btcutil.NewTx(tx) // convert for validation
		// checks basic stuff like there are inputs and ouputs
		err = blockchain.CheckTransactionSanity(utilTx)
		if err != nil {
			return hits, err
		}
		// cache all txids
		cachedShas[i] = utilTx.Hash()
		// before entering into db, serialize all inputs of ingested txs
		for _, txin := range tx.TxIn {
			spentOPs = append(spentOPs, lnutil.OutPointToBytes(txin.PreviousOutPoint))
			spentTxIdx = append(spentTxIdx, uint32(i)) // save tx it came from
		}
	}

	// now do the db write (this is the expensive / slow part)
	err = w.StateDB.Update(func(btx *bolt.Tx) error {
		// get all 4 buckets
		dufb := btx.Bucket(BKToutpoint)
		adrb := btx.Bucket(BKTadr)
		old := btx.Bucket(BKTStxos)
		txns := btx.Bucket(BKTTxns)

		// first gain utxos.
		// for each txout, see if the pkscript matches something we're watching.
		for i, tx := range txs {
			for j, out := range tx.TxOut {
				// Don't try to Get() a nil.  I think? works ok though?
				keygenBytes := adrb.Get(lnutil.KeyHashFromPkScript(out.PkScript))
				if keygenBytes != nil {
					// fmt.Printf("txout script:%x matched kg: %x\n", out.PkScript, keygenBytes)
					// build new portxo

					txob, err := NewPorTxoBytesFromKGBytes(tx, uint32(j), height, keygenBytes)
					if err != nil {
						return err
					}

					// add hits now though
					hits++
					hitTxs[i] = true
					err = dufb.Put(txob[:36], txob[36:])
					if err != nil {
						return err
					}
				}
			}
		}

		// iterate through txids, then outpoint bucket to see if height changes
		// use seek prefix as we know the txid which could match any outpoint
		// with that hash (in practice usually just 0, 1, 2)
		for i, txid := range cachedShas {
			cur := dufb.Cursor()
			pre := txid.CloneBytes()
			// iterate through all outpoints that start with the txid (if any)
			// k is first 36 bytes of portxo, which is the outpoint.
			// v is the rest of the portxo data, or nothing if it's watch only
			for k, v := cur.Seek(pre); bytes.HasPrefix(k, pre); k, v = cur.Next() {
				// note if v is not empty, we'll get back the exported portxo
				// a second time, so we don't need to do the detection here.
				// only do this if OPEventChan has been initialized
				if len(v) == 0 && cap(w.OPEventChan) != 0 {
					// confirmation of unknown / watch only outpoint, send up to ln
					// confirmation match detected; return OP event with nil tx
					// fmt.Printf("|||| zomg match  ")
					hitTxs[i] = true // flag to save tx in db

					var opArr [36]byte
					copy(opArr[:], k)
					op := lnutil.OutPointFromBytes(opArr)

					// build new outpoint event
					var ev lnutil.OutPointEvent
					ev.Op = *op         // assign outpoint
					ev.Height = height  // assign height (may be 0)
					ev.Tx = nil         // doesn't do anything but... for clarity
					w.OPEventChan <- ev // send into the channel...
				}
			}
		}

		// iterate through spent outpoints, then outpoint bucket and look for matches
		// this makes us lose money, which is regrettable, but we need to know.
		// could lose stuff we just gained, that's OK.
		for i, curOP := range spentOPs {
			v := dufb.Get(curOP[:])
			if v != nil && len(v) == 0 && cap(w.OPEventChan) != 0 {
				// fmt.Printf("|||watch only here zomg\n")
				hitTxs[spentTxIdx[i]] = true // just save everything
				op := lnutil.OutPointFromBytes(curOP)
				// build new outpoint event
				var ev lnutil.OutPointEvent
				ev.Op = *op
				ev.Height = height
				ev.Tx = txs[spentTxIdx[i]]
				w.OPEventChan <- ev
			}
			if v != nil && len(v) > 0 {
				hitTxs[spentTxIdx[i]] = true
				// do all this just to figure out value we lost
				x := make([]byte, len(curOP)+len(v))
				copy(x, curOP[:])
				copy(x[len(curOP):], v)
				lostTxo, err := portxo.PorTxoFromBytes(x)
				if err != nil {
					return err
				}
				// print lost portxo
				log.Printf(lostTxo.String())

				// after marking for deletion, save stxo to old bucket
				var st Stxo                               // generate spent txo
				st.PorTxo = *lostTxo                      // assign outpoint
				st.SpendHeight = height                   // spent at height
				st.SpendTxid = *cachedShas[spentTxIdx[i]] // spent by txid
				stxb, err := st.ToBytes()                 // serialize
				if err != nil {
					return err
				}
				// stxos are saved in the DB like portxos, with k:op, v:the rest
				err = old.Put(stxb[:36], stxb[36:]) // write stxo
				if err != nil {
					return err
				}
				err = dufb.Delete(curOP[:])
				if err != nil {
					return err
				}
			}
		}

		// save all txs with hits
		for i, tx := range txs {
			if hitTxs[i] == true {
				hits++
				var buf bytes.Buffer
				tx.Serialize(&buf) // always store witness version
				err = txns.Put(cachedShas[i].CloneBytes(), buf.Bytes())
				if err != nil {
					return err
				}
			}
		}
		return nil
	})

	log.Printf("ingest %d txs, %d hits\n", len(txs), hits)
	return hits, err
}
