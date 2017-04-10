package wallit

import (
	"log"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/uspv"
)

func NewWallit(
	rootkey *hdkeychain.ExtendedKey,
	birthHeight int32, spvhost, path string, p *chaincfg.Params) *Wallit {

	var w Wallit
	w.rootPrivKey = rootkey
	w.Param = p
	w.FreezeSet = make(map[wire.OutPoint]*FrozenTx)

	wallitpath := filepath.Join(path, p.Name)

	// create wallit sub dir if it's not there
	_, err := os.Stat(wallitpath)
	if os.IsNotExist(err) {
		os.Mkdir(wallitpath, 0700)
	}

	// Tricky part here is that we want the sync height to tell the chainhook,
	// so we have to open the db first, then turn on the chainhook, THEN tell
	// chainhook about all our addresses.

	u := new(uspv.SPVCon)
	w.Hook = u

	wallitdbname := filepath.Join(wallitpath, "utxo.db")
	err = w.OpenDB(wallitdbname)
	if err != nil {
		log.Printf("NewWallit crash  %s ", err.Error())
	}
	// get height
	height := w.CurrentHeight()
	log.Printf("DB height %d\n", height)
	if height < birthHeight {
		height = birthHeight
	}
	log.Printf("DB height %d\n", height)
	incomingTx, incomingBlockheight, err := w.Hook.Start(height, spvhost, wallitpath, p)
	if err != nil {
		log.Printf("NewWallit crash  %s ", err.Error())
	}

	// check if there are any addresses.  If there aren't (initial wallet setup)
	// then make an address.
	adrs, err := w.AdrDump()
	if err != nil {
		log.Printf("NewWallit crash  %s ", err.Error())
	}
	if len(adrs) == 0 {
		_ = w.NewAdr()
	}

	// deal with the incoming txs
	go w.TxHandler(incomingTx)

	// deal with incoming height
	go w.HeightHandler(incomingBlockheight)

	return &w
}

func (w *Wallit) TxHandler(incomingTxAndHeight chan lnutil.TxAndHeight) {
	for {
		txah := <-incomingTxAndHeight
		w.Ingest(txah.Tx, txah.Height)
		log.Printf("got tx %s at height %d\n",
			txah.Tx.TxHash().String(), txah.Height)
	}
}

func (w *Wallit) HeightHandler(incomingHeight chan int32) {
	for {
		h := <-incomingHeight
		err := w.SetDBSyncHeight(h)
		if err != nil {
			log.Printf("HeightHandler crash  %s ", err.Error())
		}
	}
}

// OpenDB starts up the database.  Creates the file if it doesn't exist.
func (w *Wallit) OpenDB(filename string) error {
	var err error
	var numKeys uint32
	w.StateDB, err = bolt.Open(filename, 0644, nil)
	if err != nil {
		return err
	}
	// create buckets if they're not already there
	err = w.StateDB.Update(func(btx *bolt.Tx) error {
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
			log.Printf("db says %d keys\n", numKeys)
		} else { // no adrs yet, make it 0.  Then make an address.
			log.Printf("NumKeys not in DB, must be new DB. 0 Keys\n")
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

	return nil
}
