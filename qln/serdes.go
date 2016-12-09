package qln

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

/*----- serialization for StatCom ------- */
/*
bytes   desc   ends at
8	StateIdx		8
8	MyAmt		16
4	Delta		20
33	MyRev		53
33	MyPrevRev	86
64	Sig			150


note that sigs are truncated and don't have the sighash type byte at the end.

their rev hash can be derived from the elkrem sender
and the stateidx.  hash160(elkremsend(sIdx)[:16])

*/

// ToBytes turns a StatCom into 158ish bytes
func (s *StatCom) ToBytes() ([]byte, error) {
	var buf bytes.Buffer
	var err error

	// write 8 byte state index
	err = binary.Write(&buf, binary.BigEndian, s.StateIdx)
	if err != nil {
		return nil, err
	}
	// write 8 byte watch up to state index
	err = binary.Write(&buf, binary.BigEndian, s.WatchUpTo)
	if err != nil {
		return nil, err
	}

	// write 8 byte amount of my allocation in the channel
	err = binary.Write(&buf, binary.BigEndian, s.MyAmt)
	if err != nil {
		return nil, err
	}
	// write 4 byte delta.  At steady state it's 0.
	err = binary.Write(&buf, binary.BigEndian, s.Delta)
	if err != nil {
		return nil, err
	}
	// write 33 byte my elk point R
	_, err = buf.Write(s.ElkPoint[:])
	if err != nil {
		return nil, err
	}
	// write 33 byte Next elk point
	_, err = buf.Write(s.NextElkPoint[:])
	if err != nil {
		return nil, err
	}

	// write their sig
	_, err = buf.Write(s.sig[:])
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// StatComFromBytes turns 158 bytes into a StatCom
func StatComFromBytes(b []byte) (*StatCom, error) {
	var s StatCom
	if len(b) < 158 || len(b) > 158 {
		return nil, fmt.Errorf("StatComFromBytes got %d bytes, expect 158",
			len(b))
	}
	buf := bytes.NewBuffer(b)
	// read 8 byte state index
	err := binary.Read(buf, binary.BigEndian, &s.StateIdx)
	if err != nil {
		return nil, err
	}
	// read 8 byte WatchUpTo index
	err = binary.Read(buf, binary.BigEndian, &s.WatchUpTo)
	if err != nil {
		return nil, err
	}

	// read 8 byte amount of my allocation in the channel
	err = binary.Read(buf, binary.BigEndian, &s.MyAmt)
	if err != nil {
		return nil, err
	}
	// read 4 byte delta.
	err = binary.Read(buf, binary.BigEndian, &s.Delta)
	if err != nil {
		return nil, err
	}
	// read 33 byte elk point
	copy(s.ElkPoint[:], buf.Next(33))
	// read 33 byte next elk point
	copy(s.NextElkPoint[:], buf.Next(33))

	// the rest is their sig
	copy(s.sig[:], buf.Next(64))

	return &s, nil
}

/*----- serialization for QChannels ------- */

/* Qchan serialization:
bytes   desc   at offset

60	utxo		0
33	nonce	60
33	thrref	93

length 126

peeridx is inferred from position in db.
*/
//TODO !!! don't store the outpoint!  it's redundant!!!!!
// it's just a nonce and a refund, that's it! 40 bytes!

func (q *Qchan) ToBytes() ([]byte, error) {
	var buf bytes.Buffer

	// write their channel pubkey
	_, err := buf.Write(q.TheirPub[:])
	if err != nil {
		return nil, err
	}

	// write their refund pubkey
	_, err = buf.Write(q.TheirRefundPub[:])
	if err != nil {
		return nil, err
	}
	// write their HAKD base
	_, err = buf.Write(q.TheirHAKDBase[:])
	if err != nil {
		return nil, err
	}

	// then serialize the utxo part
	uBytes, err := q.PorTxo.Bytes()
	if err != nil {
		return nil, err
	}
	// and write that into the buffer
	_, err = buf.Write(uBytes)
	if err != nil {
		return nil, err
	}

	// done
	return buf.Bytes(), nil
}

// QchanFromBytes turns bytes into a Qchan.
// the first 99 bytes are the 3 pubkeys: channel, refund, HAKD base
// then 8 bytes for the DH mask
// the rest is the utxo
func QchanFromBytes(b []byte) (Qchan, error) {
	var q Qchan

	if len(b) < 205 {
		return q, fmt.Errorf("Got %d bytes for qchan, expect 205+", len(b))
	}

	copy(q.TheirPub[:], b[:33])
	copy(q.TheirRefundPub[:], b[33:66])
	copy(q.TheirHAKDBase[:], b[66:99])
	u, err := portxo.PorTxoFromBytes(b[99:])
	if err != nil {
		return q, err
	}

	q.PorTxo = *u // assign the utxo
	// hard-coded.  save this soon, it's easy
	q.Delay = 5

	return q, nil
}

/*----- serialization for CloseTXOs -------

  serialization:
closetxid	32
closeheight	4

only closeTxid needed, I think

*/

func (c *QCloseData) ToBytes() ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("nil qclose")
	}
	b := make([]byte, 36)
	copy(b[:32], c.CloseTxid.CloneBytes())
	copy(b[32:], lnutil.I32tB(c.CloseHeight))
	return b, nil
}

// QCloseFromBytes deserializes a Qclose.  Note that a nil slice
// gives an empty / non closed qclose.
func QCloseFromBytes(b []byte) (QCloseData, error) {
	var c QCloseData
	if len(b) == 0 { // empty is OK
		return c, nil

	}
	if len(b) < 36 {
		return c, fmt.Errorf("close data %d bytes, expect 36", len(b))
	}
	var empty chainhash.Hash
	c.CloseTxid.SetBytes(b[:32])
	if !c.CloseTxid.IsEqual(&empty) {
		c.Closed = true
	}
	c.CloseHeight = lnutil.BtI32(b[32:36])

	return c, nil
}
