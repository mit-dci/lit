package qln

import (
	"fmt"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/mit-dci/lit/elkrem"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/watchtower"
)

/*
Channels (& multisig) go in the DB here.
first there's the peer bucket.

Here's the structure:

Peers
|
|-Pubkey
	|
	|-idx:uint32 - assign a 32bit number to this peer for HD keys and quick ref
	|
	|-channelID (36 byte outpoint)
		|
		|-idx: uint32 - assign a 32 bit number for each channel w/ peer
		|
		|-channel state data
		|
		|-watchtower bucket
			|
			|-state number: txid, sig


PeerMap
|
|-peerIdx(4) : peerPubkey(33)


Right now these buckets are all in one boltDB.  This limits it to one db write
at a time, which for super high thoughput could be too slow.
Later on we can chop it up so that each channel gets it's own db file.

*/

// LnNode is the main struct for the node, keeping track of all channel state and
// communicating with the underlying UWallet
type LitNode struct {
	LitDB *bolt.DB // place to write all this down

	// all nodes have a watchtower.  but could have a tower without a node
	Tower watchtower.WatchTower

	// BaseWallet is the underlying wallet which keeps track of utxos, secrets,
	// and network i/o
	BaseWallet UWallet

	RemoteCons map[uint32]*lndc.LNDConn
	RemoteMtx  sync.Mutex

	// WatchCon is currently just for the watchtower
	WatchCon *lndc.LNDConn // merge these later

	// OmniChan is the channel for the OmniHandler
	OmniIn  chan *lnutil.LitMsg
	OmniOut chan *lnutil.LitMsg

	// the current channel that in the process of being created
	// (1 at a time for now)
	InProg *InFlightFund

	// Params live here... AND SCon
	Param *chaincfg.Params // network parameters (testnet3, segnet, etc)

	// UpdateClear has notifications of when LN channel updates finish.
	// Right now doesn't distinguish between LN channels, so can only do 1 at a time.
	// Need to change this to ... a waitgroup?  map of channels?  Some other
	// structure...

	PushClear      map[chainhash.Hash]chan bool // known good txids and their heights
	PushClearMutex sync.Mutex

	// queue for async messages to RPC user
	UserMessageBox chan string
}

// InFlightFund is a funding transaction that has not yet been broadcast
type InFlightFund struct {
	PeerIdx, ChanIdx uint32
	Amt, InitSend    int64

	op *wire.OutPoint

	done chan uint32
	// use this to avoid crashiness
	mtx sync.Mutex
}

func (inff *InFlightFund) Clear() {
	inff.PeerIdx = 0
	inff.ChanIdx = 0

	inff.Amt = 0
	inff.InitSend = 0
}

func (nd *LitNode) GetPubHostFromPeerIdx(idx uint32) ([33]byte, string) {
	var pub [33]byte
	var host string
	// look up peer in db; need an efficient mapping for this.
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		mp := btx.Bucket(BKTMap)
		if mp == nil {
			return nil
		}
		pubBytes := mp.Get(lnutil.U32tB(idx))
		if pubBytes != nil {
			copy(pub[:], pubBytes[:33])
			host = string(pubBytes[33:])
		}
		return nil
	})
	if err != nil {
		fmt.Printf(err.Error())
	}
	return pub, host
}

// CountKeysInBucket is needed for NewPeer.  Counts keys in a bucket without
// going into the sub-buckets and their keys. 2^32 max.
// returns 0xffffffff if there's an error
func CountKeysInBucket(bkt *bolt.Bucket) uint32 {
	var i uint32
	err := bkt.ForEach(func(_, _ []byte) error {
		i++
		return nil
	})
	if err != nil {
		fmt.Printf("CountKeysInBucket error: %s\n", err.Error())
		return 0xffffffff
	}
	return i
}

// NextPubForPeer returns the next pubkey index to use with the peer.
// It first checks that the peer exists. Read only.
// Feed the indexes into GetFundPUbkey.
func (nd *LitNode) NextIdxForPeer(peerBytes [33]byte) (uint32, uint32, error) {
	var peerIdx, cIdx uint32
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return fmt.Errorf("NextIdxForPeer: no peers")
		}
		pr := prs.Bucket(peerBytes[:])
		if pr == nil {
			return fmt.Errorf("NextIdxForPeer: peer %x not found", peerBytes)
		}
		peerIdxBytes := pr.Get(KEYIdx)
		if peerIdxBytes == nil {
			return fmt.Errorf("NextIdxForPeer: peer %x has no index? db bad", peerBytes)
		}
		peerIdx = lnutil.BtU32(peerIdxBytes) // store for key creation
		// can't use keyN.  Use BucketN.  So we start at 1.  Also this means
		// NO SUB-BUCKETS in peers.  If we want to add sub buckets we'll need
		// to count or track a different way.
		// nah we can't use this.  Gotta count each time.  Lame.
		cIdx = CountKeysInBucket(pr) + 1
		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	return peerIdx, cIdx, nil
}

// GetPeerIdx returns the peer index given a pubkey.  Creates it if it's not there
// yet!  Also return a bool for new..?  not needed?
func (nd *LitNode) GetPeerIdx(pub *btcec.PublicKey, host string) (uint32, error) {
	var idx uint32
	var pubHost []byte
	err := nd.LitDB.Update(func(btx *bolt.Tx) error {
		prs, _ := btx.CreateBucketIfNotExists(BKTPeers) // only errs on name
		thisPeerBkt := prs.Bucket(pub.SerializeCompressed())
		// peer is already registered, return index without altering db.
		if thisPeerBkt != nil {
			idx = lnutil.BtU32(thisPeerBkt.Get(KEYIdx))
			return nil
		}

		// this peer doesn't exist yet.  Add new peer
		mp, _ := btx.CreateBucketIfNotExists(BKTMap)
		idx = CountKeysInBucket(mp) + 1

		// save peer index:pubkey,host into map bucket
		pubHost = pub.SerializeCompressed()
		if host != "" {
			pubHost = append(pubHost, []byte(host)...)
		}
		err := mp.Put(lnutil.U32tB(idx), pub.SerializeCompressed())
		if err != nil {
			return err
		}

		thisPeerBkt, err = prs.CreateBucket(pub.SerializeCompressed())
		if err != nil {
			return err
		}
		return thisPeerBkt.Put(KEYIdx, lnutil.U32tB(idx))
	})
	return idx, err
}

func (nd *LitNode) SaveQchanUtxoData(q *Qchan) error {
	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return fmt.Errorf("no peers")
		}
		pr := prs.Bucket(q.PeerId[:]) // go into this peer's bucket
		if pr == nil {
			return fmt.Errorf("peer %x not in db", q.PeerId)
		}
		opArr := lnutil.OutPointToBytes(q.Op)
		qcBucket := pr.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db under peer %x",
				q.Op.String(), q.PeerId)
		}

		if q.CloseData.Closed {
			closeBytes, err := q.CloseData.ToBytes()
			if err != nil {
				return err
			}
			err = qcBucket.Put(KEYqclose, closeBytes)
			if err != nil {
				return err
			}
		}

		// serialize channel
		qcBytes, err := q.ToBytes()
		if err != nil {
			return err
		}

		// save qchannel
		return qcBucket.Put(KEYutxo, qcBytes)
	})
}

// register a new Qchan in the db
func (nd *LitNode) SaveQChan(q *Qchan) error {
	if q == nil {
		return fmt.Errorf("SaveQChan: nil qchan")
	}

	// save channel to db.  It has no state, and has no outpoint yet
	err := nd.LitDB.Update(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers) // go into bucket for all peers
		if prs == nil {
			return fmt.Errorf("SaveQChan: no peers")
		}
		pr := prs.Bucket(q.PeerId[:]) // go into this peers bucket
		if pr == nil {
			return fmt.Errorf("SaveQChan: peer %x not found", q.PeerId)
		}

		// make bucket for this channel
		qOPArr := lnutil.OutPointToBytes(q.Op)
		qcBucket, err := pr.CreateBucket(qOPArr[:])
		if pr == nil {
			return fmt.Errorf("SaveQChan: can't make channel bucket")
		}

		// serialize channel
		qcBytes, err := q.ToBytes()
		if err != nil {
			return err
		}

		// save qchannel in the bucket
		err = qcBucket.Put(KEYutxo, qcBytes)
		if err != nil {
			return err
		}

		// also save all state; maybe there isn't any ..?
		// serialize elkrem receiver if it exists

		if q.ElkRcv != nil {
			fmt.Printf("--- elk rcv exists, saving\n")

			eb, err := q.ElkRcv.ToBytes()
			if err != nil {
				return err
			}
			// save elkrem
			err = qcBucket.Put(KEYElkRecv, eb)
			if err != nil {
				return err
			}
		}

		// serialize state
		b, err := q.State.ToBytes()
		if err != nil {
			return err
		}
		// save state
		fmt.Printf("writing %d byte state to bucket\n", len(b))
		return qcBucket.Put(KEYState, b)
	})
	if err != nil {
		return err
	}

	return nil
}

// MakeFundTx fills out a channel funding tx.
// You need to give it a partial tx with the inputs and change output
// (everything but the multisig output), the amout of the multisig output,
// the peerID, and the peer's multisig pubkey.
// It then creates the local multisig pubkey, makes the output, and stores
// the multi tx info in the db.  Doesn't RETURN a tx, but the *tx you
// hand it will be filled in.  (but not signed!)
// Returns the multi outpoint and myPubkey (bytes) & err
// also... this is kindof ugly.  It could be re-written as a more integrated func
// which figures out the inputs and outputs.  So basically move
// most of the code from MultiRespHandler() into here.  Yah.. should do that.
//TODO ^^^^^^^^^^
//func (nd *LnNode) MakeFundTx(tx *wire.MsgTx, amt int64, peerIdx, cIdx uint32,
//	peerId, theirPub, theirRefund, theirHAKDbase [33]byte) (*wire.OutPoint, error) {

//	var err error
//	var qc Qchan

//	err = nd.LnDB.Update(func(btx *bolt.Tx) error {
//		prs := btx.Bucket(BKTPeers) // go into bucket for all peers
//		if prs == nil {
//			return fmt.Errorf("MakeMultiTx: no peers")
//		}
//		pr := prs.Bucket(peerId[:]) // go into this peers bucket
//		if pr == nil {
//			return fmt.Errorf("MakeMultiTx: peer %x not found", peerId)
//		}
//		//		peerIdxBytes := pr.Get(KEYIdx) // find peer index
//		//		if peerIdxBytes == nil {
//		//			return fmt.Errorf("MakeMultiTx: peer %x has no index? db bad", peerId)
//		//		}
//		//		peerIdx = BtU32(peerIdxBytes)       // store peer index for key creation
//		//		cIdx = (CountKeysInBucket(pr) << 1) // local, lsb 0

//		qc.TheirPub = theirPub
//		qc.TheirRefundPub = theirRefund
//		qc.TheirHAKDBase = theirHAKDbase
//		qc.Height = -1
//		qc.KeyGen.Depth = 5
//		qc.KeyGen.Step[0] = 44 + 0x80000000
//		qc.KeyGen.Step[1] = 0 + 0x80000000
//		qc.KeyGen.Step[2] = UseChannelFund
//		qc.KeyGen.Step[3] = peerIdx + 0x80000000
//		qc.KeyGen.Step[4] = cIdx + 0x80000000
//		qc.Value = amt
//		qc.Mode = portxo.TxoP2WSHComp

//		myChanPub := nd.GetUsePub(qc.KeyGen, UseChannelFund)

//		// generate multisig output from two pubkeys
//		multiTxOut, err := FundTxOut(theirPub, myChanPub, amt)
//		if err != nil {
//			return err
//		}
//		// stash script for post-sort detection (kindof ugly)
//		outScript := multiTxOut.PkScript
//		tx.AddTxOut(multiTxOut) // add mutlisig output to tx

//		// figure out outpoint of new multiacct
//		txsort.InPlaceSort(tx) // sort before getting outpoint
//		txid := tx.TxSha()     // got the txid

//		// find index... it will actually be 1 or 0 but do this anyway
//		for i, out := range tx.TxOut {
//			if bytes.Equal(out.PkScript, outScript) {
//				qc.Op = *wire.NewOutPoint(&txid, uint32(i))
//				break // found it
//			}
//		}
//		// make new bucket for this mutliout
//		qcOPArr := lnutil.OutPointToBytes(qc.Op)
//		qcBucket, err := pr.CreateBucket(qcOPArr[:])
//		if err != nil {
//			return err
//		}

//		// serialize multiOut
//		qcBytes, err := qc.ToBytes()
//		if err != nil {
//			return err
//		}

//		// save qchannel in the bucket; it has no state yet
//		err = qcBucket.Put(KEYutxo, qcBytes)
//		if err != nil {
//			return err
//		}
//		// stash whole TX in unsigned bucket
//		// you don't need to remember which key goes to which txin
//		// since the outpoint is right there and quick to look up.

//		//TODO -- Problem!  These utxos are not flagged or removed until
//		// the TX is signed and sent.  If other txs happen before the
//		// ack comes in, the signing could fail.  So... call utxos
//		// spent here I guess.

//		var buf bytes.Buffer
//		tx.Serialize(&buf) // no witness yet, but it will be witty
//		return qcBucket.Put(KEYUnsig, buf.Bytes())
//	})
//	if err != nil {
//		return nil, err
//	}

//	return &qc.Op, nil
//}

// RestoreQchanFromBucket loads the full qchan into memory from the
// bucket where it's stored.  Loads the channel info, the elkrems,
// and the current state.
// You have to tell it the peer index because that comes from 1 level
// up in the db.  Also the peer's id pubkey.
// restore happens all at once, but saving to the db can happen
// incrementally (updating states)
// This should populate everything in the Qchan struct: the elkrems and the states.
// Elkrem sender always works; is derived from local key data.
// Elkrem receiver can be "empty" with nothing in it (no data in db)
// Current state can also be not in the DB, which results in
// State *0* for either.  State 0 is no a valid state and states start at
// state index 1.  Data errors within the db will return errors, but having
// *no* data for states or elkrem receiver is not considered an error, and will
// populate with a state 0 / empty elkrem receiver and return that.
func (nd *LitNode) RestoreQchanFromBucket(
	peerIdx uint32, peerPub []byte, bkt *bolt.Bucket) (*Qchan, error) {
	if bkt == nil { // can't do anything without a bucket
		return nil, fmt.Errorf("empty qchan bucket from peer %d", peerIdx)
	}

	// load the serialized channel base description
	qc, err := QchanFromBytes(bkt.Get(KEYutxo))
	if err != nil {
		return nil, err
	}
	qc.CloseData, err = QCloseFromBytes(bkt.Get(KEYqclose))
	if err != nil {
		return nil, err
	}
	// note that peerIndex is not set from deserialization!  set it here!
	// I think it is now because the whole path is in there
	//	qc.KeyGen.Step[3] = peerIdx
	copy(qc.PeerId[:], peerPub)
	// get my channel pubkey
	qc.MyPub = nd.GetUsePub(qc.KeyGen, UseChannelFund)

	// derive my refund / base point from index
	qc.MyRefundPub = nd.GetUsePub(qc.KeyGen, UseChannelRefund)
	qc.MyHAKDBase = nd.GetUsePub(qc.KeyGen, UseChannelHAKDBase)

	// derive my watchtower refund PKH
	watchRefundPub := nd.GetUsePub(qc.KeyGen, UseChannelWatchRefund)
	watchRefundPKHslice := btcutil.Hash160(watchRefundPub[:])
	copy(qc.WatchRefundAdr[:], watchRefundPKHslice)

	qc.State = new(StatCom)

	// load state.  If it exists.
	// if it doesn't, leave as empty state, will fill in
	stBytes := bkt.Get(KEYState)
	if stBytes != nil {
		qc.State, err = StatComFromBytes(stBytes)
		if err != nil {
			return nil, err
		}
	}

	// load elkrem from elkrem bucket.
	// shouldn't error even if nil.  So shouldn't error, ever.  Right?
	// ignore error?
	qc.ElkRcv, err = elkrem.ElkremReceiverFromBytes(bkt.Get(KEYElkRecv))
	if err != nil {
		return nil, err
	}
	if qc.ElkRcv != nil {
		// fmt.Printf("loaded elkrem receiver at state %d\n", qc.ElkRcv.UpTo())
	}

	// derive elkrem sender root from HD keychain
	r := nd.GetElkremRoot(qc.KeyGen)
	// set sender
	qc.ElkSnd = elkrem.NewElkremSender(r)

	return &qc, nil
}

// ReloadQchan loads updated data from the db into the qchan.  Loads elkrem
// and state, but does not change qchan info itself.  Faster than GetQchan()
func (nd *LitNode) ReloadQchan(q *Qchan) error {
	var err error
	opArr := lnutil.OutPointToBytes(q.Op)

	return nd.LitDB.View(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return fmt.Errorf("no peers")
		}
		pr := prs.Bucket(q.PeerId[:]) // go into this peer's bucket
		if pr == nil {
			return fmt.Errorf("peer %x not in db", q.PeerId[:])
		}
		qcBucket := pr.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db under peer %x",
				q.Op.String(), q.PeerId[:])
		}

		// load state and update
		// if it doesn't, leave as empty state, will fill in
		stBytes := qcBucket.Get(KEYState)
		if stBytes == nil {
			return fmt.Errorf("state value empty")
		}
		q.State, err = StatComFromBytes(stBytes)
		if err != nil {
			return err
		}

		// load elkrem from elkrem bucket.
		q.ElkRcv, err = elkrem.ElkremReceiverFromBytes(qcBucket.Get(KEYElkRecv))
		if err != nil {
			return err
		}
		return nil
	})
}

// SetQchanRefund overwrites "theirrefund" and "theirHAKDbase" in a qchan.
//   This is needed after getting a chanACK.
func (nd *LitNode) SetQchanRefund(q *Qchan, refund, hakdBase [33]byte) error {
	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return fmt.Errorf("no peers")
		}
		pr := prs.Bucket(q.PeerId[:]) // go into this peer's bucket
		if pr == nil {
			return fmt.Errorf("peer %x not in db", q.PeerId)
		}
		opArr := lnutil.OutPointToBytes(q.Op)
		qcBucket := pr.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db under peer %x",
				q.Op.String(), q.PeerId)
		}

		// load the serialized channel base description
		qc, err := QchanFromBytes(qcBucket.Get(KEYutxo))
		if err != nil {
			return err
		}
		// modify their refund
		qc.TheirRefundPub = refund
		// modify their HAKDbase
		qc.TheirHAKDBase = hakdBase
		// re -serialize
		qcBytes, err := qc.ToBytes()
		if err != nil {
			return err
		}
		// save/overwrite
		return qcBucket.Put(KEYutxo, qcBytes)
	})
}

// Save / overwrite state of qChan in db
// the descent into the qchan bucket is boilerplate and it'd be nice
// if we can make that it's own function.  Get channel bucket maybe?  But then
// you have to close it...
func (nd *LitNode) SaveQchanState(q *Qchan) error {
	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return fmt.Errorf("no peers")
		}
		pr := prs.Bucket(q.PeerId[:]) // go into this peer's bucket
		if pr == nil {
			return fmt.Errorf("peer %x not in db", q.PeerId)
		}
		opArr := lnutil.OutPointToBytes(q.Op)
		qcBucket := pr.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db under peer %x",
				q.Op.String(), q.PeerId)
		}
		// serialize elkrem receiver
		eb, err := q.ElkRcv.ToBytes()
		if err != nil {
			return err
		}
		// save elkrem
		err = qcBucket.Put(KEYElkRecv, eb)
		if err != nil {
			return err
		}
		// serialize state
		b, err := q.State.ToBytes()
		if err != nil {
			return err
		}
		// save state
		fmt.Printf("writing %d byte state to bucket\n", len(b))
		return qcBucket.Put(KEYState, b)
	})
}

// GetAllQchans returns a slice of all Multiouts. empty slice is OK.
func (nd *LitNode) GetAllQchans() ([]*Qchan, error) {
	var qChans []*Qchan
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return nil
		}
		return prs.ForEach(func(idPub, nothin []byte) error {
			if nothin != nil {
				return nil // non-bucket
			}
			pr := prs.Bucket(idPub) // go into this peer's bucket

			return pr.ForEach(func(op, nthin []byte) error {
				//				fmt.Printf("key %x ", op)
				if nthin != nil {
					//					fmt.Printf("val %x\n", nthin)
					return nil // non-bucket / outpoint
				}
				qcBucket := pr.Bucket(op)
				if qcBucket == nil {
					return nil // nothing stored
				}

				pIdx := lnutil.BtU32(pr.Get(KEYIdx))
				newQc, err := nd.RestoreQchanFromBucket(pIdx, idPub, qcBucket)
				if err != nil {
					return err
				}

				// add to slice
				qChans = append(qChans, newQc)
				return nil
			})
			return nil
		})
		//TODO deal with close txs
		//		for _, qc := range qChans {
		//			if qc.CloseData.Closed {
		//				clTx, err := nd.GetTx(&qc.CloseData.CloseTxid)
		//				if err != nil {
		//					return err
		//				}
		//				_, err = qc.GetCloseTxos(clTx)
		//				if err != nil {
		//					return err
		//				}
		//			}
		//		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return qChans, nil
}

// GetQchan returns a single multi out.  You need to specify the peer
// pubkey and outpoint bytes.
func (nd *LitNode) GetQchan(
	peerArr [33]byte, opArr [36]byte) (*Qchan, error) {

	qc := new(Qchan)
	var err error
	op := lnutil.OutPointFromBytes(opArr)
	err = nd.LitDB.View(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return fmt.Errorf("no peers")
		}
		pr := prs.Bucket(peerArr[:]) // go into this peer's bucket
		if pr == nil {
			return fmt.Errorf("peer %x not in db", peerArr)
		}
		qcBucket := pr.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db under peer %x",
				op.String(), peerArr)
		}

		pIdx := lnutil.BtU32(pr.Get(KEYIdx))

		qc, err = nd.RestoreQchanFromBucket(pIdx, peerArr[:], qcBucket)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// decode close tx, if channel is closed
	//TODO closechans
	//	if qc.CloseData.Closed {
	//		clTx, err := ts.GetTx(&qc.CloseData.CloseTxid)
	//		if err != nil {
	//			return nil, err
	//		}
	//		_, err = qc.GetCloseTxos(clTx)
	//		if err != nil {
	//			return nil, err
	//		}
	//	}
	return qc, nil
}

// GetQGlobalFromIdx gets the globally unique identifiers (pubkey, outpoint)
// from the local index numbers (peer, channel).
// If the UI does it's job well you shouldn't really need this.
// the unique identifiers are returned as []bytes because
// they're probably going right back in to GetQchan()
func (nd *LitNode) GetQGlobalIdFromIdx(
	peerIdx, cIdx uint32) ([]byte, []byte, error) {
	var err error
	var pubBytes, opBytes []byte

	// go into the db
	err = nd.LitDB.View(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return fmt.Errorf("no peers")
		}
		// look through peers for peer index
		prs.ForEach(func(idPub, nothin []byte) error {
			if nothin != nil {
				return nil // non-bucket
			}
			// this is "break" basically
			if opBytes != nil {
				return nil
			}
			pr := prs.Bucket(idPub) // go into this peer's bucket
			if lnutil.BtU32(pr.Get(KEYIdx)) == peerIdx {
				return pr.ForEach(func(op, nthin []byte) error {
					if nthin != nil {
						return nil // non-bucket / outpoint
					}
					// "break"
					if opBytes != nil {
						return nil
					}
					qcBkt := pr.Bucket(op)
					if qcBkt == nil {
						return nil // nothing stored
					}
					// make new qChannel from the db data
					// inefficient but the key index is somewhere
					// in the middle there, like 40 bytes in or something...
					nqc, err := QchanFromBytes(qcBkt.Get(KEYutxo))
					if err != nil {
						return err
					}
					if nqc.KeyGen.Step[4] == cIdx|1<<31 { // hit; done
						pubBytes = idPub
						opBytes = op
					}
					return nil
				})
			}
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if pubBytes == nil || opBytes == nil {
		return nil, nil, fmt.Errorf(
			"channel (%d,%d) not found in db", peerIdx, cIdx)
	}
	return pubBytes, opBytes, nil
}

// GetQchanByIdx is a gets the channel when you don't know the peer bytes and
// outpoint.  Probably shouldn't have to use this if the UI is done right though.
func (nd *LitNode) GetQchanByIdx(peerIdx, cIdx uint32) (*Qchan, error) {
	pubBytes, opBytes, err := nd.GetQGlobalIdFromIdx(peerIdx, cIdx)
	if err != nil {
		return nil, err
	}
	var op [36]byte
	copy(op[:], opBytes)
	var peerArr [33]byte
	copy(peerArr[:], pubBytes)
	qc, err := nd.GetQchan(peerArr, op)
	if err != nil {
		return nil, err
	}
	return qc, nil
}

// SetChanClose sets the address to close to.
func (nd *LitNode) SetChanClose(
	peerBytes []byte, opArr [36]byte, adrArr [20]byte) error {

	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return fmt.Errorf("no peers")
		}
		pr := prs.Bucket(peerBytes[:]) // go into this peer's bucket
		if pr == nil {
			return fmt.Errorf("peer %x not in db", peerBytes)
		}
		multiBucket := pr.Bucket(opArr[:])
		if multiBucket == nil {
			return fmt.Errorf("outpoint (reversed) %x not in db under peer %x",
				opArr, peerBytes)
		}
		err := multiBucket.Put(KEYCladr, adrArr[:])
		if err != nil {
			return err
		}
		return nil
	})
}

// GetChanClose recalls the address the multisig/channel has been requested to
// close to.  If there's nothing there it returns a nil slice and an error.
func (nd *LitNode) GetChanClose(peerBytes []byte, opArr [36]byte) ([]byte, error) {
	adrBytes := make([]byte, 20)

	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers)
		if prs == nil {
			return fmt.Errorf("no peers")
		}
		pr := prs.Bucket(peerBytes[:]) // go into this peer's bucket
		if pr == nil {
			return fmt.Errorf("peer %x not in db", peerBytes)
		}
		multiBucket := pr.Bucket(opArr[:])
		if multiBucket == nil {
			return fmt.Errorf("outpoint (reversed) %x not in db under peer %x",
				opArr, peerBytes)
		}
		adrToxicBytes := multiBucket.Get(KEYCladr)
		if adrToxicBytes == nil {
			return fmt.Errorf("%x in peer %x has no close address",
				opArr, peerBytes)
		}
		copy(adrBytes, adrToxicBytes)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return adrBytes, nil
}
