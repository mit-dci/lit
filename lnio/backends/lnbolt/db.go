package lnbolt

import (
	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/lnio"
	"path"
)

// LitBoltDB uses Bolt as a storage backend.
type LitBoltDB struct {
	// TODO Bolt DB implementation
}

func (db LitBoltDB) Open(path string) error {
	return nil
}

func (db LitBoltDB) Close() error {
	return nil
}

func (db LitBoltDB) GetWalletDB(cointype uint32) LitWalletStorage {
	return nil
}

func (db LitBoltDB) GetPeerDB() LitNetworkStorage {
	return nil
}

func (db LitBoltDB) GetChannelDB() LitChannelStorage {
	return nil
}
