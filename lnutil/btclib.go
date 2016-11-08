package lnutil

import (
	"bytes"
	"encoding/binary"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/fastsha256"
)

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
