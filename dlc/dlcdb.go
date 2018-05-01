package dlc

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/lnutil"
)

// const strings for db usage
var (
	BKTOracles   = []byte("Oracles")
	BKTContracts = []byte("Contracts")
)

// InitDB initializes the database for Discreet Log Contract storage
func (mgr *DlcManager) InitDB(dbPath string) error {
	var err error

	mgr.DLCDB, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}

	// Ensure buckets exist that we need
	err = mgr.DLCDB.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(BKTOracles)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(BKTContracts)
		return err
	})

	return nil
}

// SaveOracle saves an oracle into the database. Generates a new index if the
// passed oracle doesn't have one
func (mgr *DlcManager) SaveOracle(o *DlcOracle) error {
	err := mgr.DLCDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTOracles)

		if o.Idx == 0 {
			o.Idx, _ = b.NextSequence()
		}
		var wb bytes.Buffer
		binary.Write(&wb, binary.BigEndian, o.Idx)
		err := b.Put(wb.Bytes(), o.Bytes())

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

// LoadOracle loads an oracle from the database by index.
func (mgr *DlcManager) LoadOracle(idx uint64) (*DlcOracle, error) {
	o := new(DlcOracle)

	err := mgr.DLCDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTOracles)

		var wb bytes.Buffer
		binary.Write(&wb, binary.BigEndian, idx)

		v := b.Get(wb.Bytes())

		if v == nil {
			return fmt.Errorf("Oracle %d does not exist", idx)
		}
		var err error
		o, err = DlcOracleFromBytes(v)
		if err != nil {
			return err
		}
		o.Idx = idx
		return nil
	})

	if err != nil {
		return nil, err
	}

	return o, nil

}

// ListOracles loads all oracles from the database and returns them as an array
func (mgr *DlcManager) ListOracles() ([]*DlcOracle, error) {
	oracles := make([]*DlcOracle, 0)
	err := mgr.DLCDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTOracles)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			buf := bytes.NewBuffer(k)
			o, err := DlcOracleFromBytes(v)
			if err != nil {
				return err
			}
			binary.Read(buf, binary.BigEndian, &o.Idx)
			oracles = append(oracles, o)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return oracles, nil
}

// SaveContract saves a contract into the database. Will generate a new index
// if the passed object doesn't have one.
func (mgr *DlcManager) SaveContract(c *lnutil.DlcContract) error {
	err := mgr.DLCDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTContracts)

		if c.Idx == 0 {
			c.Idx, _ = b.NextSequence()
		}
		var wb bytes.Buffer
		binary.Write(&wb, binary.BigEndian, c.Idx)
		err := b.Put(wb.Bytes(), c.Bytes())

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

// LoadContract loads a contract from the database by index.
func (mgr *DlcManager) LoadContract(idx uint64) (*lnutil.DlcContract, error) {
	c := new(lnutil.DlcContract)

	err := mgr.DLCDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTContracts)

		var wb bytes.Buffer
		binary.Write(&wb, binary.BigEndian, idx)

		v := b.Get(wb.Bytes())

		if v == nil {
			return fmt.Errorf("Contract %d does not exist", idx)
		}
		var err error
		c, err = lnutil.DlcContractFromBytes(v)
		if err != nil {
			return err
		}
		c.Idx = idx
		return nil
	})

	if err != nil {
		return nil, err
	}

	return c, nil

}

// ListContracts loads all contracts from the database
func (mgr *DlcManager) ListContracts() ([]*lnutil.DlcContract, error) {
	contracts := make([]*lnutil.DlcContract, 0)
	err := mgr.DLCDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTContracts)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			buf := bytes.NewBuffer(k)
			c, err := lnutil.DlcContractFromBytes(v)
			if err != nil {
				return err
			}
			binary.Read(buf, binary.BigEndian, &c.Idx)
			contracts = append(contracts, c)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return contracts, nil
}
