// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package database_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/chaincfg"
	"github.com/mit-dci/lit/btcutil/database"
	_ "github.com/mit-dci/lit/btcutil/database/ffldb"
	"github.com/mit-dci/lit/wire"
)

// This example demonstrates creating a new database.
func ExampleCreate() {
	// This example assumes the ffldb driver is imported.
	//
	// import (
	// 	"github.com/mit-dci/lit/btcutil/database"
	// 	_ "github.com/mit-dci/lit/btcutil/database/ffldb"
	// )

	// Create a database and schedule it to be closed and removed on exit.
	// Typically you wouldn't want to remove the database right away like
	// this, nor put it in the temp directory, but it's done here to ensure
	// the example cleans up after itself.
	dbPath := filepath.Join(os.TempDir(), "examplecreate")
	db, err := database.Create("ffldb", dbPath, wire.MainNet)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

	// Output:
}

// This example demonstrates creating a new database and using a managed
// read-write transaction to store and retrieve metadata.
func Example_basicUsage() {
	// This example assumes the ffldb driver is imported.
	//
	// import (
	// 	"github.com/mit-dci/lit/btcutil/database"
	// 	_ "github.com/mit-dci/lit/btcutil/database/ffldb"
	// )

	// Create a database and schedule it to be closed and removed on exit.
	// Typically you wouldn't want to remove the database right away like
	// this, nor put it in the temp directory, but it's done here to ensure
	// the example cleans up after itself.
	dbPath := filepath.Join(os.TempDir(), "exampleusage")
	db, err := database.Create("ffldb", dbPath, wire.MainNet)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

	// Use the Update function of the database to perform a managed
	// read-write transaction.  The transaction will automatically be rolled
	// back if the supplied inner function returns a non-nil error.
	err = db.Update(func(tx database.Tx) error {
		// Store a key/value pair directly in the metadata bucket.
		// Typically a nested bucket would be used for a given feature,
		// but this example is using the metadata bucket directly for
		// simplicity.
		key := []byte("mykey")
		value := []byte("myvalue")
		if err := tx.Metadata().Put(key, value); err != nil {
			return err
		}

		// Read the key back and ensure it matches.
		if !bytes.Equal(tx.Metadata().Get(key), value) {
			return fmt.Errorf("unexpected value for key '%s'", key)
		}

		// Create a new nested bucket under the metadata bucket.
		nestedBucketKey := []byte("mybucket")
		nestedBucket, err := tx.Metadata().CreateBucket(nestedBucketKey)
		if err != nil {
			return err
		}

		// The key from above that was set in the metadata bucket does
		// not exist in this new nested bucket.
		if nestedBucket.Get(key) != nil {
			return fmt.Errorf("key '%s' is not expected nil", key)
		}

		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	// Output:
}

// This example demonstrates creating a new database, using a managed read-write
// transaction to store a block, and using a managed read-only transaction to
// fetch the block.
func Example_blockStorageAndRetrieval() {
	// This example assumes the ffldb driver is imported.
	//
	// import (
	// 	"github.com/mit-dci/lit/btcutil/database"
	// 	_ "github.com/mit-dci/lit/btcutil/database/ffldb"
	// )

	// Create a database and schedule it to be closed and removed on exit.
	// Typically you wouldn't want to remove the database right away like
	// this, nor put it in the temp directory, but it's done here to ensure
	// the example cleans up after itself.
	dbPath := filepath.Join(os.TempDir(), "exampleblkstorage")
	db, err := database.Create("ffldb", dbPath, wire.MainNet)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

	// Use the Update function of the database to perform a managed
	// read-write transaction and store a genesis block in the database as
	// and example.
	err = db.Update(func(tx database.Tx) error {
		genesisBlock := chaincfg.MainNetParams.GenesisBlock
		temp := btcutil.NewBlock(genesisBlock)
		tx.StoreBlock(temp)

		genesisHash := chaincfg.MainNetParams.GenesisHash
		blockBytes, err := tx.FetchBlock(genesisHash)
		if err != nil {
			return err
		}

		// As documented, all data fetched from the database is only
		// valid during a database transaction in order to support
		// zero-copy backends.  Thus, make a copy of the data so it
		// can be used outside of the transaction.
		loadedBlockBytes := make([]byte, len(blockBytes))
		copy(loadedBlockBytes, blockBytes)

		fmt.Printf("Serialized block size: %d bytes\n", len(loadedBlockBytes))
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}
}
