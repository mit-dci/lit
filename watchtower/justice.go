package watchtower

import (
	"bytes"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/elkrem"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/sig64"
)

// BuildJusticeTx takes the badTx and IdxSig found by IngestTx, and returns a
// Justice transaction moving funds with great vengance & furious anger.
// Re-opens the DB which just was closed by IngestTx, but since this almost never
// happens, we need to end IngestTx as quickly as possible.
// Note that you should flag the channel for deletion after the JusticeTx is broadcast.
func (w *WatchTower) BuildJusticeTx(badTx *wire.MsgTx) (*wire.MsgTx, error) {
	var err error

	// wd and elkRcv are the two things we need to get out of the db
	var wd WatchannelDescriptor
	var elkRcv *elkrem.ElkremReceiver
	var iSig *IdxSig

	// open DB and get static channel info
	err = w.WatchDB.View(func(btx *bolt.Tx) error {
		// get
		// open the big bucket
		txidbkt := btx.Bucket(BUCKETTxid)
		if txidbkt == nil {
			return fmt.Errorf("no txid bucket")
		}
		txid := badTx.TxHash()
		idxSigBytes := txidbkt.Get(txid[:16])
		if idxSigBytes == nil {
			return fmt.Errorf("couldn't get txid %x")
		}
		iSig, err = IdxSigFromBytes(idxSigBytes)
		if err != nil {
			return err
		}

		mapBucket := btx.Bucket(BUCKETPKHMap)
		if mapBucket == nil {
			return fmt.Errorf("no PKHmap bucket")
		}
		// figure out who this Justice belongs to
		pkh := mapBucket.Get(lnutil.U32tB(iSig.PKHIdx))
		if pkh == nil {
			return fmt.Errorf("No pkh found for index %d", iSig.PKHIdx)
		}

		channelBucket := btx.Bucket(BUCKETChandata)
		if channelBucket == nil {
			return fmt.Errorf("No channel bucket")
		}

		pkhBucket := channelBucket.Bucket(pkh)
		if pkhBucket == nil {
			return fmt.Errorf("No bucket for pkh %x", pkh)
		}

		static := pkhBucket.Get(KEYStatic)
		if static == nil {
			return fmt.Errorf("No static data for pkh %x", pkh)
		}
		// deserialize static watchDescriptor struct
		wd = WatchannelDescriptorFromBytes(static)

		// get the elkrem receiver
		elkBytes := pkhBucket.Get(KEYElkRcv)
		if elkBytes == nil {
			return fmt.Errorf("No elkrem receiver for pkh %x", pkh)
		}
		// deserialize it
		elkRcv, err = elkrem.ElkremReceiverFromBytes(elkBytes)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// done with DB, could do this in separate func?  or leave here.

	// get the elkrem we need.  above check is redundant huh.
	elkScalarHash, err := elkRcv.AtIndex(iSig.StateIdx)
	if err != nil {
		return nil, err
	}

	_, elkPoint := btcec.PrivKeyFromBytes(btcec.S256(), elkScalarHash[:])

	// build the script so we can match it with a txout
	// to do so, generate Pubkeys for the script

	// get the attacker's base point, cast to a pubkey
	AttackerBase, err := btcec.ParsePubKey(wd.AdversaryBasePoint[:], btcec.S256())
	if err != nil {
		return nil, err
	}

	// get the customer's base point as well
	CustomerBase, err := btcec.ParsePubKey(wd.CustomerBasePoint[:], btcec.S256())
	if err != nil {
		return nil, err
	}

	// timeout key is the attacker's base point combined with the elk-point
	keysForTimeout := lnutil.CombinablePubKeySlice{AttackerBase, elkPoint}
	TimeoutKey := keysForTimeout.Combine()

	// revocable key is the customer's base point combined with the same elk-point
	keysForRev := lnutil.CombinablePubKeySlice{CustomerBase, elkPoint}
	Revkey := keysForRev.Combine()

	// get byte arrays for the combined pubkeys
	var RevArr, TimeoutArr [33]byte
	copy(RevArr[:], Revkey.SerializeCompressed())
	copy(TimeoutArr[:], TimeoutKey.SerializeCompressed())

	// build script from the two combined pubkeys and the channel delay
	script := lnutil.CommitScript(RevArr, TimeoutArr, wd.Delay)

	// get P2WSH output script
	shOutputScript := lnutil.P2WSHify(script)

	// try to match WSH with output from tx
	txoutNum := 999
	for i, out := range badTx.TxOut {
		if bytes.Equal(shOutputScript, out.PkScript) {
			txoutNum = i
			break
		}
	}
	// if txoutNum wasn't set, that means we couldn't find the right txout,
	// so either we've generated the script incorrectly, or we've been led
	// on a wild goose chase of some kind.  If this happens for real (not in
	// testing) then we should nuke the channel after this)
	if txoutNum == 999 {
		// TODO do something else here
		return nil, fmt.Errorf("couldn't match generated script with detected txout")
	}

	justiceAmt := badTx.TxOut[txoutNum].Value - wd.Fee
	justicePkScript := lnutil.DirectWPKHScriptFromPKH(wd.DestPKHScript)
	// build the JusticeTX.  First the output
	justiceOut := wire.NewTxOut(justiceAmt, justicePkScript)
	// now the input
	badtxid := badTx.TxHash()
	badOP := wire.NewOutPoint(&badtxid, uint32(txoutNum))
	justiceIn := wire.NewTxIn(badOP, nil, nil)
	// expand the sig back to 71 bytes
	bigSig := sig64.SigDecompress(iSig.Sig)
	// witness stack is (1, sig) -- 1 means revoked path

	justiceIn.Sequence = 1                // sequence 1 means grab immediately
	justiceIn.Witness = make([][]byte, 2) // timeout SH has one presig item
	justiceIn.Witness[0] = []byte{0x01}   // stack top is a 1, for justice
	justiceIn.Witness[1] = bigSig         // expanded signature goes on last

	// add in&out to the the final justiceTx
	justiceTx := wire.NewMsgTx()
	justiceTx.AddTxIn(justiceIn)
	justiceTx.AddTxOut(justiceOut)

	return justiceTx, nil
}

// don't use this?  inline is OK...
func BuildIdxSig(who uint32, when uint64, sig [64]byte) IdxSig {
	var x IdxSig
	x.PKHIdx = who
	x.StateIdx = when
	x.Sig = sig
	return x
}
