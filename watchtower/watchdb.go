package watchtower

import (
	"fmt"
	"log"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/elkrem"
	"github.com/mit-dci/lit/lnutil"

	"github.com/boltdb/bolt"
)

/*
WatchDB has 3 top level buckets -- 2 small ones and one big one.
(also could write it so that the big one is a different file or different machine)

PKHMapBucket is k:v
localChannelId : PKH

ChannelBucket is full of PKH sub-buckets
PKH (lots)
  |
  |-KEYElkRcv : Serialized elkrem receiver (couple KB)
  |
  |-KEYIdx : channelIdx (4 bytes)
  |
  |-KEYStatic : ChanStatic (~100 bytes)


(could also add some metrics, like last write timestamp)

the big one:

TxidBucket is k:v
Txid[:16] : IdxSig (74 bytes)

TODO: both ComMsgs and IdxSigs need to support multiple signatures for HTLCs.
What's nice is that this is the *only* thing needed to support HTLCs.


Potential optimizations to try:
Store less than 16 bytes of the txid
Store

Leave as is for now, but could modify the txid to make it smaller.  Could
HMAC it with a local key to prevent collision attacks and get the txid size down
to 8 bytes or so.  An issue is then you can't re-export the states to other nodes.
Only reduces size by 24 bytes, or about 20%.  Hm.  Try this later.

... actually the more I think about it, this is an easy win.
Also collision attacks seem ineffective; even random false positives would
be no big deal, just a couple ms of CPU to compute the grab tx and see that
it doesn't match.

Yeah can crunch down to 8 bytes, and have the value be 2+ idxSig structs.
In the rare cases where there's a collision, generate both scripts and check.
Quick to check.

To save another couple bytes could make the idx in the idxsig varints.
Only a 3% savings and kindof annoying so will leave that for now.


*/

var (
	BUCKETPKHMap   = []byte("pkm") // bucket for idx:pkh mapping
	BUCKETChandata = []byte("cda") // bucket for channel data (elks, points)
	BUCKETTxid     = []byte("txi") // big bucket with every txid

	KEYStatic = []byte("sta") // static per channel data as value
	KEYElkRcv = []byte("elk") // elkrem receiver
	KEYIdx    = []byte("idx") // index mapping
)

// Opens the DB file for the LnNode
func (w *WatchTower) OpenDB(filename string) error {
	var err error

	// also make the channel for output
	w.OutBox = make(chan *wire.MsgTx, 1)

	w.WatchDB, err = bolt.Open(filename, 0644, nil)
	if err != nil {
		return err
	}
	// create buckets if they're not already there
	err = w.WatchDB.Update(func(btx *bolt.Tx) error {
		_, err := btx.CreateBucketIfNotExists(BUCKETPKHMap)
		if err != nil {
			return err
		}
		_, err = btx.CreateBucketIfNotExists(BUCKETChandata)
		if err != nil {
			return err
		}
		txidBkt, err := btx.CreateBucketIfNotExists(BUCKETTxid)
		if err != nil {
			return err
		}
		// if there are txids in the bucket, set watching to true
		if txidBkt.Stats().KeyN != 0 {
			w.Watching = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// AddNewChannel puts a new channel into the watchtower db.
// Probably need some way to prevent overwrites.
func (w *WatchTower) AddNewChannel(wd WatchannelDescriptor) error {
	return w.WatchDB.Update(func(btx *bolt.Tx) error {
		// open index : pkh mapping bucket
		mapBucket := btx.Bucket(BUCKETPKHMap)
		if mapBucket == nil {
			return fmt.Errorf("no PKHmap bucket")
		}
		// figure out this new channel's index
		// 4B channels forever... could fix, but probably enough.
		var newIdx uint32
		cur := mapBucket.Cursor()
		k, _ := cur.Last() // go to the end
		if k != nil {
			newIdx = lnutil.BtU32(k) + 1 // and add 1
		}
		log.Printf("assigning new channel index %d\n", newIdx)
		newIdxBytes := lnutil.U32tB(newIdx)

		allChanbkt := btx.Bucket(BUCKETChandata)
		if allChanbkt == nil {
			return fmt.Errorf("no Chandata bucket")
		}
		// make new channel bucket
		chanBucket, err := allChanbkt.CreateBucket(wd.DestPKHScript[:])
		if err != nil {
			return err
		}
		// save truncated descriptor for static info (drop elk0)
		wdBytes := wd.ToBytes()
		if len(wdBytes) < 96 {
			return fmt.Errorf("watchdescriptor %d bytes, expect 96")
		}
		chanBucket.Put(KEYStatic, wdBytes[:96])
		log.Printf("saved new channel to pkh %x\n", wd.DestPKHScript)
		// save index
		err = chanBucket.Put(KEYIdx, newIdxBytes)
		if err != nil {
			return err
		}
		// even though we haven't actually added anything to watch for,
		// we're pretty sure there will be soon; the watch tower is "on" at this
		// point so assert "watching".
		w.Watching = true

		// save into index mapping
		return mapBucket.Put(newIdxBytes, wd.DestPKHScript[:])
		// done
	})
}

// AddMsg adds a new message describing a penalty tx to the db.
// optimization would be to add a bunch of messages at once.  Not a huge speedup though.
func (w *WatchTower) AddState(cm ComMsg) error {
	return w.WatchDB.Update(func(btx *bolt.Tx) error {

		// first get the channel bucket, update the elkrem and read the idx
		allChanbkt := btx.Bucket(BUCKETChandata)
		if allChanbkt == nil {
			return fmt.Errorf("no Chandata bucket")
		}
		chanBucket := allChanbkt.Bucket(cm.DestPKH[:])
		if chanBucket == nil {
			return fmt.Errorf("no bucket for channel %x", cm.DestPKH)
		}

		// deserialize elkrems.  Future optimization: could keep
		// all elkrem receivers in RAM for every channel, only writing here
		// each time instead of reading then writing back.
		elkr, err := elkrem.ElkremReceiverFromBytes(chanBucket.Get(KEYElkRcv))
		if err != nil {
			return err
		}
		// add next elkrem hash.  Should work.  If it fails...?
		err = elkr.AddNext(&cm.Elk)
		if err != nil {
			return err
		}
		// fmt.Printf("added elkrem %x at index %d OK\n", cm.Elk[:], elkr.UpTo())

		// get state number, after elk insertion.  also convert to 8 bytes.
		stateNumBytes := lnutil.U64tB(elkr.UpTo())
		// worked, so save it back.  First serialize
		elkBytes, err := elkr.ToBytes()
		if err != nil {
			return err
		}
		// then write back to DB.
		err = chanBucket.Put(KEYElkRcv, elkBytes)
		if err != nil {
			return err
		}
		// get local index of this channel
		cIdxBytes := chanBucket.Get(KEYIdx)
		if cIdxBytes == nil {
			return fmt.Errorf("channel %x has no index", cm.DestPKH)
		}

		// we've updated the elkrem and saved it, so done with channel bucket.
		// next go to txid bucket to save

		txidbkt := btx.Bucket(BUCKETTxid)
		if txidbkt == nil {
			return fmt.Errorf("no txid bucket")
		}
		// create the sigIdx 74 bytes.  A little ugly but only called here and
		// pretty quick.  Maybe make a function for this.
		sigIdxBytes := make([]byte, 74)
		copy(sigIdxBytes[:4], cIdxBytes)           // first 4 bytes is the PKH index
		copy(sigIdxBytes[4:10], stateNumBytes[2:]) // next 6 is state number
		copy(sigIdxBytes[10:], cm.Sig[:])          // the rest is signature

		log.Printf("chan %x (pkh %x) up to state %x\n",
			cIdxBytes, cm.DestPKH, stateNumBytes)
		// save sigIdx into the txid bucket.
		// TODO truncate txid, and deal with collisions.
		return txidbkt.Put(cm.ParTxid[:16], sigIdxBytes)
	})
}

// MatchTxid takes in a txid, checks against the DB, and if there's a hit, returns a
// IdxSig with which to make a JusticeTx.  Hits should be rare.
func (w *WatchTower) MatchTxids(txids []chainhash.Hash) ([]chainhash.Hash, error) {
	var err error
	var hits []chainhash.Hash
	err = w.WatchDB.View(func(btx *bolt.Tx) error {
		// open the big bucket
		txidbkt := btx.Bucket(BUCKETTxid)
		if txidbkt == nil {
			return fmt.Errorf("no txid bucket")
		}

		for i, txid := range txids {
			if i == 0 {
				// coinbase tx cannot be a bad tx
				continue
			}
			b := txidbkt.Get(txid[:16])
			if b != nil {
				log.Printf("zomg hit %s\n", txid.String())
				hits = append(hits, txid)
			}
		}
		return nil
	})
	return hits, err
}

func (w *WatchTower) BlockHandler(bchan chan *wire.MsgBlock) {
	log.Printf("-- started BlockHandler, cap %d\n", cap(bchan))
	for {
		err := w.IngestBlock(<-bchan)
		if err != nil {
			log.Printf(err.Error())
		}
	}
}

func (w *WatchTower) IngestBlock(block *wire.MsgBlock) error {
	if !w.Watching {
		// we're not actually watching anything, ignore blocks
		return nil
	}
	if block == nil || len(block.Transactions) < 2 {
		log.Printf("nil / empty block")
		return nil
	}
	log.Printf("checking block %s, %d txs\n",
		block.BlockHash().String(), len(block.Transactions))

	txids, err := block.TxHashes()
	if err != nil {
		return err
	}

	hits, err := w.MatchTxids(txids)
	if err != nil {
		return err
	}
	if len(hits) > 0 {
		for _, hitTxid := range hits {
			log.Printf("zomg tx %s matched db\n", hitTxid.String())
			for _, tx := range block.Transactions { // inefficient here
				curTxid := tx.TxHash()
				if curTxid.IsEqual(&hitTxid) {
					justice, err := w.BuildJusticeTx(tx)
					if err != nil {
						return err
					}
					log.Printf("made & sent out justice tx %s\n", justice.TxHash().String())
					w.OutBox <- justice
				}
			}
		}
	}
	return nil
}

// Status returns a string describing what's in the watchtower.
func (w *WatchTower) Status() (string, error) {
	var err error
	var s string

	err = w.WatchDB.View(func(btx *bolt.Tx) error {
		// open the big bucket
		txidbkt := btx.Bucket(BUCKETTxid)
		if txidbkt == nil {
			return fmt.Errorf("no txid bucket")
		}

		return txidbkt.ForEach(func(txid, idxsig []byte) error {
			s += fmt.Sprintf("\txid %x\t idxsig: %x\n", txid, idxsig)
			return nil
		})
		return nil
	})
	return s, err
}
