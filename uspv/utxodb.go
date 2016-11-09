package uspv

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"

	"github.com/boltdb/bolt"
)

/*

DB for the wallet.
Here's the structure:

The adr bucket contains all the addresses being watched.
These are stored as k: keyhash, v: KeyGen (serialized)
The keyhash is either 20 or 32 bytes, not the full pkscript.
The rest of the keygen data must be there.

Adr
|
|-Keyhash : Keygen

Utxos are stored as k: outpoint, v: rest of portxo
v can be nil for watch-only outpoints

Utxo
|
|-OutPoint : Portxo (or nothing)

MiscState
|
contains sync height, number of addresses made

Also store txs for rebroadcast.


*/

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

func (ts *TxStore) OpenDB(filename string) error {
	var err error
	var numKeys uint32
	ts.StateDB, err = bolt.Open(filename, 0644, nil)
	if err != nil {
		return err
	}
	// create buckets if they're not already there
	err = ts.StateDB.Update(func(btx *bolt.Tx) error {
		_, err = btx.CreateBucketIfNotExists(BKToutpoint)
		if err != nil {
			return err
		}
		_, err = btx.CreateBucketIfNotExists(BKTadr)
		if err != nil {
			return err
		}
		_, err = btx.CreateBucketIfNotExists(BKTStxos)
		if err != nil {
			return err
		}
		_, err = btx.CreateBucketIfNotExists(BKTTxns)
		if err != nil {
			return err
		}

		sta, err := btx.CreateBucketIfNotExists(BKTState)
		if err != nil {
			return err
		}

		numKeysBytes := sta.Get(KEYNumKeys)
		if numKeysBytes != nil { // NumKeys exists, read into uint32
			numKeys = lnutil.BtU32(numKeysBytes)
			fmt.Printf("db says %d keys\n", numKeys)
		} else { // no adrs yet, make it 0.  Then make an address.
			fmt.Printf("NumKeys not in DB, must be new DB. 0 Keys\n")
			numKeys = 0
			b0 := lnutil.U32tB(numKeys)
			err = sta.Put(KEYNumKeys, b0)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// make a new address if the DB is new.  Might not work right with no addresses.
	if numKeys == 0 {
		_, err := ts.NewAdr160()
		if err != nil {
			return err
		}
	}
	return nil
}

// Get all Addresses, in order.
func (ts *TxStore) GetAllAddresses() ([]btcutil.Address, error) {
	var i, last uint32 // number of addresses made so far
	var adrSlice []btcutil.Address

	err := ts.StateDB.View(func(btx *bolt.Tx) error {
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

	if last > 1<<30 {
		return nil, fmt.Errorf("Got %d keys stored, expect something reasonable", last)
	}

	for i = 0; i < last; i++ {
		nKg := GetWalletKeygen(i)
		nAdr160 := ts.PathPubHash160(nKg)
		if nAdr160 == nil {
			return nil, fmt.Errorf("NewAdr error: got nil h160")
		}

		wa, err := btcutil.NewAddressWitnessPubKeyHash(nAdr160, ts.Param)
		if err != nil {
			return nil, err
		}
		adrSlice = append(adrSlice, btcutil.Address(wa))
	}
	return adrSlice, nil
}

// GetAllAdrs gets all the addresses hash160s stored in the DB
// unsorted.  This is only for making a filter; for UI use GetAllAdr
// This is probably faster than generating the pubkeys, but I should test that.
func (ts *TxStore) GetAllAdr160s() ([][]byte, error) {

	// all 20byte address pkhs
	var allAdr160s [][]byte

	err := ts.StateDB.View(func(btx *bolt.Tx) error {
		// get the outpoint watch bucket
		adrb := btx.Bucket(BKTadr)
		if adrb == nil {
			return fmt.Errorf("adr bucket not in db")
		}

		// iterate through every outpoint in bucket
		return adrb.ForEach(func(k, _ []byte) error {
			// append each k into the allAdr slice
			// think you have to copy here to avoid toxic bolt data?
			curAdr := make([]byte, len(k))
			copy(curAdr, k)
			allAdr160s = append(allAdr160s, curAdr)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	//	fmt.Printf("GetAllAdr160s found %d adr160s\n", len(allAdr160s))
	//	fmt.Printf("first is %x\n", allAdr160s[0])
	return allAdr160s, nil
}

// make a new change output.  I guess this is supposed to be on a different
// branch than regular addresses...
func (ts *TxStore) NewChangeOut(amt int64) (*wire.TxOut, error) {
	change160, err := ts.NewAdr160() // change is always witnessy
	if err != nil {
		return nil, err
	}
	changeAdr, err := btcutil.NewAddressWitnessPubKeyHash(
		change160, ts.Param)
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

// NewAdr creates a new, never before seen address, and increments the
// DB counter, and returns the hash160 of the pubkey.
func (ts *TxStore) NewAdr160() ([]byte, error) {
	var err error
	if ts.Param == nil {
		return nil, fmt.Errorf("NewAdr error: nil param")
	}

	var n uint32 // number of addresses made so far

	err = ts.StateDB.View(func(btx *bolt.Tx) error {
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
	nAdr160 := ts.PathPubHash160(nKg)
	if nAdr160 == nil {
		return nil, fmt.Errorf("NewAdr error: got nil h160")
	}
	fmt.Printf("adr %d hash is %x\n", n, nAdr160)

	kgBytes := nKg.Bytes()

	// total number of keys (now +1) into 4 bytes
	nKeyNumBytes := lnutil.U32tB(n + 1)

	// write to db file
	err = ts.StateDB.Update(func(btx *bolt.Tx) error {
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

	return nAdr160, nil
}

// SetDBSyncHeight sets sync height of the db, indicated the latest block
// of which it has ingested all the transactions.
func (ts *TxStore) SetDBSyncHeight(n int32) error {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, n)

	return ts.StateDB.Update(func(btx *bolt.Tx) error {
		sta := btx.Bucket(BKTState)
		return sta.Put(KEYTipHeight, buf.Bytes())
	})
}

// SyncHeight returns the chain height to which the db has synced
func (ts *TxStore) GetDBSyncHeight() (int32, error) {
	var n int32
	err := ts.StateDB.View(func(btx *bolt.Tx) error {
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

// GetAllUtxos returns a slice of all portxos in the db. empty slice is OK.
// Doesn't return watch only outpoints
func (ts *TxStore) GetAllUtxos() ([]*portxo.PorTxo, error) {
	var utxos []*portxo.PorTxo
	err := ts.StateDB.View(func(btx *bolt.Tx) error {
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

// GetAllStxos returns a slice of all stxos known to the db. empty slice is OK.
func (ts *TxStore) GetAllStxos() ([]*Stxo, error) {
	// this is almost the same as GetAllUtxos but whatever, it'd be more
	// complicated to make one contain the other or something
	var stxos []*Stxo
	err := ts.StateDB.View(func(btx *bolt.Tx) error {
		old := btx.Bucket(BKTStxos)
		if old == nil {
			return fmt.Errorf("no old txos")
		}
		return old.ForEach(func(k, v []byte) error {
			// have to copy k and v here, otherwise append will crash it.
			// not quite sure why but append does weird stuff I guess.

			// create a new stxo
			x := make([]byte, len(k)+len(v))
			copy(x, k)
			copy(x[len(k):], v)
			newS, err := StxoFromBytes(x)
			if err != nil {
				return err
			}
			// and add it to ram
			stxos = append(stxos, &newS)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return stxos, nil
}

// GetTx takes a txid and returns the transaction.  If we have it.
func (ts *TxStore) GetTx(txid *chainhash.Hash) (*wire.MsgTx, error) {
	rtx := wire.NewMsgTx()

	err := ts.StateDB.View(func(btx *bolt.Tx) error {
		txns := btx.Bucket(BKTTxns)
		if txns == nil {
			return fmt.Errorf("no transactions in db")
		}
		txbytes := txns.Get(txid.CloneBytes())
		if txbytes == nil {
			return fmt.Errorf("tx %s not in db", txid.String())
		}
		buf := bytes.NewBuffer(txbytes)
		return rtx.Deserialize(buf)
	})
	if err != nil {
		return nil, err
	}
	return rtx, nil
}

// GetAllTxs returns all the stored txs
func (ts *TxStore) GetAllTxs() ([]*wire.MsgTx, error) {
	var rtxs []*wire.MsgTx

	err := ts.StateDB.View(func(btx *bolt.Tx) error {
		txns := btx.Bucket(BKTTxns)
		if txns == nil {
			return fmt.Errorf("no transactions in db")
		}

		return txns.ForEach(func(k, v []byte) error {
			tx := wire.NewMsgTx()
			buf := bytes.NewBuffer(v)
			err := tx.Deserialize(buf)
			if err != nil {
				return err
			}
			rtxs = append(rtxs, tx)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return rtxs, nil
}

// GetAllTxids returns all the stored txids. Note that we don't remember
// what height they were at.
/*
Don't think this is needed.
func (ts *TxStore) GetAllTxids() ([]*wire.ShaHash, error) {
	var txids []*wire.ShaHash

	err := ts.StateDB.View(func(btx *bolt.Tx) error {
		txns := btx.Bucket(BKTTxns)
		if txns == nil {
			return fmt.Errorf("no transactions in db")
		}

		return txns.ForEach(func(k, v []byte) error {
			txid, err := wire.NewShaHash(k)
			if err != nil {
				return err
			}
			txids = append(txids, txid)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return txids, nil
}
*/

// GetAllWatchOPs returns all outpoints we're watching.  Both portxos and watch-only.
func (ts *TxStore) GetAllOPs() ([]*wire.OutPoint, error) {
	var OPs []*wire.OutPoint
	// open db
	err := ts.StateDB.View(func(btx *bolt.Tx) error {
		// get the outpoint watch bucket
		dufb := btx.Bucket(BKToutpoint)
		if dufb == nil {
			return fmt.Errorf("watch bucket not in db")
		}

		// iterate through every outpoint in bucket
		return dufb.ForEach(func(k, _ []byte) error {
			// all keys should be 36 bytes
			if len(k) != 36 {
				return fmt.Errorf("%d byte outpoint in db (expect 36)", len(k))
			}
			var opArr [36]byte
			copy(opArr[:], k)
			// deserialize into an outpoint
			curOP := lnutil.OutPointFromBytes(opArr)
			OPs = append(OPs, curOP)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return OPs, nil
}

// RegisterWatchOP registers an outpoint to watch.  Called from ReallySend()
func (ts *TxStore) RegisterWatchOP(op wire.OutPoint) error {
	opArr := lnutil.OutPointToBytes(op)
	// open db
	return ts.StateDB.Update(func(btx *bolt.Tx) error {
		// get the outpoint watch bucket
		dufb := btx.Bucket(BKToutpoint)
		if dufb == nil {
			return fmt.Errorf("watch bucket not in db")
		}
		return dufb.Put(opArr[:], nil)
	})
}

// GetPendingInv returns an inv message containing all txs known to the
// db which are at height 0 (not known to be confirmed).
// This can be useful on startup or to rebroadcast unconfirmed txs.
func (ts *TxStore) GetPendingInv() (*wire.MsgInv, error) {
	// use a map (really a set) do avoid dupes
	txidMap := make(map[chainhash.Hash]struct{})

	utxos, err := ts.GetAllUtxos() // get utxos from db
	if err != nil {
		return nil, err
	}
	stxos, err := ts.GetAllStxos() // get stxos from db
	if err != nil {
		return nil, err
	}

	// iterate through utxos, adding txids of anything with height 0
	for _, utxo := range utxos {
		if utxo.Height == 0 {
			txidMap[utxo.Op.Hash] = struct{}{} // adds to map
		}
	}
	// do the same with stxos based on height at which spent
	for _, stxo := range stxos {
		if stxo.SpendHeight == 0 {
			txidMap[stxo.SpendTxid] = struct{}{}
		}
	}

	invMsg := wire.NewMsgInv()
	for txid := range txidMap {
		item := wire.NewInvVect(wire.InvTypeTx, &txid)
		err = invMsg.AddInvVect(item)
		if err != nil {
			if err != nil {
				return nil, err
			}
		}
	}

	// return inv message with all txids (maybe none)
	return invMsg, nil
}

// ExportUtxo is really *IM*port utxo on this side.
// Not implemented yet.  Fix "ingest many" at the same time eh?
func (s *SPVCon) ExportUtxo(u *portxo.PorTxo) error {
	err := s.TS.GainUtxo(*u)
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

// GainUtxo registers the utxo in the duffel bag
// don't register address; they shouldn't be re-used ever anyway.
func (ts *TxStore) GainUtxo(u portxo.PorTxo) error {
	fmt.Printf("gaining exported utxo %s at height %d'n",
		u.Op.String(), u.Height)
	// serialize porTxo
	utxoBytes, err := u.Bytes()
	if err != nil {
		return err
	}

	// open db
	return ts.StateDB.Update(func(btx *bolt.Tx) error {
		// get the outpoint watch bucket
		dufb := btx.Bucket(BKToutpoint)
		if dufb == nil {
			return fmt.Errorf("duffel bag not in db")
		}

		// add utxo itself
		return dufb.Put(utxoBytes[:36], utxoBytes[36:])
	})
}

func NewPorTxo(
	tx *wire.MsgTx, idx uint32, height int32, kg portxo.KeyGen) (*portxo.PorTxo, error) {
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

// KeyHashFromPkScript extracts the 20 or 32 byte hash from a txout PkScript
func KeyHashFromPkScript(pkscript []byte) []byte {
	// match p2pkh
	if len(pkscript) == 25 && pkscript[0] == 0x76 && pkscript[1] == 0xa9 &&
		pkscript[2] == 0x14 && pkscript[23] == 0x88 && pkscript[24] == 0xac {
		return pkscript[3:23]
	}

	// match p2wpkh
	if len(pkscript) == 22 && pkscript[0] == 0x00 && pkscript[1] == 0x14 {
		return pkscript[2:]
	}

	// match p2wsh
	if len(pkscript) == 34 && pkscript[0] == 0x00 && pkscript[1] == 0x20 {
		return pkscript[2:]
	}

	return nil
}

func (ts *TxStore) Ingest(tx *wire.MsgTx, height int32) (uint32, error) {
	return ts.IngestMany([]*wire.MsgTx{tx}, height)
}

//TODO !!!!!!!!!!!!!!!111
// IngestMany() is way too long and complicated and ugly.  It's like 150 lines
// Need to refactor and clean it up / break it up in to little pieces.
// getting better.

// IngestMany puts txs into the DB atomically.  This can result in a
// gain, a loss, or no result.
// It also checks against the watch list, and returns txs that hit the watchlist.
// IngestMany can probably work OK even if the txs are out of order.
// But don't do that, that's weird and untested.
// also it'll error if you give it more than 1M txs, so don't.
func (ts *TxStore) IngestMany(txs []*wire.MsgTx, height int32) (uint32, error) {
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
	err = ts.StateDB.Update(func(btx *bolt.Tx) error {
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
				keygenBytes := adrb.Get(KeyHashFromPkScript(out.PkScript))
				if keygenBytes != nil {
					fmt.Printf("txout script:%x matched kg: %x\n", out.PkScript, keygenBytes)
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
				if len(v) == 0 {
					// confirmation of unknown / watch only outpoint, send up to ln
					// confirmation match detected; return OP event with nil tx
					fmt.Printf("|||| zomg match  ")
					hitTxs[i] = true // flag to save tx in db

					var opArr [36]byte
					copy(opArr[:], k)
					op := lnutil.OutPointFromBytes(opArr)

					// build new outpoint event
					var ev lnutil.OutPointEvent
					ev.Op = *op          // assign outpoint
					ev.Height = height   // assign height (may be 0)
					ev.Tx = nil          // doesn't do anything but... for clarity
					ts.OPEventChan <- ev // send into the channel...
				}
			}
		}

		// iterate through spent outpoints, then outpoint bucket and look for matches
		// this makes us lose money, which is regrettable, but we need to know.
		// could lose stuff we just gained, that's OK.
		for i, curOP := range spentOPs {
			v := dufb.Get(curOP[:])
			if v != nil && len(v) == 0 {
				fmt.Printf("|||watch only here zomg\n")
				hitTxs[spentTxIdx[i]] = true // just save everything
				op := lnutil.OutPointFromBytes(curOP)
				// build new outpoint event
				var ev lnutil.OutPointEvent
				ev.Op = *op
				ev.Height = height
				ev.Tx = txs[spentTxIdx[i]]
				ts.OPEventChan <- ev
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
				fmt.Printf(lostTxo.String())

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

	fmt.Printf("ingest %d txs, %d hits\n", len(txs), hits)
	return hits, err
}
