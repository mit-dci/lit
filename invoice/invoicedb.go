package invoice

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/lnutil"
)

// const strings for easy db usage
var (
	BKTGeneratedInvoices = []byte("GeneratedInvoices")
	// store generated invoices
	BKTRepliedInvoices = []byte("RepliedInvoices")
	// sent an invoice in geninvoices out to someone
	BKTRequestedInvoices = []byte("RequestedInvoices")
	// send another peer a request for an invoice
	BKTPendingInvoices = []byte("PendingInvoices")
	// received invoice info from someone, check against SentInvoiceRequest
	BKTPaidInvoices = []byte("PaidInvoices")
	// paid this invoice successfully, store in db
)

// InitDB declares and instantiates invoices.db storage
func (mgr *InvoiceManager) InitDB(dbPath string) error {
	var err error

	mgr.InvoiceDB, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}

	// Ensure the buckets we need exist
	err = mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(BKTGeneratedInvoices)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(BKTRepliedInvoices)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(BKTRequestedInvoices)
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
	return mgr.storeinBucket(BKTGeneratedInvoices, invoice)
}

// LoadGeneratedInvoice loads a given invoice based on its invoiceId from memory
// there is no need for peerIDx here because this is retrieved from our storage
// and while creating an invoice, we don't know who / what is going to pay us
func (mgr *InvoiceManager) LoadGeneratedInvoice(invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	var msg lnutil.InvoiceReplyMsg
	var err error
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTGeneratedInvoices)
		v := b.Get([]byte(invoiceId))
		if v == nil {
			return fmt.Errorf("InvoiceId %d does not exist", invoiceId)
		}
		msg, err = InvoiceReplyMsgFromBytes(v)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return msg, err
	}
	return msg, nil
}

func (mgr *InvoiceManager) SaveRepliedInvoice(invoice *lnutil.InvoiceMsg) error {
	// log invoices which we send to peers indexed by most recent added first
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTRepliedInvoices)

		var writeBuffer bytes.Buffer
		binary.Write(&writeBuffer, binary.BigEndian, invoice.PeerIdx)
		temp := strconv.Itoa(int(invoice.PeerIdx))
		err := b.Put([]byte(temp), invoice.Bytes())
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
func (mgr *InvoiceManager) LoadRepliedInvoice(peerIdx string) (lnutil.InvoiceMsg, error) {
	// retrieve the peerId attached to the given peerIdx. Hopefully there should be one
	var msg lnutil.InvoiceMsg
	var err error
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTRepliedInvoices)
		v := b.Get([]byte(peerIdx))
		if v == nil {
			return fmt.Errorf("peerIdx %d does not exist", peerIdx)
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
	// convert string to uint32
	temp, err := strconv.Atoi(peerIdx)
	if err != nil {
		return msg, err
	}
	msg.PeerIdx = uint32(temp)
	return msg, nil
}

// storeinBucket is a common handler that is shared between all instances
// which want to write to a bucket bucketName
func (mgr *InvoiceManager) storeinBucket(bucketName []byte, invoice *lnutil.InvoiceReplyMsg) error {
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		var writeBuffer bytes.Buffer
		binary.Write(&writeBuffer, binary.BigEndian, invoice.Id)
		err := b.Put([]byte(invoice.Id), invoice.Bytes())
		// taking advantage of the bytes method defined in lnutil
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

// loadFromBucket is a common handler for all methods which want to go through
// the key value pairs in the buckets and then retrieve the invoice information
// form the respective buckets.
func (mgr *InvoiceManager) loadFromBucket(bucketName []byte, peerIdx uint32,
	invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	// InvoiceReplyMsg params for easy reference
	//	PeerIdx  uint32
	//	Id       string
	//	CoinType string
	//	Amount   uint64
	var err error
	var finmsg lnutil.InvoiceReplyMsg
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		v := b.Get([]byte(invoiceId))
		b.ForEach(func(k, v []byte) error {
			fmt.Printf("key=%s, value=%s\n", k, v)
			if k[0] == invoiceId[0] {
				msg, err := InvoiceReplyMsgFromBytes(v)
				if err != nil {
					return err
				}
				log.Println("MSG", msg, peerIdx)
				if msg.PeerIdx != peerIdx {
					log.Println("NOPE, ANOTHER GUY'S INVOICE")
				} else {
					finmsg = msg
				}
			}
			return nil
		})
		if v == nil {
			return fmt.Errorf("InvoiceId %d does not exist", invoiceId)
		}
		return nil
	})
	if err != nil {
		return finmsg, err
	}
	return finmsg, nil
}

func (mgr *InvoiceManager) SaveRequestedInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	// we sent someone an invoice request ie we want to pay them money
	return mgr.storeinBucket(BKTRequestedInvoices, invoice)
}

func (mgr *InvoiceManager) LoadRequestedInvoice(peerIdx uint32,
	invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	return mgr.loadFromBucket(BKTRequestedInvoices, peerIdx, invoiceId)
}

func (mgr *InvoiceManager) SavePendingInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	return mgr.storeinBucket(BKTPendingInvoices, invoice)
}

func (mgr *InvoiceManager) LoadPendingInvoice(peerIdx uint32,
	invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	return mgr.loadFromBucket(BKTPendingInvoices, peerIdx, invoiceId)
}

func (mgr *InvoiceManager) SavePaidInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	return mgr.storeinBucket(BKTPaidInvoices, invoice)
}

func (mgr *InvoiceManager) LoadPaidInvoice(peerIdx uint32,
	invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	return mgr.loadFromBucket(BKTPaidInvoices, peerIdx, invoiceId)
}
