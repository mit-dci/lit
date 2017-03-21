package qln

import (
	"fmt"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/btcec"
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

Channels
|
|-channelID (36 byte outpoint)
	|
	|- portxo data (includes peer id, channel ID)
	|
	|- Watchtower: watchtower data
	|
	|- State: state data

Peers
|
|- peerID (33 byte pubkey)
	|
	|- index (4 bytes)
	|
	|- hostname...?
	|
	|- channels..?


PeerMap
|
|-peerIdx(4) : peerPubkey(33)

ChannelMap
|
|-chanIdx(4) : channelID (36 byte outpoint)


Right now these buckets are all in one boltDB.  This limits it to one db write
at a time, which for super high thoughput could be too slow.
Later on we can chop it up so that each channel gets it's own db file.


*/

// LnNode is the main struct for the node, keeping track of all channel state and
// communicating with the underlying UWallet
type LitNode struct {
	LitDB *bolt.DB // place to write all this down

	LitFolder string // path to save stuff

	// all nodes have a watchtower.  but could have a tower without a node
	Tower watchtower.WatchTower

	// BaseWallet is the underlying wallet which keeps track of utxos, secrets,
	// and network i/o
	SubWallet UWallet

	RemoteCons map[uint32]*RemotePeer
	RemoteMtx  sync.Mutex

	// WatchCon is currently just for the watchtower
	WatchCon *lndc.LNDConn // merge these later

	// OmniChan is the channel for the OmniHandler
	OmniIn  chan *lnutil.LitMsg
	OmniOut chan *lnutil.LitMsg

	// the current channel that in the process of being created
	// (1 at a time for now)
	InProg *InFlightFund

	// Nodes don't have Params; their SubWallets do
	// Param *chaincfg.Params // network parameters (testnet3, segnet, etc)

	// queue for async messages to RPC user
	UserMessageBox chan string

	// The port(s) in which it listens for incoming connections
	LisIpPorts []string
}

type RemotePeer struct {
	Idx   uint32 // the peer index
	Con   *lndc.LNDConn
	QCs   map[uint32]*Qchan   // keep map of all peer's channels in ram
	OpMap map[[36]byte]uint32 // quick lookup for channels
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

// GetPubHostFromPeerIdx gets the pubkey and internet host name for a peer
func (nd *LitNode) GetPubHostFromPeerIdx(idx uint32) ([33]byte, string) {
	var pub [33]byte
	var host string
	// look up peer in db
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		mp := btx.Bucket(BKTPeerMap)
		if mp == nil {
			return nil
		}
		pubBytes := mp.Get(lnutil.U32tB(idx))
		if pubBytes != nil {
			copy(pub[:], pubBytes)
		}
		peerBkt := btx.Bucket(BKTPeers)
		if peerBkt == nil {
			return fmt.Errorf("no Peers")
		}
		prBkt := peerBkt.Bucket(pubBytes)
		if prBkt == nil {
			return fmt.Errorf("no peer %x", pubBytes)
		}
		host = string(prBkt.Get(KEYhost))

		return nil
	})
	if err != nil {
		fmt.Printf(err.Error())
	}
	return pub, host
}

// NextIdx returns the next channel index to use.
func (nd *LitNode) NextChannelIdx() (uint32, error) {
	var cIdx uint32
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		cmp := btx.Bucket(BKTChanMap)
		if cmp == nil {
			return fmt.Errorf("NextIdxForPeer: no ChanMap")
		}

		cIdx = uint32(cmp.Stats().KeyN + 1)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return cIdx, nil
}

// GetPeerIdx returns the peer index given a pubkey.  Creates it if it's not there
// yet!  Also return a bool for new..?  not needed?
func (nd *LitNode) GetPeerIdx(pub *btcec.PublicKey, host string) (uint32, error) {
	var idx uint32
	err := nd.LitDB.Update(func(btx *bolt.Tx) error {
		prs := btx.Bucket(BKTPeers) // only errs on name
		thisPeerBkt := prs.Bucket(pub.SerializeCompressed())
		// peer is already registered, return index without altering db.
		if thisPeerBkt != nil {
			idx = lnutil.BtU32(thisPeerBkt.Get(KEYIdx))
			return nil
		}

		// this peer doesn't exist yet.  Add new peer
		mp := btx.Bucket(BKTPeerMap)
		idx = uint32(mp.Stats().KeyN + 1)

		// add index : pubkey into mapping
		err := mp.Put(lnutil.U32tB(idx), pub.SerializeCompressed())
		if err != nil {
			return err
		}

		thisPeerBkt, err = prs.CreateBucket(pub.SerializeCompressed())
		if err != nil {
			return err
		}

		// save peer index in peer bucket
		err = thisPeerBkt.Put(KEYIdx, lnutil.U32tB(idx))
		if err != nil {
			return err
		}

		// save remote host name (if it's there)
		if host != "" {
			err = thisPeerBkt.Put(KEYhost, []byte(host))
			if err != nil {
				return err
			}
		}
		return nil
	})
	return idx, err
}

func (nd *LitNode) SaveQchanUtxoData(q *Qchan) error {
	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTChannel)
		if cbk == nil {
			return fmt.Errorf("no peers")
		}

		opArr := lnutil.OutPointToBytes(q.Op)

		qcBucket := cbk.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db ", q.Op.String())
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

		qOPArr := lnutil.OutPointToBytes(q.Op)

		// make mapping of index to outpoint
		cmp := btx.Bucket(BKTChanMap)
		if cmp == nil {
			return fmt.Errorf("SaveQChan: no channel map bucket")
		}

		// save index : outpoint
		err := cmp.Put(lnutil.U32tB(q.Idx()), qOPArr[:])
		if err != nil {
			return err
		}
		fmt.Printf("saved %d : %s mapping in db\n", q.Idx(), q.Op.String())

		cbk := btx.Bucket(BKTChannel) // go into bucket for all peers
		if cbk == nil {
			return fmt.Errorf("SaveQChan: no channel bucket")
		}

		// make bucket for this channel

		qcBucket, err := cbk.CreateBucket(qOPArr[:])
		if qcBucket == nil {
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

// RestoreQchanFromBucket loads the full qchan into memory from the
// bucket where it's stored.  Loads the channel info, the elkrems,
// and the current state.
// You have to tell it the peer index because that comes from 1 level
// up in the db.  Also the peer's id pubkey.
// restore happens all at once, but saving to the db can happen
// incrementally (updating states)
// This should populate everything int he Qchan struct: the elkrems and the states.
// Elkrem sender always works; is derived from local key data.
// Elkrem receiver can be "empty" with nothing in it (no data in db)
// Current state can also be not in the DB, which results in
// State *0* for either.  State 0 is no a valid state and states start at
// state index 1.  Data errors within the db will return errors, but having
// *no* data for states or elkrem receiver is not considered an error, and will
// populate with a state 0 / empty elkrem receiver and return that.
func (nd *LitNode) RestoreQchanFromBucket(bkt *bolt.Bucket) (*Qchan, error) {
	if bkt == nil { // can't do anything without a bucket
		return nil, fmt.Errorf("empty qchan bucket ")
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

	// make the clear to send chan
	qc.ClearToSend = make(chan bool, 1)
	// set it to true (all qchannels start as clear to send in ram
	// maybe they shouldn't be...?
	qc.ClearToSend <- true

	return &qc, nil
}

// ReloadQchan loads updated data from the db into the qchan.  Loads elkrem
// and state, but does not change qchan info itself.  Faster than GetQchan()
func (nd *LitNode) ReloadQchan(q *Qchan) error {
	var err error
	opArr := lnutil.OutPointToBytes(q.Op)

	return nd.LitDB.View(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTChannel)
		if cbk == nil {
			return fmt.Errorf("no channels")
		}

		qcBucket := cbk.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db", q.Op.String())
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
		cbk := btx.Bucket(BKTChannel)
		if cbk == nil {
			return fmt.Errorf("no channels")
		}

		opArr := lnutil.OutPointToBytes(q.Op)
		qcBucket := cbk.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db ", q.Op.String())
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
		cbk := btx.Bucket(BKTChannel)
		if cbk == nil {
			return fmt.Errorf("no channels")
		}

		opArr := lnutil.OutPointToBytes(q.Op)
		qcBucket := cbk.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db ", q.Op.String())
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

// GetAllQchans returns a slice of all channels. empty slice is OK.
func (nd *LitNode) GetAllQchans() ([]*Qchan, error) {
	var qChans []*Qchan
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTChannel)
		if cbk == nil {
			return fmt.Errorf("no channels")
		}
		return cbk.ForEach(func(op, nothin []byte) error {
			if nothin != nil {
				return nil // non-bucket
			}
			qcBucket := cbk.Bucket(op)
			if qcBucket == nil {
				return nil // nothing stored
			}
			newQc, err := nd.RestoreQchanFromBucket(qcBucket)
			if err != nil {
				return err
			}

			// add to slice
			qChans = append(qChans, newQc)
			return nil

		})
	})
	if err != nil {
		return nil, err
	}
	return qChans, nil
}

// GetQchan returns a single channel.
// pubkey and outpoint bytes.
func (nd *LitNode) GetQchan(opArr [36]byte) (*Qchan, error) {

	qc := new(Qchan)
	var err error
	op := lnutil.OutPointFromBytes(opArr)
	err = nd.LitDB.View(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTChannel)
		if cbk == nil {
			return fmt.Errorf("no channels")
		}

		qcBucket := cbk.Bucket(opArr[:])
		if qcBucket == nil {
			return fmt.Errorf("outpoint %s not in db", op.String())
		}

		qc, err = nd.RestoreQchanFromBucket(qcBucket)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return qc, nil
}

func (nd *LitNode) GetQchanOPfromIdx(cIdx uint32) ([36]byte, error) {
	var rOp [36]byte
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		cmp := btx.Bucket(BKTChanMap)
		if cmp == nil {
			return fmt.Errorf("no chanel map")
		}
		op := cmp.Get(lnutil.U32tB(cIdx))
		if op == nil {
			return fmt.Errorf("no chanel %d in db", cIdx)
		}
		copy(rOp[:], op)
		return nil
	})
	return rOp, err
}

// GetQchanByIdx is a gets the channel when you don't know the peer bytes and
// outpoint.  Probably shouldn't have to use this if the UI is done right though.
func (nd *LitNode) GetQchanByIdx(cIdx uint32) (*Qchan, error) {
	op, err := nd.GetQchanOPfromIdx(cIdx)
	if err != nil {
		return nil, err
	}
	fmt.Printf("got op %x\n", op)
	qc, err := nd.GetQchan(op)
	if err != nil {
		return nil, err
	}
	return qc, nil
}

// SetChanClose sets the address to close to.
func (nd *LitNode) SetChanClose(opArr [36]byte, adrArr [20]byte) error {

	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTChannel)
		if cbk == nil {
			return fmt.Errorf("no channels")
		}

		qBkt := cbk.Bucket(opArr[:])
		if qBkt == nil {
			return fmt.Errorf("outpoint %s not in db", opArr)
		}
		err := qBkt.Put(KEYCladr, adrArr[:])
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
		cbk := btx.Bucket(BKTChannel)
		if cbk == nil {
			return fmt.Errorf("no channels")
		}

		qBkt := cbk.Bucket(opArr[:])
		if qBkt == nil {
			return fmt.Errorf("outpoint (reversed) %x not in db under peer %x",
				opArr, peerBytes)
		}
		adrToxicBytes := qBkt.Get(KEYCladr)
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
