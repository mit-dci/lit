package wallit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

// The Wallit is lit's main wallet struct.  It's got the root key, the dbs, and
// contains the SPVhooks into the network.
type Wallit struct {
	// could get rid of adr slice, it's just an in-ram cache...
	StateDB *bolt.DB // place to write all this down

	// Set of frozen utxos not to use... they point to the tx using em
	FreezeSet   map[wire.OutPoint]*FrozenTx
	FreezeMutex sync.Mutex

	// OPEventChan sends events to the LN wallet.
	// Gets initialized and activates when called by qln
	OPEventChan chan lnutil.OutPointEvent

	// Params live here...
	Param *chaincfg.Params // network parameters (testnet3, segnet, etc)

	// Hook is the connection to a blockchain.
	Hook ChainHook

	// From here, comes everything. It's a secret to everybody.
	rootPrivKey *hdkeychain.ExtendedKey
}

type FrozenTx struct {
	Ins       []*portxo.PorTxo
	Outs      []*wire.TxOut
	ChangeOut *wire.TxOut
	Txid      chainhash.Hash
}

// Stxo is a utxo that has moved on.
type Stxo struct {
	portxo.PorTxo                // when it used to be a utxo
	SpendHeight   int32          // height at which it met its demise
	SpendTxid     chainhash.Hash // the tx that consumed it
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

/*----- serialization for stxos ------- */
/* Stxo serialization:
byte length   desc   at offset

53	utxo		0
4	sheight	53
32	stxid	57

end len 	89
*/

// ToBytes turns an Stxo into some bytes.
// prevUtxo serialization, then spendheight [4], spendtxid [32]
func (s *Stxo) ToBytes() ([]byte, error) {
	var buf bytes.Buffer

	// write 4 byte height where the txo was spent
	err := binary.Write(&buf, binary.BigEndian, s.SpendHeight)
	if err != nil {
		return nil, err
	}
	// write 32 byte txid of the spending transaction
	_, err = buf.Write(s.SpendTxid.CloneBytes())
	if err != nil {
		return nil, err
	}
	// serialize the utxo part
	uBytes, err := s.PorTxo.Bytes()
	if err != nil {
		return nil, err
	}
	// write that into the buffer
	_, err = buf.Write(uBytes)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// StxoFromBytes turns bytes into a Stxo.
// first 36 bytes are how it's spent, after that is portxo
func StxoFromBytes(b []byte) (Stxo, error) {
	var s Stxo
	if len(b) < 96 {
		return s, fmt.Errorf("Got %d bytes for stxo, expect a bunch", len(b))
	}
	buf := bytes.NewBuffer(b)
	// read 4 byte spend height
	err := binary.Read(buf, binary.BigEndian, &s.SpendHeight)
	if err != nil {
		return s, err
	}
	// read 32 byte txid
	err = s.SpendTxid.SetBytes(buf.Next(32))
	if err != nil {
		return s, err
	}

	u, err := portxo.PorTxoFromBytes(buf.Bytes())
	if err != nil {
		log.Printf(" eof? ")
		return s, err
	}
	s.PorTxo = *u // assign the utxo

	return s, nil
}
