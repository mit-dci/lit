package dlc

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
)

// const strings for db usage
var (
	BKTOracles = []byte("Oracles")
)

func (mgr *DlcManager) InitDB(dbPath string) error {
	var err error

	mgr.DLCDB, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}

	// Ensure buckets exist that we need
	err = mgr.DLCDB.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(BKTOracles)
		return err
	})

	return nil
}

func (mgr *DlcManager) SaveOracle(o *Oracle) error {
	var index uint64
	err := mgr.DLCDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTOracles)

		index, _ = b.NextSequence()
		var wb bytes.Buffer
		binary.Write(&wb, binary.BigEndian, index)
		err := b.Put(wb.Bytes(), o.Bytes())

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}
	o.Idx = index
	return nil
}

func (mgr *DlcManager) LoadOracle(idx uint64) (*Oracle, error) {
	o := new(Oracle)

	err := mgr.DLCDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTOracles)

		var wb bytes.Buffer
		binary.Write(&wb, binary.BigEndian, idx)

		v := b.Get(wb.Bytes())

		if v == nil {
			return fmt.Errorf("Oracle %d does not exist", idx)
		}
		var err error
		o, err = OracleFromBytes(v)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return o, nil

}

func (mgr *DlcManager) LoadAllOracles() ([]*Oracle, error) {
	oracles := make([]*Oracle, 0)
	err := mgr.DLCDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTOracles)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			buf := bytes.NewBuffer(k)
			o, err := OracleFromBytes(v)
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
