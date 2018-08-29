package invoice

import (
	"bytes"
	"encoding/binary"
	"fmt"
	//"log"
	"strconv"

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

func (mgr *InvoiceManager) SaveSentInvoicesOut(invoice *lnutil.InvoiceMsg) error {
	// log invoices which we send to peers indexed by most recent added first
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTSentInvoicesOut)

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
func (mgr *InvoiceManager) LoadSentInvoicesOut(peerIdx string) (lnutil.InvoiceMsg, error) {
	// retrieve the peerId attached to the given peerIdx. Hopefully should be one
	var msg lnutil.InvoiceMsg
	var err error
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTSentInvoicesOut)
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

// BKTSentInvoiceReq
// Most stuff below this is just a repetition of what we've seen before, have to repeat.
// is there a better way to do it?
func (mgr *InvoiceManager) SaveSentInvoiceReq(invoice *lnutil.InvoiceReplyMsg) error {
	// we sent someone an invoice request
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTSentInvoiceReq)

		var writeBuffer bytes.Buffer
		binary.Write(&writeBuffer, binary.BigEndian, invoice.Id)
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

// Need to ask sent invoices indexed by peerId since there may be multiple ivnoiceIds
// that we ask from different peers
// but key, value pairs, so need to go through multiple ones
func (mgr *InvoiceManager) LoadSentInvoiceReq(peerIdx uint32, invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	// do we need both peeridx and invoiceId here? idk
	var msg lnutil.InvoiceReplyMsg
	var err error
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTGeneratedInvoices)

		v := b.Get([]byte(invoiceId))
		msg, err = InvoiceReplyMsgFromBytes(v)
		if err != nil {
			return err
		}
		// 	PeerIdx  uint32
		// 	Id       string
		// 	CoinType string
		// 	Amount   uint64
		b.ForEach(func(k, v []byte) error {
			fmt.Printf("key=%s, value=%s\n", k, v)
			if k[0] == invoiceId[0] {
				fmt.Println("INVOICE ID", k[0])
				msg, err = InvoiceReplyMsgFromBytes(v)
				if err != nil {
					return err
				}
				fmt.Println("BINGO")
				fmt.Println("CATCH HIST", msg.PeerIdx == peerIdx)
			}
			return nil
		})
		if v == nil {
			return fmt.Errorf("InvoiceId %d does not exist", invoiceId)
		}
		return nil
	})
	if err != nil {
		return msg, err
	}
	return msg, nil
}
