package qln

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/dlc"
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnp2p"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/watchtower"
	"github.com/mit-dci/lit/wire"
)

/*
Channels (& multisig) go in the DB here.
first there's the peer bucket.

Here's the structure:

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


MultiWallit:

One LitNode can have a bunch of SubWallets.  This is useful if you want to
have both testnet3 and regtest channels active simultaneously.
The SubWallet is a map of uint32s to Uwallet interfaces.  The identifier for the
channel is the coin's HDCoinType, which is available from the params.

I said regtest is 257 because it's not defined in a BIP, and set to 1
(collision w/ testnet3) in the btcsuite code.

Other coins could use SLIP-44, which will be IPV4 all over again as people
make millions of pointless altcoins to grab that address space.

Since usually there is only 1 wallit connected, there is a DefaultWallet
which functions can use if the wallet is not specified.  The first wallet
to get attached to DefaultWallet.  There is also a bool MultiWallet which is
false while there is only 1 wallet, and true once there are more than one
wallets connected.

You can't remove wallets once they're attached; just restart instead.

*/

// LitNode is the main struct for the node, keeping track of all channel state and
// communicating with the underlying UWallet
type LitNode struct {
	LitDB *bolt.DB // place to write all this down

	NewLitDB lncore.LitStorage

	LitFolder string // path to save stuff

	IdentityKey *koblitz.PrivateKey

	// p2p remote control key
	DefaultRemoteControlKey *koblitz.PublicKey

	// event bus
	Events *eventbus.EventBus

	// Networking
	PeerMan *lnp2p.PeerManager

	// all nodes have a watchtower.  but could have a tower without a node
	Tower watchtower.Watcher

	// discreet log contract manager
	DlcManager *dlc.DlcManager

	// BaseWallet is the underlying wallet which keeps track of utxos, secrets,
	// and network i/o
	// map of cointypes to wallets
	SubWallet map[uint32]UWallet
	// indicates if multiple wallets are connected
	MultiWallet bool
	// cointype of the first (possibly only) wallet connected
	DefaultCoin uint32

	ConnectedCoinTypes map[uint32]bool
	RemoteCons         map[uint32]*RemotePeer
	RemoteMtx          sync.Mutex

	// the current channel that in the process of being created
	// (1 at a time for now)
	InProg *InFlightFund

	// the current channel in process of being dual funded
	InProgDual *InFlightDualFund

	// Nodes don't have Params; their SubWallets do
	// Param *coinparam.Params // network parameters (testnet3, segnet, etc)

	// queue for async messages to RPC user
	UserMessageBox chan string

	// The URL from which lit attempts to resolve the LN address
	TrackerURL string

	ChannelMap    map[[20]byte][]LinkDesc
	ChannelMapMtx sync.Mutex
	AdvTimeout    *time.Ticker

	RPC interface{}

	// Contains the URL string to connect to a SOCKS5 proxy, if provided
	ProxyURL string
	Nat      string

	InProgMultihop []*InFlightMultihop
	MultihopMutex  sync.Mutex

	ExchangeRates map[uint32][]lnutil.RateDesc

	// TODO REFACTORING FIELDS
	PeerMap    map[*lnp2p.Peer]*RemotePeer // we never remove things from here, so this is a memory leak
	PeerMapMtx *sync.Mutex

	OpEventTx *wire.MsgTx // TODO for testing only
}

type LinkDesc struct {
	Link  lnutil.LinkMsg
	Dirty bool
}

type InFlightMultihop struct {
	Path      []lnutil.RouteHop
	Amt       int64
	HHash     [32]byte
	PreImage  [16]byte
	Succeeded bool
}

func (p *InFlightMultihop) Bytes() []byte {
	var buf bytes.Buffer

	wire.WriteVarInt(&buf, 0, uint64(len(p.Path)))
	for _, nd := range p.Path {
		buf.Write(nd.Bytes())
	}

	wire.WriteVarInt(&buf, 0, uint64(p.Amt))

	buf.Write(p.HHash[:])
	buf.Write(p.PreImage[:])

	binary.Write(&buf, binary.BigEndian, p.Succeeded)

	return buf.Bytes()
}

func InFlightMultihopFromBytes(b []byte) (*InFlightMultihop, error) {
	mh := new(InFlightMultihop)

	buf := bytes.NewBuffer(b) // get rid of messageType

	hops, _ := wire.ReadVarInt(buf, 0)
	for i := uint64(0); i < hops; i++ {
		hop, err := lnutil.NewRouteHopFromBytes(buf.Next(24))
		if err != nil {
			return nil, err
		}

		mh.Path = append(mh.Path, *hop)
	}
	amount, _ := wire.ReadVarInt(buf, 0)
	mh.Amt = int64(amount)

	copy(mh.HHash[:], buf.Next(32))
	copy(mh.PreImage[:], buf.Next(16))

	err := binary.Read(buf, binary.BigEndian, &mh.Succeeded)
	if err != nil {
		return mh, err
	}

	return mh, nil
}

type RemotePeer struct {
	Idx      uint32 // the peer index
	Nickname string
	Con      *lndc.Conn
	QCs      map[uint32]*Qchan   // keep map of all peer's channels in ram
	OpMap    map[[36]byte]uint32 // quick lookup for channels
}

// InFlightFund is a funding transaction that has not yet been broadcast
type InFlightFund struct {
	PeerIdx, ChanIdx, Coin uint32
	Amt, InitSend          int64

	op *wire.OutPoint

	done chan uint32
	// use this to avoid crashiness
	mtx sync.Mutex

	Data [32]byte
}

func (inff *InFlightFund) Clear() {
	inff.PeerIdx = 0
	inff.ChanIdx = 0

	inff.Amt = 0
	inff.InitSend = 0
}

// InFlightDualFund is a dual funding transaction that has not yet been broadcast
type InFlightDualFund struct {
	PeerIdx, ChanIdx, CoinType              uint32
	OurAmount, TheirAmount                  int64
	OurInputs, TheirInputs                  []lnutil.DualFundingInput
	OurChangeAddress, TheirChangeAddress    [20]byte
	OurPub, OurRefundPub, OurHAKDBase       [33]byte
	TheirPub, TheirRefundPub, TheirHAKDBase [33]byte
	OurNextHTLCBase, OurN2HTLCBase          [33]byte
	TheirNextHTLCBase, TheirN2HTLCBase      [33]byte
	OurSignatures, TheirSignatures          [][60]byte
	InitiatedByUs                           bool
	OutPoint                                *wire.OutPoint
	done                                    chan *DualFundingResult
	mtx                                     sync.Mutex
}

type DualFundingResult struct {
	ChannelId     uint32
	Error         bool
	Accepted      bool
	DeclineReason uint8
}

func (inff *InFlightDualFund) Clear() {
	inff.PeerIdx = 0
	inff.ChanIdx = 0
	inff.OurAmount = 0
	inff.TheirAmount = 0
	inff.OurInputs = nil
	inff.TheirInputs = nil
	inff.OurChangeAddress = [20]byte{}
	inff.TheirChangeAddress = [20]byte{}
	inff.OurPub = [33]byte{}
	inff.OurRefundPub = [33]byte{}
	inff.OurHAKDBase = [33]byte{}
	inff.TheirPub = [33]byte{}
	inff.TheirRefundPub = [33]byte{}
	inff.TheirHAKDBase = [33]byte{}
	inff.OurNextHTLCBase = [33]byte{}
	inff.OurN2HTLCBase = [33]byte{}
	inff.TheirNextHTLCBase = [33]byte{}
	inff.TheirN2HTLCBase = [33]byte{}

	inff.OurSignatures = nil
	inff.TheirSignatures = nil
	inff.InitiatedByUs = false
}

// GetLnAddr gets the lightning address for this node.
func (nd *LitNode) GetLnAddr() string {
	return nd.PeerMan.GetExternalAddress()
}

// GetPubHostFromPeerIdx gets the pubkey and internet host name for a peer
func (nd *LitNode) GetPubHostFromPeerIdx(idx uint32) ([33]byte, string) {
	var pub [33]byte
	var host string

	p := nd.PeerMan.GetPeerByIdx(int32(idx))
	if p != nil {
		pk := p.GetPubkey()
		copy(pub[:], pk.SerializeCompressed())
		host = p.GetRemoteAddr()
	}

	return pub, host
}

// GetNicknameFromPeerIdx gets the nickname for a peer
func (nd *LitNode) GetNicknameFromPeerIdx(idx uint32) string {
	var nickname string

	p := nd.PeerMan.GetPeerByIdx(int32(idx))
	if p != nil {
		nickname = p.GetNickname()
	}

	return nickname
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

// SaveNicknameForPeerIdx saves/overwrites a nickname for a given peer idx
func (nd *LitNode) SaveNicknameForPeerIdx(nickname string, idx uint32) error {

	peer := nd.PeerMan.GetPeerByIdx(int32(idx))
	if peer == nil {
		return fmt.Errorf("invalid peer ID %d", idx)
	}

	// Actually go and set it.
	pi := peer.IntoPeerInfo()
	err := nd.NewLitDB.GetPeerDB().AddPeer(peer.GetLnAddr(), pi)

	return err // same as if err != nil { return err } ; return nil
}

// SaveQchanUtxoData saves utxo data such as outpoint and close tx / status.
func (nd *LitNode) SaveQchanUtxoData(q *Qchan) error {
	logging.Warnln("someone tried to SaveQchanUtxoData, doing some hacks to make it save only parts of it")

	// XXX This is a horrible hack and we need to change other code to not be
	// dependent on the way this data is saved/loaded.
	opArr := lnutil.OutPointToBytes(q.Op)
	fq, err := nd.GetQchan(opArr)
	if err != nil {
		return nil
	}
	fq.PorTxo = q.PorTxo

	// we also quietly save close data when we call this function
	if q.CloseData.Closed {
		fq.CloseData = q.CloseData
	}

	return nd.SaveQChan(fq)
}

// register a new Qchan in the db
func (nd *LitNode) SaveQChan(q *Qchan) error {
	if q == nil {
		return fmt.Errorf("SaveQChan: nil qchan")
	}

	opArr := lnutil.OutPointToBytes(q.Op)
	cIdBytes := lnutil.U32tB(q.Idx())

	qdata := nd.QchanSerializeToBytes(q)

	// save channel to db.  It has no state, and has no outpoint yet
	err := nd.LitDB.Update(func(btx *bolt.Tx) error {
		var err error

		cdb := btx.Bucket(BKTChannelData)
		if cdb == nil {
			return fmt.Errorf("channel data bucket not found")
		}

		cmb := btx.Bucket(BKTChanMap)
		if cmb == nil {
			return fmt.Errorf("channel map bucket not found")
		}

		// Save both the channel...
		err = cdb.Put(opArr[:], qdata)
		if err != nil {
			return err
		}

		// ...and the channel ID mapping.
		err = cmb.Put(cIdBytes[:], opArr[:])
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// ReloadQchan loads updated data from the db into the qchan.  Loads elkrem
// and state, but does not change qchan info itself.  Faster than GetQchan()
// also reload the channel close state
func (nd *LitNode) ReloadQchanState(qc *Qchan) error {
	opArr := lnutil.OutPointToBytes(qc.Op)

	return nd.LitDB.View(func(btx *bolt.Tx) error {
		b := btx.Bucket(BKTChannelData)
		if b == nil {
			return fmt.Errorf("channel data bucket not found")
		}

		buf := b.Get(opArr[:])
		if buf == nil {
			return fmt.Errorf("channel not found in DB")
		}
		return nd.QchanUpdateFromBytes(qc, buf)
	})
}

// Save / overwrite state of qChan in db
// the descent into the qchan bucket is boilerplate and it'd be nice
// if we can make that it's own function.  Get channel bucket maybe?  But then
// you have to close it...
func (nd *LitNode) SaveQchanState(q *Qchan) error {
	logging.Warnln("someone called SaveQchanState, but this is deprecated.  doing some hacks to make it save just that.")

	// XXX This is a horrible hack and we need to change other code to not be
	// dependent on the way this data is saved/loaded.
	opArr := lnutil.OutPointToBytes(q.Op)
	fq, err := nd.GetQchan(opArr)
	if err != nil {
		return nil
	}
	fq.State = q.State

	return nd.SaveQChan(fq)
}

// GetAllQchans returns a slice of all channels. empty slice is OK.
func (nd *LitNode) GetAllQchans() ([]*Qchan, error) {
	var qChans []*Qchan
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		b := btx.Bucket(BKTChannelData)
		if b == nil {
			return fmt.Errorf("channel data bucket not found")
		}
		return b.ForEach(func(_, buf []byte) error {
			newQc, err := nd.QchanDeserializeFromBytes(buf)
			if err != nil {
				return err // should we not return this?
			}

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

	var qc *Qchan
	var err error
	err = nd.LitDB.View(func(btx *bolt.Tx) error {

		var err error

		b := btx.Bucket(BKTChannelData)
		if b == nil {
			return fmt.Errorf("channel data bucket not found")
		}

		buf := b.Get(opArr[:])
		if buf == nil {
			return fmt.Errorf("channel not found in DB")
		}

		// Go has weird scoping rules, I hope this doesn't break things.
		qc, err = nd.QchanDeserializeFromBytes(buf)
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
			return fmt.Errorf("no channel map")
		}
		op := cmp.Get(lnutil.U32tB(cIdx))
		if op == nil {
			return fmt.Errorf("no channel %d in db", cIdx)
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
	logging.Infof("got op %x\n", op)
	qc, err := nd.GetQchan(op)
	if err != nil {
		return nil, err
	}
	return qc, nil
}

// SaveMultihopPayment saves a new (or updates an existing) multihop payment in the database
func (nd *LitNode) SaveMultihopPayment(p *InFlightMultihop) error {
	err := nd.LitDB.Update(func(btx *bolt.Tx) error {
		cmp := btx.Bucket(BKTPayments)
		if cmp == nil {
			return fmt.Errorf("SaveMultihopPayment: no payments bucket")
		}

		// save hash : payment
		err := cmp.Put(p.HHash[:], p.Bytes())
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (nd *LitNode) GetAllMultihopPayments() ([]*InFlightMultihop, error) {
	var payments []*InFlightMultihop

	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		bkt := btx.Bucket(BKTPayments)
		if bkt == nil {
			return fmt.Errorf("no payments bucket")
		}

		return bkt.ForEach(func(RHash []byte, paymentBytes []byte) error {
			payment, err := InFlightMultihopFromBytes(paymentBytes)
			if err != nil {
				return err
			}

			// add to slice
			payments = append(payments, payment)
			return nil
		})
	})

	return payments, err
}
