package lncore

// Txid represents a transaction on the blockchain.
type Txid struct {
	Cointype int32
	Txhash   []byte
}

// Utxo is an unspent transaction output that we could be able to spend.
type Utxo struct {
	Txid
}
