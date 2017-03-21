package lnutil

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/fastsha256"
)

// TxAndHeight is just a tx, and the height at which it was confirmed.
type TxAndHeight struct {
	Tx     *wire.MsgTx
	Height int32
}

// OutPointEvent is a message describing events concerning an outpoint.
// There's 2 event types: confirmation and spend.  If the Tx pointer is nil,
// then it's a confirm.  If the Tx has an actual MsgTx in there, it's a spend.
// The Height refers to either the confirmation height
// or the height at which it was spent. (0 means seen but unconfirmed)
type OutPointEvent struct {
	Op     wire.OutPoint // the outpoint being described
	Height int32         // the height of the event
	Tx     *wire.MsgTx   // the tx spending the outpoint
}

// need this because before I was comparing pointers maybe?
// so they were the same outpoint but stored in 2 places so false negative?
func OutPointsEqual(a, b wire.OutPoint) bool {
	if !a.Hash.IsEqual(&b.Hash) {
		return false
	}
	return a.Index == b.Index
}

/*----- serialization for tx outputs ------- */

// outPointToBytes turns an outpoint into 36 bytes.
func OutPointToBytes(op wire.OutPoint) (b [36]byte) {
	var buf bytes.Buffer
	_, err := buf.Write(op.Hash.CloneBytes())
	if err != nil {
		return
	}
	// write 4 byte outpoint index within the tx to spend
	err = binary.Write(&buf, binary.BigEndian, op.Index)
	if err != nil {
		return
	}
	copy(b[:], buf.Bytes())

	return
}

// OutPointFromBytes gives you an outpoint from 36 bytes.
// since 36 is enforced, it doesn't error
func OutPointFromBytes(b [36]byte) *wire.OutPoint {
	op := new(wire.OutPoint)
	_ = op.Hash.SetBytes(b[:32])
	op.Index = BtU32(b[32:])
	return op
}

// P2WSHify takes a script and turns it into a 34 byte long P2WSH PkScript
func P2WSHify(scriptBytes []byte) []byte {
	bldr := txscript.NewScriptBuilder()
	bldr.AddOp(txscript.OP_0)
	wsh := fastsha256.Sum256(scriptBytes)
	bldr.AddData(wsh[:])
	b, _ := bldr.Script() // ignore script errors
	return b
}

func DirectWPKHScript(pub [33]byte) []byte {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_0).AddData(btcutil.Hash160(pub[:]))
	b, _ := builder.Script()
	return b
}

func DirectWPKHScriptFromPKH(pkh [20]byte) []byte {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_0).AddData(pkh[:])
	b, _ := builder.Script()
	return b
}

// KeyHashFromPkScript extracts the 20 or 32 byte hash from a txout PkScript
func KeyHashFromPkScript(pkscript []byte) []byte {
	// match p2pkh
	if len(pkscript) == 25 && pkscript[0] == 0x76 && pkscript[1] == 0xa9 &&
		pkscript[2] == 0x14 && pkscript[23] == 0x88 && pkscript[24] == 0xac {
		return pkscript[3:23]
	}

	// match p2wpkh
	if len(pkscript) == 22 && pkscript[0] == 0x00 && pkscript[1] == 0x14 {
		return pkscript[2:]
	}

	// match p2wsh
	if len(pkscript) == 34 && pkscript[0] == 0x00 && pkscript[1] == 0x20 {
		return pkscript[2:]
	}

	return nil
}

// TxToString prints out some info about a transaction. for testing / debugging
func TxToString(tx *wire.MsgTx) string {
	utx := btcutil.NewTx(tx)
	str := fmt.Sprintf("size %d vsize %d wsize %d locktime %d wit: %t txid %s\n",
		tx.SerializeSizeStripped(), blockchain.GetTxVirtualSize(utx),
		tx.SerializeSize(), tx.LockTime, tx.HasWitness(), tx.TxHash().String())
	for i, in := range tx.TxIn {
		str += fmt.Sprintf("Input %d spends %s seq %d\n",
			i, in.PreviousOutPoint.String(), in.Sequence)
		str += fmt.Sprintf("\tSigScript: %x\n", in.SignatureScript)
		for j, wit := range in.Witness {
			str += fmt.Sprintf("\twitness %d: %x\n", j, wit)
		}
	}
	for i, out := range tx.TxOut {
		if out != nil {
			str += fmt.Sprintf("output %d script: %x amt: %d\n",
				i, out.PkScript, out.Value)
		} else {
			str += fmt.Sprintf("output %d nil (WARNING)\n", i)
		}
	}
	return str
}
