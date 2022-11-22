package portxo

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/mit-dci/lit/wire"
)

type TxoMode uint8

/* PorTxo specify a utxo, and all the information needed to spend it.
The first 3 fields (Op, Amt, Mode) are required in all cases.
If KeyGen.Depth != 0, that means no key path is supplied, and PrivKey
is probably empty / ignored.  Having both a KeyGen and a PrivKey is redunant.
Having neither KeyGen nor PrivKey means there's no private key, and no
indication of how to get it; in that case get the private key from somewhere else.

If BOTH KeyGen AND PrivKey are filled in, add em up!  Add the two private keys,
modulo the curve order.

PkScript can also be left empty depending on the mode.  Basically only script-hash
modes need it, as the previous pkscript can be generated

PreSigStack are data pushes that happen before the signature is pushed.

I was thinking of putting PostSigStack as well but I think it makes more sense to
always have the data before the sig, since there's pretty much only CHECKSIGVERIFY
now and you might as well put the sig check last.

Also makes sense that with a MAST or pay-to-script-merkle-root type of structure,
you'd want a bunch stuff before the sig.

*/
type PorTxo struct {
	// got rid of NetID.  If you want to specify different networks / coins,
	// use KeyGen.Step[1], that's what it's for.
	// Heck, set KeyGen.Depth to 0 and still use Step[1] as the network / coin...
	//	NetID  byte          // indicates what network / coin utxo is in
	Op     wire.OutPoint `json:"outpoint"` // unique outpoint
	Value  int64         `json:"val"`      // higher is better
	Height int32         `json:"height"`   // block height of utxo (not needed? nice to know?)
	Seq    uint32        `json:"seq"`      // used for relative timelock
	Mode   TxoMode       `json:"mode"`     // what kind of output

	KeyGen `json:"keygen"`

	PkScript []byte `json:"privkeyscript"` // if empty, try to generate based on mode and priv key

	PreSigStack [][]byte `json:"stack"` // items to push before the sig
}

// Constants defining txo modes
const (
	// Flags which combined can turn into full utxo modes
	FlagTxoPubKeyHash   TxoMode = 0x01
	FlagTxoScript       TxoMode = 0x02
	FlagTxoWitness      TxoMode = 0x04
	FlagTxoCompressed   TxoMode = 0x08
	FlagTxoUncompressed TxoMode = 0x10

	// fully specified tx output modes
	// raw pubkey outputs (old school)
	TxoP2PKUncomp = FlagTxoUncompressed
	TxoP2PKComp   = FlagTxoCompressed

	// pub key hash outputs, standard p2pkh (common)
	TxoP2PKHUncomp = FlagTxoPubKeyHash | FlagTxoUncompressed
	TxoP2PKHComp   = FlagTxoCompressed | FlagTxoPubKeyHash

	// script hash
	TxoP2SHUncomp = FlagTxoScript | FlagTxoUncompressed
	TxoP2SHComp   = FlagTxoScript | FlagTxoCompressed

	// witness p2wpkh modes
	TxoP2WPKHUncomp = FlagTxoWitness | FlagTxoPubKeyHash | FlagTxoUncompressed
	TxoP2WPKHComp   = FlagTxoWitness | FlagTxoPubKeyHash | FlagTxoCompressed

	// witness script hash
	TxoP2WSHUncomp = FlagTxoWitness | FlagTxoScript | FlagTxoUncompressed
	TxoP2WSHComp   = FlagTxoWitness | FlagTxoScript | FlagTxoCompressed

	// unknown
	TxoUnknownMode = 0x80
)

var modeStrings = map[TxoMode]string{
	TxoP2PKUncomp: "raw pubkey uncompressed",
	TxoP2PKComp:   "raw pubkey compressed",

	TxoP2PKHUncomp: "pubkey hash uncompressed",
	TxoP2PKHComp:   "pubkey hash compressed",

	TxoP2SHUncomp: "script hash uncompressed",
	TxoP2SHComp:   "script hash compressed",

	TxoP2WPKHUncomp: "witness pubkey hash uncompressed",
	TxoP2WPKHComp:   "witness pubkey hash compressed",

	TxoP2WSHUncomp: "witness script hash uncompressed",
	TxoP2WSHComp:   "witness script hash compressed",
}

var (
	KeyGenForImports = KeyGen{
		Depth: 5,
		Step:  [5]uint32{0xfee154fe, 0, 0, 0, 0},
	}
	KeyGenEmpty = KeyGen{
		Depth: 0,
		Step:  [5]uint32{0, 0, 0, 0, 0},
	}
)

// String returns the InvType in human-readable form.
func (m TxoMode) String() string {
	s, ok := modeStrings[m]
	if ok {
		return s
	}
	return fmt.Sprintf("unknown TxoMode %x", uint8(m))
}

// Compare deep-compares two portable utxos, returning true if they're the same
func (u *PorTxo) Equal(z *PorTxo) bool {
	if u == nil || z == nil {
		return false
	}

	if !u.Op.Hash.IsEqual(&z.Op.Hash) {
		return false
	}
	if u.Op.Index != z.Op.Index {
		return false
	}
	if u.Value != z.Value || u.Seq != z.Seq || u.Mode != z.Mode || u.Height != z.Height {
		return false
	}
	if u.KeyGen.PrivKey != z.KeyGen.PrivKey {
		return false
	}
	if !bytes.Equal(u.KeyGen.Bytes(), z.KeyGen.Bytes()) {
		return false
	}
	if !bytes.Equal(u.PkScript, z.PkScript) {
		return false
	}

	// compare pre sig stack lengths
	if len(u.PreSigStack) != len(z.PreSigStack) {
		return false
	}

	// if we're here, lengths for both are the same.  Iterate and compare stacks
	for i, _ := range u.PreSigStack {
		if !bytes.Equal(u.PreSigStack[i], z.PreSigStack[i]) {
			return false
		}
	}

	return true
}

func (u *PorTxo) String() string {
	var s string
	var empty [32]byte
	if u == nil {
		return "nil utxo"
	}
	s = u.Op.String()
	s += fmt.Sprintf("\n\ta:%d h:%d seq:%d %s\n",
		u.Value, u.Height, u.Seq, u.Mode.String())

	if u.KeyGen.PrivKey == empty {
		s += fmt.Sprintf("\tprivate key not available (zero)\n")
	} else {
		s += fmt.Sprintf("\tprivate key available (non-zero)\n")
	}
	if u.KeyGen.Depth == 0 || u.KeyGen.Depth > 5 {
		s += fmt.Sprintf("\tno key derivation path\n")
	} else {
		s += fmt.Sprintf("%s\n", u.KeyGen.String())
	}
	s += fmt.Sprintf("\tPkScript (len %d): %x\n", len(u.PkScript), u.PkScript)

	// list pre-sig elements
	s += fmt.Sprintf("\t%d pre-sig elements:", len(u.PreSigStack))
	for _, e := range u.PreSigStack {
		s += fmt.Sprintf(" [%x]", e)
	}
	s += fmt.Sprintf("\n")

	return s
}

/* serialized (im/ex)Portable Utxos are 106 up to 357 bytes.
Op 36
Amt 8
Height 4
Seq 4
Mode 1

Keygen 53

PreStackNum (1 byte)
	PreStackItemLen (1 byte)
	PreStackItem (max 255 bytes each)
PostStackNum (1 byte)
	PostStackItemLen (1 byte)
	PostStackItem (max 255 bytes each)
PkScriptLen (1 byte)
	PkScript (max 255 bytes)


*/

func PorTxoFromBytes(b []byte) (*PorTxo, error) {
	// should be max 1KiB for now
	if len(b) < 106 || len(b) > 1024 {
		return nil, fmt.Errorf("%d bytes, need 106-1024", len(b))
	}

	buf := bytes.NewBuffer(b)

	var u PorTxo
	var err error

	err = u.Op.Hash.SetBytes(buf.Next(32))
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.BigEndian, &u.Op.Index)
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.BigEndian, &u.Value)
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.BigEndian, &u.Height)
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.BigEndian, &u.Seq)
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.BigEndian, &u.Mode)
	if err != nil {
		return nil, err
	}

	var kgenarr [53]byte
	copy(kgenarr[:], buf.Next(53))
	u.KeyGen = KeyGenFromBytes(kgenarr)

	// get PkScript length byte
	PkScriptLen, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	// make PkScript the right size to write into
	u.PkScript = make([]byte, PkScriptLen)

	// write from buffer into PkScript
	_, err = buf.Read(u.PkScript)
	if err != nil {
		return nil, err
	}

	// get number of pre-sig stack items
	PreSigStackNumItems, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	u.PreSigStack = make([][]byte, PreSigStackNumItems)
	// iterate through reading each presigStack item
	for i, _ := range u.PreSigStack {
		// read length of this element
		elementLength, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		// size element slice
		u.PreSigStack[i] = make([]byte, elementLength)
		// copy element data
		_, err = buf.Read(u.PreSigStack[i])
		if err != nil {
			return nil, err
		}
	}

	return &u, nil
}

// Spendable takes in the current block height, and returns whether
// this utxo is spendable now or not.  It checks height (unconfirmed
func (u *PorTxo) Mature(curHeight int32) bool {
	// Nil portxo should be an error but just return false
	if u == nil {
		return false
	}
	//if sequence is nonzero, it's CSV locked, so check if it's either
	// unconfirmed or confirmed at a height that's too shallow
	if u.Seq > 1 &&
		(u.Height < 100 || u.Height+int32(u.Seq) > curHeight) {
		return false
	}
	return true
}

// EstSize returns an estimated vsize in bytes for using this portxo as
// an input.  Might not be perfectly accurate.  Also maybe should switch
// from vsize to weight..?  everything is vsize right now though.
func (u *PorTxo) EstSize() int64 {
	// Txins by mode:
	// P2 PKH is op,seq (40) + pub(33) + sig(71) = 144
	// P2 WPKH is op,seq(40) + [(33+71 / 4) = 26] = 66
	// P2 WSH is op,seq(40) + [75(script) + 71]/4 (36) = 76
	switch u.Mode {
	case TxoP2PKHComp: // non witness is about 150 bytes
		return 144
	case TxoP2WPKHComp: // witness mode is around 66 vsize
		return 66
	case TxoP2WSHComp:
		return 76
	}
	return 150 // guess that unknown is 150 bytes
}

func (u *PorTxo) Bytes() ([]byte, error) {
	if u == nil {
		return nil, fmt.Errorf("Can't serialize nil Utxo")
	}

	var buf bytes.Buffer

	//	_, err := buf.Write(u.Op.Hash.CloneBytes())
	_, err := buf.Write(u.Op.Hash.CloneBytes())
	if err != nil {
		return nil, err
	}

	err = binary.Write(&buf, binary.BigEndian, u.Op.Index)
	if err != nil {
		return nil, err
	}

	err = binary.Write(&buf, binary.BigEndian, u.Value)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&buf, binary.BigEndian, u.Height)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&buf, binary.BigEndian, u.Seq)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&buf, binary.BigEndian, u.Mode) // mode
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(u.KeyGen.Bytes()) // keypath
	if err != nil {
		return nil, err
	}

	// check pkScript length
	if len(u.PkScript) > 255 {
		return nil, fmt.Errorf("PkScript too long (255 byte max)")
	}
	// write length of pkScript
	err = buf.WriteByte(uint8(len(u.PkScript)))
	if err != nil {
		return nil, err
	}
	// write PkScript
	_, err = buf.Write(u.PkScript)
	if err != nil {
		return nil, err
	}

	// check Pre-sig stack number of elements
	if len(u.PreSigStack) > 255 {
		return nil, fmt.Errorf("Too many PreSigStack items (255 items max)")
	}
	// write number of PreSigStack items
	err = buf.WriteByte(uint8(len(u.PreSigStack)))
	if err != nil {
		return nil, err
	}
	// iterate through PreSigStack items and write each
	for i, element := range u.PreSigStack {
		// check element length
		if len(element) > 255 {
			return nil, fmt.Errorf("PreSigStack item %d %d bytes (255 max)",
				i, len(element))
		}
		// write length of element
		err = buf.WriteByte(uint8(len(element)))
		if err != nil {
			return nil, err
		}
		// write element itself
		_, err = buf.Write(element)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
