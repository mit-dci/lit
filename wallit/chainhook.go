package wallit

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
)

// ChainHook is an interface which provides access to a blockchain for the
// wallit.  Right now the USPV package conforms to this interface, but it
// should also be possible to plug in things like fullnode-rpc clients, and
// (yuck) trusted block explorers.

/*  The model here is that the wallit has lots of state on disk about
what it's seen.  The ChainHook is ephemeral, and has some state in RAM but
shouldn't have to keep track of too much; or at least what it keeps track of
is internal and not exported out to the wallit (eg a fullnode keeps track
of a lot, and SPV node, somewhat less, and a block explorer shim basically nothing)
*/

type ChainHook interface {

	// Start turns on the ChainHook.  Later on, pass more parameters here.
	// Also gets the txChan where txs come in from the ChainHook to the wallit.
	// The TxChannel should never give txs that the wallit doesn't care about.  In the
	// case of bloom fitlers, false positives should be handled and stopped at the
	// ChainHook layer; wallit should not have to handle ingesting irrelevant txs.
	// You get back an error and 2 channels: one for txs with height attached, and
	// one with block heights.  Subject to change; maybe this is redundant.
	Start(height int32, host, path string, params *chaincfg.Params) (
		chan lnutil.TxAndHeight, chan int32, error)

	// The Register functions send information to the ChainHook about what txs to
	// return.  Txs matching either the addresses or outpoints will be returned
	// via the TxChannel

	// RegisterAddress tells the ChainHook about an address of interest.
	// Give it an array; Currently needs to be 20 bytes.  Only p2pkh / p2wpkh
	// are supported.
	// Later could add a separate function for script hashes (20/32)
	RegisterAddress(address [20]byte) error

	// RegisterOutPoint tells the ChainHook about an outpoint of interest.
	RegisterOutPoint(wire.OutPoint) error

	// SetHeight sets the height ChainHook needs to look above.
	// Returns a channel which tells the wallit what height the ChainHook has
	// sync'd up to.  This chan should push int32s *after* the TxAndHeights
	// have come in; so getting a "54" here means block 54 is done and fully parsed.
	// Removed, put into Start().
	//	SetHeight(startHeight int32) chan int32

	// PushTx sends a tx out to the network via the ChainHook.
	// Note that this does NOT register anything in the tx, so by just using this,
	// nothing will come back about confirmation.  It WILL come back with errors
	// though, so this takes some time.
	PushTx(tx *wire.MsgTx) error

	// Request all incoming blocks over this channel.  If RawBlocks isn't called,
	// then the undelying hook package doesn't need to get full blocks.
	// Currently you always call it with uspv...
	RawBlocks() chan *wire.MsgBlock
	// TODO -- reorgs.  Oh and doublespends and stuff.
}
