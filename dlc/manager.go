package dlc

import (
	"github.com/boltdb/bolt"
)

type DlcManager struct {
	DLCDB *bolt.DB
}

// NewManager generates a new manager to add to the LitNode
func NewManager(dbPath string) (*DlcManager, error) {

	var mgr DlcManager
	err := mgr.InitDB(dbPath)
	if err != nil {
		return nil, err
	}

	return &mgr, nil
}
