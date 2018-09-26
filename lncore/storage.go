package lncore

// CoinSpecific is a meta-interface for coin-specific types.
type CoinSpecific interface {
	GetCoinTypeId() int32
	Bytes() []byte
}

// LitStorage is an abstract wrapper layer around an arbitrary database.
type LitStorage interface {
	Open(dbpath string) error
	IsSingleFile() bool
	Close() error

	GetWalletDB(uint32) LitWalletStorage
	GetPeerDB() LitPeerStorage
	GetChannelDB() LitChannelStorage

	Check() error
}

// LitWalletStorage is storage for wallet data.
type LitWalletStorage interface {
	CoinSpecific

	GetAddresses() ([]CoinAddress, error)

	GetUtxos() ([]Utxo, error)
	AddUtxo(Utxo) error
	RemoveUtxo(Utxo) error

	// TODO More
}

type CoinAddress struct {
	Cointype int32
	Addr     []byte
}

// AreCoinsCompatible checks to see if two coin-specific objects are for the same coin.
func AreCoinsCompatible(a, b CoinSpecific) bool {
	return a.GetCoinTypeId() == b.GetCoinTypeId()
}
