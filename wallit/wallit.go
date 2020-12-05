package wallit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/blockchain"
	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/uspv"
	"github.com/mit-dci/lit/wire"
)

/* implements the uwallet interface.  Which is much too large!
 ... but for now works this way, with like 17 functions, bleh.

 // Ask for a pubkey based on a bip32 path
 GetPub(k portxo.KeyGen) *btcec.PublicKey

 // Have GetPriv for now.  Maybe later get rid of this and have
 // the underlying wallet sign?
 GetPriv(k portxo.KeyGen) *btcec.PrivateKey

 // Send a tx out to the network.  Maybe could replace?  Maybe not.
 // Needed for channel break / cooperative close.  Maybe grabs.

 // export the chainhook that the UWallet uses, for pushTx and fullblock
 ExportHook() uspv.ChainHook

 PushTx(tx *wire.MsgTx) error

 // ExportUtxo gives a utxo to the underlying wallet; that wallet saves it
 // and can spend it later.  Doesn't return errors; error will exist only in
 // base wallet.
 ExportUtxo(txo *portxo.PorTxo)

 // MaybeSend makes an unsigned tx, populated with inputs and outputs.
 // The specified txouts are in there somewhere.
 // Only segwit txins are in the generated tx. (txid won't change)
 // There's probably an extra change txout in there which is OK.
 // The inputs are "frozen" until ReallySend / NahDontSend / program restart.
 // Retruns the txid, and then the txout indexes of the specified txos.
 // The outpoints returned will all have the same hash (txid)
 // So if you (as usual) just give one txo, you basically get back an outpoint.
 MaybeSend(txos []*wire.TxOut, onlyWit bool) ([]*wire.OutPoint, error)

 // ReallySend really sends the transaction specified previously in MaybeSend.
 // Underlying wallet does all needed signing.
 // Once you call ReallySend, the outpoint is tracked and responses are
 // sent through LetMeKnow
 ReallySend(txid *chainhash.Hash) error

 // NahDontSend cancels the MaybeSend transaction.
 NahDontSend(txid *chainhash.Hash) error

 // Return a new address
 NewAdr() ([20]byte, error)

 // Dump all the utxos in the sub wallet
 UtxoDump() ([]*portxo.PorTxo, error)

 // Dump all the addresses the sub wallet is watching
 AdrDump() ([][20]byte, error)

 // Return current height the wallet is synced to
 CurrentHeight() int32

 // WatchThis tells the basewallet to watch an outpoint
 WatchThis(wire.OutPoint) error

 // LetMeKnow opens the chan where OutPointEvent flows from the underlying
 // wallet up to the LN module.
 LetMeKnow() chan lnutil.OutPointEvent

 // Ask for network parameters
 Params() *coinparam.Params

 // Get current fee rate.
 Fee() int64

 // Set fee rate
 SetFee(int64) int64

 // ===== TESTING / SPAMMING ONLY, these funcs will not be in the real interface
 // Sweep sends lots of txs (uint32 of them) to the specified address.
 Sweep([]byte, uint32) ([]*chainhash.Hash, error)

 */

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

	// HeightEventChan sends block height changes to the LN wallet.
	// Gets initialized and activates when called by qln
	HeightEventChan chan lnutil.HeightEvent

	// Params live here...
	Param *coinparam.Params // network parameters (testnet3, segnet, etc)

	// Hook is the connection to a blockchain.
	// imports the uspv interface.  Could put that somewhere else.
	// like an interfaces library, ... lnutil?
	Hook uspv.ChainHook

	// current fee per byte
	FeeRate int64

	// From here, comes everything. It's a secret to everybody.
	rootPrivKey *hdkeychain.ExtendedKey
}

type FrozenTx struct {
	Ins       []*portxo.PorTxo `json:"ins"`
	Outs      []*wire.TxOut    `json:"outs"`
	ChangeOut *wire.TxOut      `json:"changeout"`
	Nlock     uint32           `json:"nlocktime"`
	Txid      chainhash.Hash   `json:"txid"`
}

// Stxo is a utxo that has moved on.
type Stxo struct {
	PorTxo      portxo.PorTxo  `json:"txo"`    // when it used to be a utxo
	SpendHeight int32          `json:"height"` // height at which it met its demise
	SpendTxid   chainhash.Hash `json:"txid"`   // the tx that consumed it
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
bytelength   desc   at offset

53			portxo		0
4			sheight		53
32			stxid		57

end len 	89
*/

// ToBytes turns an Stxo into some bytes.
// prevUtxo serialization, then spendheight [4], spendtxid [32]
func (s *Stxo) ToBytes() ([]byte, error) {
	var buf bytes.Buffer

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

	// write 4 byte height where the txo was spent
	err = binary.Write(&buf, binary.BigEndian, s.SpendHeight)
	if err != nil {
		return nil, err
	}
	// write 32 byte txid of the spending transaction
	_, err = buf.Write(s.SpendTxid.CloneBytes())
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// StxoFromBytes turns bytes into a Stxo.
// it's a portxo with a spendHeight and spendTxid at the end.
func StxoFromBytes(b []byte) (Stxo, error) {
	var s Stxo

	l := len(b)
	if l < 96 {
		return s, fmt.Errorf("Got %d bytes for stxo, expect a bunch", len(b))
	}

	// last 36 bytes are height & spend txid.
	u, err := portxo.PorTxoFromBytes(b[:l-36])
	if err != nil {
		logging.Errorf(" eof? ")
		return s, err
	}

	buf := bytes.NewBuffer(b[l-36:])
	// read 4 byte spend height
	err = binary.Read(buf, binary.BigEndian, &s.SpendHeight)
	if err != nil {
		return s, err
	}
	// read 32 byte txid
	err = s.SpendTxid.SetBytes(buf.Next(32))
	if err != nil {
		return s, err
	}

	s.PorTxo = *u // assign the utxo

	return s, nil
}
