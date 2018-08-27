package lnbolt

import (
	"encoding/json"
	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/lnio"
)

var (
	peersLabel = []byte(`peers`)
	pdbbuckets = [][]byte{
		peersLabel,
	}
)

type peerboltdb struct {
	db *bolt.DB
}

func (pdb peerboltdb) init() error {
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

func (pdb peerboltdb) GetPeerAddrs() ([]lnio.LnAddr, error) {

	addrs := make([]lnio.LnAddr, 0)

	err := pdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(peersLabel)

		// Iterate over all of the members of the bucket.
		cur := b.Cursor()
		atmp := make([]lnio.LnAddr, 0)
		for {
			k, _ := cur.Next()
			if k == nil {
				break
			}
			atmp = append(atmp, lnio.LnAddr(string(k)))
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

func (pdb peerboltdb) GetPeerInfo(addr lnio.LnAddr) (*lnio.PeerInfo, error) {

	var raw []byte
	var err error

	// Just get the raw data from the DB.
	err = pdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(peersLabel)
		raw = b.Get([]byte(string(addr)))
		return nil
	})
	if err != nil {
		return nil, err
	}

	var pi lnio.PeerInfo
	err = json.Unmarshal(raw, &pi)
	if err != nil {
		return nil, err
	}

	return &pi, nil

}

func (pdb peerboltdb) GetPeerInfos() (map[lnio.LnAddr]lnio.PeerInfo, error) {

	var out map[lnio.LnAddr]lnio.PeerInfo
	var err error

	err = pdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(peersLabel)

		// Iterate over everything.
		cur := b.Cursor()
		mtmp := map[lnio.LnAddr]lnio.PeerInfo{}
		for {
			k, v := cur.Next()
			if k == nil {
				break
			}

			var pi lnio.PeerInfo
			err2 := json.Unmarshal(v, &pi) // TODO Move outside tx block.
			if err2 != nil {
				return err2
			}

			ka := lnio.LnAddr(string(k))
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

func (pdb peerboltdb) AddPeer(addr lnio.LnAddr, pi lnio.PeerInfo) error {

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

func (pdb peerboltdb) DeletePeer(addr lnio.LnAddr) error {

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
