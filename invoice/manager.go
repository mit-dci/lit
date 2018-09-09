package invoice

import (
	"github.com/boltdb/bolt"
)

type InvoiceManager struct {
	InvoiceDB *bolt.DB
}

// NewManager generates a new manager to add to the LitNode
func NewManager(dbPath string) (*InvoiceManager, error) {

	var mgr InvoiceManager
	err := mgr.InitDB(dbPath)
	if err != nil {
		return nil, err
	}

	return &mgr, nil
}
