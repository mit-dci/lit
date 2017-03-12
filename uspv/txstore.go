package uspv

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
	"github.com/btcsuite/btcutil/bloom"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

type TxStore struct {
	// could get rid of adr slice, it's just an in-ram cache...
	StateDB *bolt.DB // place to write all this down

	// Set of frozen utxos not to use... they point to the tx using em
	FreezeSet   map[wire.OutPoint]*FrozenTx
	FreezeMutex sync.Mutex

	// OPEventChan sends events to the LN wallet.
	// Gets initialized and activates when called by qln
	OPEventChan chan lnutil.OutPointEvent

	// Params live here... AND SCon
	Param *chaincfg.Params // network parameters (testnet3, segnet, etc)

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

func NewTxStore(rootkey *hdkeychain.ExtendedKey, p *chaincfg.Params) TxStore {
	var txs TxStore
	txs.rootPrivKey = rootkey
	txs.Param = p
	txs.FreezeSet = make(map[wire.OutPoint]*FrozenTx)
	return txs
}

// OKTxid assigns a height to a txid.  This means that
// the txid exists at that height, with whatever assurance (for height 0
// it's no assurance at all)
func (s *SPVCon) OKTxid(txid *chainhash.Hash, height int32) error {
	if txid == nil {
		return fmt.Errorf("tried to add nil txid")
	}
	log.Printf("added %s to OKTxids at height %d\n", txid.String(), height)
	s.OKMutex.Lock()
	s.OKTxids[*txid] = height
	s.OKMutex.Unlock()
	return nil
}

// GimmeFilter ... or I'm gonna fade away
func (t *TxStore) GimmeFilter() (*bloom.Filter, error) {
	// get all address hash160s
	adr160s, err := t.GetAllAdr160s()
	if err != nil {
		return nil, err
	}

	// get all utxos to add outpoints to filter
	//	allUtxos, err := t.GetAllUtxos()
	//	if err != nil {
	//		return nil, err
	//	}
	// get all outpoints
	allWatchOP, err := t.GetAllOPs()
	if err != nil {
		return nil, err
	}

	filterElements := uint32(len(adr160s) + (len(allWatchOP) * 2))
	// *2 because I'm adding the op and the txid...?

	f := bloom.NewFilter(filterElements, 0, 0.000001, wire.BloomUpdateAll)

	// note there could be false positives since we're just looking
	// for the 20 byte PKH without the opcodes.
	for _, a160 := range adr160s { // add 20-byte pubkeyhash
		//		fmt.Printf("adding address hash %x\n", a160)
		f.Add(a160)
	}
	//	for _, u := range allUtxos {
	//		f.AddOutPoint(&u.Op)
	//	}

	// actually... we should monitor addresses, not txids, right?
	// or no...?
	for _, wop := range allWatchOP {
		//	 aha, add HASH here, not the outpoint! (txid of fund tx)
		f.AddHash(&wop.Hash)
		// also add outpoint...?  wouldn't the hash be enough?
		// not sure why I have to do both of these, but seems like close txs get
		// ignored without the outpoint, and fund txs get ignored without the
		// shahash. Might be that shahash operates differently (on txids, not txs)
		f.AddOutPoint(wop)
	}
	// still some problem with filter?  When they broadcast a close which doesn't
	// send any to us, sometimes we don't see it and think the channel is still open.
	// so not monitoring the channel outpoint properly?  here or in ingest()

	log.Printf("made %d element filter\n", filterElements)
	return f, nil
}

// GetDoubleSpends takes a transaction and compares it with
// all transactions in the db.  It returns a slice of all txids in the db
// which are double spent by the received tx.
func CheckDoubleSpends(
	argTx *wire.MsgTx, txs []*wire.MsgTx) ([]*chainhash.Hash, error) {

	var dubs []*chainhash.Hash // slice of all double-spent txs
	argTxid := argTx.TxHash()

	for _, compTx := range txs {
		compTxid := compTx.TxHash()
		// check if entire tx is dup
		if argTxid.IsEqual(&compTxid) {
			return nil, fmt.Errorf("tx %s is dup", argTxid.String())
		}
		// not dup, iterate through inputs of argTx
		for _, argIn := range argTx.TxIn {
			// iterate through inputs of compTx
			for _, compIn := range compTx.TxIn {
				if lnutil.OutPointsEqual(
					argIn.PreviousOutPoint, compIn.PreviousOutPoint) {
					// found double spend
					dubs = append(dubs, &compTxid)
					break // back to argIn loop
				}
			}
		}
	}
	return dubs, nil
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
