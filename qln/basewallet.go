package qln

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

// The UWallet interface are the functions needed to work with the LnNode
// Verbs are from the perspective of the LnNode, not the underlying wallet
type UWallet interface {
	// Ask for a pubkey based on a bip32 path
	GetPub(k portxo.KeyGen) *btcec.PublicKey

	// Have GetPriv for now.  Maybe later get rid of this and have
	// the underlying wallet sign?
	GetPriv(k portxo.KeyGen) *btcec.PrivateKey

	// Send a tx out to the network.  Maybe could replace?  Maybe not.
	// Needed for channel break / cooperative close.  Maybe grabs.
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
	NewAdr() btcutil.Address

	// Dump all the utxos in the sub wallet
	UtxoDump() ([]*portxo.PorTxo, error)

	// Dump all the addresses the sub wallet is watching
	AdrDump() ([]btcutil.Address, error)

	// Return current height the wallet is synced to
	CurrentHeight() int32

	// This is redundand... just use UtxoDump and figure it out yourself.
	// Feels like helper functions shouldn't be in the interface.
	// how much utxo the wallet has -- only confirmed segwit outputs
	//	HowMuchWitConf() int64

	// How much utxo the sub wallet has, including non-segwit, unconfirmed, immature
	//	HowMuchTotal() int64

	// WatchThis tells the basewallet to watch an outpoint
	WatchThis(wire.OutPoint) error

	// LetMeKnow opens the chan where OutPointEvent flows from the underlying
	// wallet up to the LN module.
	LetMeKnow() chan lnutil.OutPointEvent

	// raw blocks coming in for the watchtower to check
	BlockMonitor() chan *wire.MsgBlock

	// Ask for network parameters
	Params() *chaincfg.Params

	// ===== TESTING / SPAMMING ONLY, these funcs will not be in the real interface
	// Sweep sends lots of txs (uint32 of them) to the specified address.
	Sweep([]byte, uint32) ([]*chainhash.Hash, error)
}

// GetUsePub gets a pubkey from the base wallet, but first modifies
// the "use" step
func (nd *LitNode) GetUsePub(k portxo.KeyGen, use uint32) (pubArr [33]byte) {
	k.Step[2] = use
	pub := nd.SubWallet.GetPub(k)
	copy(pubArr[:], pub.SerializeCompressed())
	return
}

// Get rid of this function soon and replace with signing function
func (nd *LitNode) GetPriv(k portxo.KeyGen) *btcec.PrivateKey {
	return nd.SubWallet.GetPriv(k)
}

// GetElkremRoot returns the Elkrem root for a given key path
// gets the use-pub for elkrems and hashes it.
// A little weird because it's a "pub" key you shouldn't reveal.
// either do this or export privkeys... or signing empty txs or something.
func (nd *LitNode) GetElkremRoot(k portxo.KeyGen) chainhash.Hash {
	pubArr := nd.GetUsePub(k, UseChannelElkrem)
	return chainhash.DoubleHashH(pubArr[:])
}
