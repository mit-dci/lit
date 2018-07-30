package qln

import (
	"bytes"
	"fmt"
	"log"
	"sync"

	"github.com/mit-dci/lit/elkrem"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"

	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
)

// Uhh, quick channel.  For now.  Once you get greater spire it upgrades to
// a full channel that can do everything.
type Qchan struct {
	// S for stored (on disk), D for derived

	portxo.PorTxo            // S underlying utxo data
	CloseData     QCloseData // S closing outpoint

	MyPub    [33]byte // D my channel specific pubkey
	TheirPub [33]byte // S their channel specific pubkey

	// Refunds are also elkremified
	MyRefundPub    [33]byte // D my refund pubkey for channel break
	TheirRefundPub [33]byte // S their pubkey for channel break

	MyHAKDBase    [33]byte // D my base point for HAKD and timeout keys
	TheirHAKDBase [33]byte // S their base point for HAKD and timeout keys
	// PKH for penalty tx.  Derived
	WatchRefundAdr [20]byte

	// Elkrem is used for revoking state commitments
	ElkSnd *elkrem.ElkremSender   // D derived from channel specific key
	ElkRcv *elkrem.ElkremReceiver // S stored in db

	Delay uint16 // blocks for timeout (default 5 for testing)

	State *StatCom // S current state of channel

	ClearToSend chan bool // send a true here when you get a rev
	ChanMtx     sync.Mutex
	// exists only in ram, doesn't touch disk
}

// 4 + 1 + 8 + 32 + 4 + 33 + 33 + 1 + 5 + 32 + 64 = 217 bytes
type HTLC struct {
	Idx uint32

	Incoming bool
	Amt      int64
	RHash    [32]byte
	Locktime uint32

	MyHTLCBase    [33]byte
	TheirHTLCBase [33]byte

	KeyGen portxo.KeyGen

	Sig [64]byte

	R              [16]byte
	Clearing       bool
	Cleared        bool
	ClearedOnChain bool // To keep track of what HTLCs we claimed on-chain
}

// StatComs are State Commitments.
// all elements are saved to the db.
type StatCom struct {
	StateIdx uint64 // this is the n'th state commitment

	WatchUpTo uint64 // have sent out to watchtowers up to this state  ( < stateidx)

	MyAmt int64 // my channel allocation

	Fee int64 // symmetric fee in absolute satoshis

	Data [32]byte

	// their Amt is the utxo.Value minus this
	Delta int32 // fund amount in-transit; is negative for the pusher
	// Delta for when the channel is in a collision state which needs to be resolved
	Collision int32

	// Elkrem point from counterparty, used to make
	// Homomorphic Adversarial Key Derivation public keys (HAKD)
	ElkPoint     [33]byte // saved to disk, current revealable point
	NextElkPoint [33]byte // Point stored for next state
	N2ElkPoint   [33]byte // Point for state after next (in case of collision)

	sig [64]byte // Counterparty's signature for current state
	// don't write to sig directly; only overwrite via fn() call

	// note sig can be nil during channel creation. if stateIdx isn't 0,
	// sig should have a sig.
	// only one sig is ever stored, to prevent broadcasting the wrong tx.
	// could add a mutex here... maybe will later.

	HTLCIdx       uint32
	InProgHTLC    *HTLC // Current in progress HTLC
	CollidingHTLC *HTLC // HTLC for when the channel is colliding

	CollidingHashDelta     bool // True when colliding between a DeltaSig and HashSig/PreImageSig
	CollidingHashPreimage  bool // True when colliding between HashSig and PreimageSig
	CollidingPreimages     bool // True when colliding between PreimageSig and PreimageSig
	CollidingPreimageDelta bool // True when colliding between a DeltaSig and HashSig/PreImageSig

	// Analogous to the ElkPoints above but used for generating their pubkey for the HTLC
	NextHTLCBase [33]byte
	N2HTLCBase   [33]byte

	MyNextHTLCBase [33]byte
	MyN2HTLCBase   [33]byte

	// Any HTLCs associated with this channel state (can be nil)
	HTLCs []HTLC
}

// QCloseData is the output resulting from an un-cooperative close
// of the channel.  This happens when either party breaks non-cooperatively.
// It describes "your" output, either pkh or time-delay script.
// If you have pkh but can grab the other output, "grabbable" is set to true.
// This can be serialized in a separate bucket

type QCloseData struct {
	// 3 txid / height pairs are stored.  All 3 only are used in the
	// case where you grab their invalid close.
	CloseTxid   chainhash.Hash
	CloseHeight int32
	Closed      bool // if channel is closed; if CloseTxid != -1
}

// ChannelInfo prints info about a channel.
func (nd *LitNode) QchanInfo(q *Qchan) error {
	// display txid instead of outpoint because easier to copy/paste
	log.Printf("CHANNEL %s h:%d %s cap: %d\n",
		q.Op.String(), q.Height, q.KeyGen.String(), q.Value)
	log.Printf("\tPUB mine:%x them:%x REFBASE mine:%x them:%x BASE mine:%x them:%x\n",
		q.MyPub[:4], q.TheirPub[:4], q.MyRefundPub[:4], q.TheirRefundPub[:4],
		q.MyHAKDBase[:4], q.TheirHAKDBase[:4])
	if q.State == nil || q.ElkRcv == nil {
		log.Printf("\t no valid state or elkrem\n")
	} else {
		log.Printf("\ta %d (them %d) state index %d\n",
			q.State.MyAmt, q.Value-q.State.MyAmt, q.State.StateIdx)

		log.Printf("\tdelta:%d HAKD:%x elk@ %d\n",
			q.State.Delta, q.State.ElkPoint[:4], q.ElkRcv.UpTo())
		elkp, _ := q.ElkPoint(false, q.State.StateIdx)
		myRefPub := lnutil.AddPubsEZ(q.MyRefundPub, elkp)
		theirRefPub := lnutil.AddPubsEZ(q.TheirRefundPub, q.State.ElkPoint)
		log.Printf("\tMy Refund: %x Their Refund %x\n", myRefPub[:4], theirRefPub[:4])
	}

	if !q.CloseData.Closed { // still open, finish here
		return nil
	}

	log.Printf("\tCLOSED at height %d by tx: %s\n",
		q.CloseData.CloseHeight, q.CloseData.CloseTxid.String())
	//	clTx, err := t.GetTx(&q.CloseData.CloseTxid)
	//	if err != nil {
	//		return err
	//	}
	//	ctxos, err := q.GetCloseTxos(clTx)
	//	if err != nil {
	//		return err
	//	}

	//	if len(ctxos) == 0 {
	//		log.Printf("\tcooperative close.\n")
	//		return nil
	//	}

	//	log.Printf("\tClose resulted in %d spendable txos\n", len(ctxos))
	//	if len(ctxos) == 2 {
	//		log.Printf("\t\tINVALID CLOSE!!!11\n")
	//	}
	//	for i, u := range ctxos {
	//		log.Printf("\t\t%d) amt: %d spendable: %d\n", i, u.Value, u.Seq)
	//	}
	return nil
}

// Peer returns the local peer index of the channel
func (q *Qchan) Peer() uint32 {
	if q == nil {
		return 0
	}
	return q.KeyGen.Step[3] & 0x7fffffff
}

// Idx returns the local index of the channel
func (q *Qchan) Idx() uint32 {
	if q == nil {
		return 0
	}
	return q.KeyGen.Step[4] & 0x7fffffff
}

// Coin returns the coin type of the channel
func (q *Qchan) Coin() uint32 {
	if q == nil {
		return 0
	}
	return q.KeyGen.Step[1] & 0x7fffffff
}

// ImFirst decides who goes first when it's unclear.  Smaller pubkey goes first.
func (q *Qchan) ImFirst() bool {
	return bytes.Compare(q.MyRefundPub[:], q.TheirRefundPub[:]) == -1
}

// GetChanHint gives the 6 byte hint mask of the channel.  It's derived from the
// hash of the PKH output pubkeys.  "Mine" means the hint is in the tx I store.
// So it's actually a hint for them to decode... which is confusing, but consistent
// with the "mine" bool for transactions, so "my" tx has "my" hint.
// (1<<48 - 1 is the max valid value)
func (q *Qchan) GetChanHint(mine bool) uint64 {
	// could cache these in memory for a slight speedup
	var h []byte
	if mine {
		h = chainhash.DoubleHashB(append(q.MyRefundPub[:], q.TheirRefundPub[:]...))
	} else {
		h = chainhash.DoubleHashB(append(q.TheirRefundPub[:], q.MyRefundPub[:]...))
	}

	if len(h) != 32 {
		return 1 << 63
	}
	// get 6 bytes from that hash (leave top 2 bytes of return value empty)
	x := make([]byte, 8)

	copy(x[2:8], h[2:8])

	return lnutil.BtU64(x)
}

// GetDHSecret gets a per-channel shared secret from the Diffie-Helman of the
// two pubkeys in the fund tx.
func (nd *LitNode) GetDHSecret(q *Qchan) ([]byte, error) {
	if nd.SubWallet[q.Coin()] == nil {
		return nil, fmt.Errorf("Not connected to coin type %d\n", q.Coin())
	}
	if nd == nil || q == nil {
		return nil, fmt.Errorf("GetDHPoint: nil node or channel")
	}

	theirPub, err := btcec.ParsePubKey(q.TheirPub[:], btcec.S256())
	if err != nil {
		return nil, err
	}
	priv, err := nd.SubWallet[q.Coin()].GetPriv(q.KeyGen)
	// if this breaks, return
	if err != nil {
		return nil, err
	}

	return btcec.GenerateSharedSecret(priv, theirPub), nil
}

// GetChannelBalances returns myAmt and theirAmt in the channel
// that aren't locked up in HTLCs in satoshis
func (q *Qchan) GetChannelBalances() (int64, int64) {
	value := q.Value

	for _, h := range q.State.HTLCs {
		if !h.Cleared {
			value -= h.Amt
		}
	}

	myAmt := q.State.MyAmt
	theirAmt := value - myAmt

	return myAmt, theirAmt
}
