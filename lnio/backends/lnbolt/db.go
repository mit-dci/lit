package lnbolt

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/lnio"
	"path"
)

// LitBoltDB uses Bolt as a storage backend.
type LitBoltDB struct {
	open     bool
	walletdb *bolt.DB
	peerdb   *bolt.DB
	chandb   *bolt.DB
}

// Open .
func (db LitBoltDB) Open(dbpath string) error {

	// Sanity check.
	if db.open {
		return fmt.Errorf("tried to open an open BoltDB database")
	}

	// Figure out file paths.
	wpath := path.Join(dbpath, "wallet.db")
	ppath := path.Join(dbpath, "peer.db")
	cpath := path.Join(dbpath, "channels.db")

	var err error

	walletdb, err := bolt.Open(wpath, 0600, nil)
	if err != nil {
		return err
	}

	peerdb, err := bolt.Open(ppath, 0600, nil)
	if err != nil {
		walletdb.Close()
		return err
	}

	chandb, err := bolt.Open(cpath, 0600, nil)
	if err != nil {
		walletdb.Close()
		peerdb.Close()
		return err
	}

	// Actually set the new databases in ourselves.
	db.walletdb = walletdb
	db.peerdb = peerdb
	db.chandb = chandb
	db.open = true

	return nil

}

// IsSingleFile .
func (LitBoltDB) IsSingleFile() bool {
	return false
}

// Close .
func (db LitBoltDB) Close() error {

	// Sanity check.
	if !db.open {
		return fmt.Errorf("tried to close a closed BoltDB database")
	}

	var err error

	err = db.walletdb.Close()
	if err != nil {
		return err
	}

	err = db.peerdb.Close()
	if err != nil {
		return err
	}

	err = db.chandb.Close()
	if err != nil {
		return err
	}

	db.open = false
	return nil

}

// GetWalletDB .
func (db LitBoltDB) GetWalletDB(cointype uint32) lnio.LitWalletStorage {
	return nil
}

// GetPeerDB .
func (db LitBoltDB) GetPeerDB() lnio.LitPeerStorage {
	w := peerboltdb{db.peerdb}
	return w
}

// GetChannelDB .
func (db LitBoltDB) GetChannelDB() lnio.LitChannelStorage {
	return nil
}
