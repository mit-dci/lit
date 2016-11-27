package qln

import (
	"fmt"

	"github.com/mit-dci/lit/elkrem"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Uhh, quick channel.  For now.  Once you get greater spire it upgrades to
// a full channel that can do everything.
type Qchan struct {
	// S for stored (on disk), D for derived

	portxo.PorTxo            // S underlying utxo data
	CloseData     QCloseData // S closing outpoint

	MyPub    [33]byte // D my channel specific pubkey
	TheirPub [33]byte // S their channel specific pubkey

	PeerId [33]byte // D useful for quick traverse of db

	// Refunds are also elkremified
	MyRefundPub    [33]byte // D my refund pubkey for channel break
	TheirRefundPub [33]byte // S their pubkey for channel break

	MyHAKDBase    [33]byte // D my base point for HAKD and timeout keys
	TheirHAKDBase [33]byte // S their base point for HAKD and timeout keys

	// Elkrem is used for revoking state commitments
	ElkSnd *elkrem.ElkremSender   // D derived from channel specific key
	ElkRcv *elkrem.ElkremReceiver // S stored in db

	TimeOut uint16 // blocks for timeout (default 5 for testing)

	State *StatCom // S current state of channel

	// PKH for penalty tx.  Derived
	WatchRefundAdr [20]byte
}

// StatComs are State Commitments.
// all elements are saved to the db.
type StatCom struct {
	StateIdx uint64 // this is the n'th state commitment

	WatchUpTo uint64 // have sent out to watchtowers up to this state  ( <= stateidx)

	MyAmt int64 // my channel allocation
	// their Amt is the utxo.Value minus this
	Delta int32 // fun amount in-transit; is negative for the pusher

	// Elkrem point from counterparty, used to make
	// Homomorphic Adversarial Key Derivation public keys (HAKD)
	ElkPointR     [33]byte // saved to disk, revealable point
	PrevElkPointR [33]byte // When you haven't gotten their revocation elkrem yet.

	ElkPointT     [33]byte // their timeout elk point; needed for script
	PrevElkPointT [33]byte // When you haven't gotten their revocation elkrem yet.

	sig [64]byte // Counterparty's signature (for StatCom tx)
	// don't write to sig directly; only overwrite via fn() call

	// note sig can be nil during channel creation. if stateIdx isn't 0,
	// sig should have a sig.
	// only one sig is ever stored, to prevent broadcasting the wrong tx.
	// could add a mutex here... maybe will later.
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
func (nd *LnNode) QchanInfo(q *Qchan) error {
	// display txid instead of outpoint because easier to copy/paste
	fmt.Printf("CHANNEL %s h:%d %s cap: %d\n",
		q.Op.Hash.String(), q.Height, q.KeyGen.String(), q.Value)
	fmt.Printf("\tPUB mine:%x them:%x REFBASE mine:%x them:%x BASE mine:%x them:%x\n",
		q.MyPub[:4], q.TheirPub[:4], q.MyRefundPub[:4], q.TheirRefundPub[:4],
		q.MyHAKDBase[:4], q.TheirHAKDBase[:4])
	if q.State == nil || q.ElkRcv == nil {
		fmt.Printf("\t no valid state or elkrem\n")
	} else {

		fmt.Printf("\ta %d (them %d) state index %d\n",
			q.State.MyAmt, q.Value-q.State.MyAmt, q.State.StateIdx)

		fmt.Printf("\tdelta:%d HAKD:%x prevHAKD:%x elk@ %d\n",
			q.State.Delta, q.State.ElkPointR[:4], q.State.PrevElkPointR[:4],
			q.ElkRcv.UpTo())
		elkp, _ := q.ElkPoint(false, q.State.StateIdx)
		myRefPub := lnutil.CombinePubs(q.MyRefundPub, elkp)

		theirRefPub := lnutil.CombinePubs(q.TheirRefundPub, q.State.ElkPointR)
		fmt.Printf("\tMy Refund: %x Their Refund %x\n", myRefPub[:4], theirRefPub[:4])
	}

	if !q.CloseData.Closed { // still open, finish here
		return nil
	}

	fmt.Printf("\tCLOSED at height %d by tx: %s\n",
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
	//		fmt.Printf("\tcooperative close.\n")
	//		return nil
	//	}

	//	fmt.Printf("\tClose resulted in %d spendable txos\n", len(ctxos))
	//	if len(ctxos) == 2 {
	//		fmt.Printf("\t\tINVALID CLOSE!!!11\n")
	//	}
	//	for i, u := range ctxos {
	//		fmt.Printf("\t\t%d) amt: %d spendable: %d\n", i, u.Value, u.Seq)
	//	}
	return nil
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
func (nd *LnNode) GetDHSecret(q *Qchan) ([]byte, error) {

	if nd == nil || q == nil {
		return nil, fmt.Errorf("GetDHPoint: nil node or channel")
	}

	theirPub, err := btcec.ParsePubKey(q.TheirPub[:], btcec.S256())
	if err != nil {
		return nil, err
	}
	priv := nd.GetPriv(q.KeyGen)
	// not sure what happens if this breaks.  Maybe it always works.

	return btcec.GenerateSharedSecret(priv, theirPub), nil
}
