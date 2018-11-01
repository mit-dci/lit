package lnbolt

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
)

var (
	peersLabel    = []byte(`peers`)
	peerMetaLabel = []byte(`peersmeta`)
	pdbbuckets    = [][]byte{
		peersLabel,
		peerMetaLabel,
	}

	peerIdxLast = []byte(`lastpeeridx`)
)

type peerboltdb struct {
	db *bolt.DB
}

func (pdb *peerboltdb) init() error {
	err := pdb.db.Update(func(tx *bolt.Tx) error {
		for _, n := range pdbbuckets {
			_, err := tx.CreateBucketIfNotExists(n)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (pdb *peerboltdb) GetPeerAddrs() ([]string, error) {

	addrs := make([]string, 0)

	err := pdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(peersLabel)

		// Iterate over all of the members of the bucket.
		cur := b.Cursor()
		atmp := make([]string, 0)
		for {
			k, _ := cur.Next()
			if k == nil {
				break
			}
			atmp = append(atmp, string(k))
		}

		// Now that we have the final array return it.
		addrs = atmp
		return nil
	})
	if err != nil {
		return nil, err
	}

	return addrs, nil
}

func (pdb *peerboltdb) GetPeerInfo(addr string) (lncore.PeerInfo, error) {

	var raw []byte
	var err error
	var peerInfo lncore.PeerInfo

	if pdb.db == nil {
		logging.Warnf("PDB.db is nil!")
		return peerInfo, fmt.Errorf("PDB.db is nil")
	}

	// Just get the raw data from the DB.
	err = pdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(peersLabel)
		raw = b.Get([]byte(addr))
		return nil
	})
	if err != nil {
		return peerInfo, err
	}

	if raw == nil {
		return peerInfo, nil
	}

	err = json.Unmarshal(raw, &peerInfo)
	return peerInfo, err
}

func (pdb *peerboltdb) GetPeerInfos() (map[string]lncore.PeerInfo, error) {

	var out map[string]lncore.PeerInfo
	var err error

	err = pdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(peersLabel)

		// Iterate over everything.
		cur := b.Cursor()
		mtmp := map[string]lncore.PeerInfo{}
		for {
			k, v := cur.Next()
			if k == nil {
				break
			}

			var pi lncore.PeerInfo
			err2 := json.Unmarshal(v, &pi) // TODO Move outside tx block.
			if err2 != nil {
				return err2
			}

			ka := string(k)
			mtmp[ka] = pi

		}

		out = mtmp
		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil

}

func (pdb *peerboltdb) AddPeer(addr string, pi lncore.PeerInfo) error {
	return pdb.UpdatePeer(addr, pi)
}

func (pdb *peerboltdb) UpdatePeer(addr string, pi lncore.PeerInfo) error {

	var err error

	araw := []byte(addr)
	piraw, err := json.Marshal(pi)
	if err != nil {
		return err
	}

	err = pdb.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(peersLabel)

		err2 := b.Put(araw, piraw)
		if err2 != nil {
			return err2
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil

}

func (pdb *peerboltdb) DeletePeer(addr string) error {

	var err error

	araw := []byte(addr)
	err = pdb.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(peersLabel)

		err2 := b.Delete(araw)
		if err2 != nil {
			return err2
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil

}

func (pdb *peerboltdb) GetUniquePeerIdx() (uint32, error) {

	var err error
	var pidx uint32

	err = pdb.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(peerMetaLabel)

		// Get the last unique peer idx, or create it.
		v := b.Get(peerIdxLast)
		if v == nil {
			x := lnutil.U32tB(1)
			err2 := b.Put(peerIdxLast, x)
			if err2 != nil {
				return err2
			}
			v = x
		}
		pidx = lnutil.BtU32(v)

		// Increment it.
		err2 := b.Put(peerIdxLast, lnutil.U32tB(pidx+1))
		if err2 != nil {
			return err2
		}

		return nil
	})

	if err != nil {
		return 0, err
	}
	return pidx, nil
}
