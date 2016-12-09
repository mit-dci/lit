package qln

import "github.com/boltdb/bolt"

func (nd *LnNode) Init(dbfilename, watchname string, basewal UWallet) error {

	err := nd.OpenDB(dbfilename)
	if err != nil {
		return err
	}

	nd.OmniChan = make(chan []byte, 10)
	go nd.OmniHandler()

	// connect to base wallet
	nd.BaseWallet = basewal
	// ask basewallet for outpoint event messages
	go nd.OPEventHandler()

	err = nd.Tower.OpenDB(watchname)
	if err != nil {
		return err
	}
	bc := nd.BaseWallet.BlockMonitor()
	go nd.Tower.BlockHandler(bc)

	return nil
}

// Opens the DB file for the LnNode
func (nd *LnNode) OpenDB(filename string) error {
	var err error

	nd.LnDB, err = bolt.Open(filename, 0644, nil)
	if err != nil {
		return err
	}
	// create buckets if they're not already there
	err = nd.LnDB.Update(func(btx *bolt.Tx) error {
		_, err := btx.CreateBucketIfNotExists(BKTPeers)
		if err != nil {
			return err
		}

		_, err = btx.CreateBucketIfNotExists(BKTWatch)
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
