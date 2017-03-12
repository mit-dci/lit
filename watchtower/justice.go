package watchtower

import (
	"bytes"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/txscript"
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
	elkHash, err := elkRcv.AtIndex(iSig.StateIdx)
	if err != nil {
		return nil, err
	}

	elkPoint := lnutil.ElkPointFromHash(elkHash)

	// build the script so we can match it with a txout
	// to do so, generate Pubkeys for the script

	// timeout key is the attacker's base point ez-added with the elk-point
	TimeoutKey := lnutil.AddPubsEZ(wd.AdversaryBasePoint, elkPoint)

	// revocable key is the customer's base point combined with same elk-point
	Revkey := lnutil.CombinePubs(wd.CustomerBasePoint, elkPoint)

	log.Printf("tower build revpub %x \ntimeoutpub %x\n", Revkey, TimeoutKey)
	// build script from the two combined pubkeys and the channel delay
	script := lnutil.CommitScript(Revkey, TimeoutKey, wd.Delay)

	// get P2WSH output script
	shOutputScript := lnutil.P2WSHify(script)
	log.Printf("built script %x\npkscript %x\n", script, shOutputScript)

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
	bigSig = append(bigSig, byte(txscript.SigHashAll)) // put sighash_all byte on at the end

	justiceIn.Sequence = 1                // sequence 1 means grab immediately
	justiceIn.Witness = make([][]byte, 3) // timeout SH has one presig item
	justiceIn.Witness[0] = bigSig         // expanded signature goes on bottom
	justiceIn.Witness[1] = []byte{0x01}   // above sig is a 1, for justice
	justiceIn.Witness[2] = script         // full script goes on at the top

	// add in&out to the the final justiceTx
	justiceTx := wire.NewMsgTx()
	justiceTx.Version = 2 // shouldn't matter, but standardize
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
