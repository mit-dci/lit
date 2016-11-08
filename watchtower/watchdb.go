package watchtower

import (
	"fmt"

	"li.lan/tx/lit/elkrem"
	"li.lan/tx/lit/lnutil"

	"github.com/boltdb/bolt"
	"github.com/roasbeef/btcd/wire"
)

/*
WatchDB has 3 top level buckets -- 2 small ones and one big one.

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
  |
  |-HTLC bucket // implement later
	  |
	  |- StateIdx : EncData (104 bytes)


(could also add some metrics, like last write timestamp)

the big one:

TxidBucket is k:v
Txid[:16] : IdxSig (74 bytes)

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

func (s *WatchTower) AddNewChannel(wd WatchannelDescriptor) error {
	return s.WatchDB.Update(func(btx *bolt.Tx) error {
		// open index : pkh mapping bucket
		mbkt := btx.Bucket(BUCKETPKHMap)
		if mbkt == nil {
			return fmt.Errorf("no PKHmap bucket")
		}
		// figure out this new channel's index
		cIdxBytes := lnutil.U32tB(uint32(mbkt.Stats().KeyN)) // this breaks if >4B chans
		allChanbkt := btx.Bucket(BUCKETChandata)
		if allChanbkt == nil {
			return fmt.Errorf("no Chandata bucket")
		}
		// make new channel bucket
		cbkt, err := allChanbkt.CreateBucket(wd.DestPKHScript[:])
		if err != nil {
			return err
		}
		// save truncated descriptor for static info (drop elk0)
		wdBytes := wd.ToBytes()
		if len(wdBytes) < 96 {
			return fmt.Errorf("watchdescriptor %d bytes, expect 96")
		}
		cbkt.Put(KEYStatic, wdBytes[:96])

		var elkr elkrem.ElkremReceiver
		_ = elkr.AddNext(&wd.ElkZero) // first add; can't fail
		elkBytes, err := elkr.ToBytes()
		if err != nil {
			return err
		}
		// save the (first) elkrem
		err = cbkt.Put(KEYElkRcv, elkBytes)
		if err != nil {
			return err
		}
		// save index
		err = cbkt.Put(KEYIdx, cIdxBytes)
		if err != nil {
			return err
		}
		// save into index mapping
		return mbkt.Put(cIdxBytes, wd.DestPKHScript[:])

		// done
	})
}

// AddMsg adds a new message describing a penalty tx to the db.
// A later optimization would be to add a bunch of messages at once.
func (s *WatchTower) AddMsg(cm ComMsg) error {
	return s.WatchDB.Update(func(btx *bolt.Tx) error {

		// first get the channel bucket, update the elkrem and read the idx
		allChanbkt := btx.Bucket(BUCKETChandata)
		if allChanbkt == nil {
			return fmt.Errorf("no Chandata bucket")
		}
		cbkt := allChanbkt.Bucket(cm.DestPKHScript[:])
		if cbkt == nil {
			return fmt.Errorf("no bucket for channel %x", cm.DestPKHScript)
		}

		// deserialize elkrems.  Future optimization: could keep
		// all elkrem receivers in RAM for every channel, only writing here
		// each time instead of reading then writing back.
		elkr, err := elkrem.ElkremReceiverFromBytes(cbkt.Get(KEYElkRcv))
		if err != nil {
			return err
		}
		// add next elkrem hash.  Should work.  If it fails...?
		err = elkr.AddNext(&cm.Elk)
		if err != nil {
			return err
		}

		// get state number, after elk insertion.  also convert to 8 bytes.
		stateNumBytes := lnutil.U64tB(elkr.UpTo())
		// worked, so save it back.  First serialize
		elkBytes, err := elkr.ToBytes()
		if err != nil {
			return err
		}
		// then write back to DB.
		err = cbkt.Put(KEYElkRcv, elkBytes)
		if err != nil {
			return err
		}
		// get local index of this channel
		cIdxBytes := cbkt.Get(KEYIdx)
		if cIdxBytes == nil {
			return fmt.Errorf("channel %x has no index", cm.DestPKHScript)
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
		copy(sigIdxBytes[:4], cIdxBytes)
		copy(sigIdxBytes[4:10], stateNumBytes[2:])
		copy(sigIdxBytes[10:], cm.Sig[:])

		// save sigIdx into the txid bucket.
		return txidbkt.Put(cm.Txid[:8], sigIdxBytes)
	})
}

// CheckTxids takes a slice of txids and sees if any are in the
// DB.  If there is, ComMsg are returned which can then be turned into txs.
// can take the txid slice direct from a msgBlock after block has been
// merkle-checked.
func (s *WatchTower) CheckTxids(inTxids []wire.ShaHash) ([]ComMsg, error) {
	var hitTxids []ComMsg
	err := s.WatchDB.View(func(btx *bolt.Tx) error {
		bkt := btx.Bucket(BUCKETTxid)
		for _, txid := range inTxids {
			idxsig := bkt.Get(txid[:8])
			if idxsig != nil { // hit!!!!1 whoa!
				// Call WatchMsg construction function here
				var sm ComMsg
				copy(sm.Txid[:], txid[:16])
				// that wasn't it.  make a real function

				hitTxids = append(hitTxids, sm)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return hitTxids, nil
}
