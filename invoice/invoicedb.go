package invoice

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/lnutil"
)

// const strings for db usage
var (
	BKTGeneratedInvoices = []byte("GeneratedInvoices")
	// store generated invoices
	BKTSentInvoicesOut = []byte("SentInvoicesOut")
	// sent an invoice in geninvoices out to someone
	BKTSentInvoiceReq = []byte("SentInvoiceReq")
	// send another peer a request for an invoice
	BKTPendingInvoices = []byte("PendingInvoices")
	// received invoice info from someone, check against SentInvoiceRequest
	BKTPaidInvoices = []byte("PaidInvoices")
	// paid this invoice successfully, store in db
)

// InitDB initializes the database for Discreet Log Contract storage
func (mgr *InvoiceManager) InitDB(dbPath string) error {
	var err error

	mgr.InvoiceDB, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}

	// Ensure buckets exist that we need
	err = mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(BKTGeneratedInvoices)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(BKTSentInvoicesOut)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(BKTSentInvoiceReq)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(BKTPendingInvoices)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(BKTPaidInvoices)
		return err
	})

	return nil
}


func (mgr *InvoiceManager) SaveGeneratedInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTGeneratedInvoices)

		var writeBuffer bytes.Buffer
		binary.Write(&writeBuffer, binary.BigEndian, invoice.Id)
		log.Println("STORING INVOICE", invoice)
		err := b.Put([]byte(invoice.Id), invoice.Bytes())
		// index by invoice ID

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
func (mgr *InvoiceManager) LoadGeneratedInvoice(invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	var msg lnutil.InvoiceReplyMsg
	var err error
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTGeneratedInvoices)
		v := b.Get([]byte(invoiceId))
		if v == nil {
			return fmt.Errorf("InvoiceId %d does not exist", invoiceId)
		}
		msg, err = InvoiceMsgFromBytes(v)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return msg, err
	}
	log.Println("PRITNED THIS OUT", msg)
	return msg, nil
}
