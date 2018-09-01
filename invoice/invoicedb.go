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
	BKTGotPaidInvoices = []byte("GotPaidInvoices")
	// we got paid for these invoices, nice
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
		_, err = tx.CreateBucketIfNotExists(BKTGotPaidInvoices)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(BKTPaidInvoices)
		return err
	})

	return nil
}

func (mgr *InvoiceManager) SaveGeneratedInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	return mgr.storeInvoiceReplyMsgInBucket(BKTGeneratedInvoices, invoice)
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

func (mgr *InvoiceManager) storeInvoiceMsgInBucket(bucketName []byte, invoice *lnutil.InvoiceMsg) error {
	// log invoices which we send to peers indexed by most recent added first
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

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

func (mgr *InvoiceManager) DoesKeyExist(bucketName []byte, invoiceId string) (bool, error) {
	free := false
	var err error
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		v := b.Get([]byte(invoiceId))
		if v == nil {
			free = true
		} else {
			log.Println(v)
		}
		return nil
	})
	return free, err
}

func (mgr *InvoiceManager) SaveRepliedInvoice(invoice *lnutil.InvoiceMsg) error {
	// log invoices which we send to peers indexed by most recent added first
	return mgr.storeInvoiceMsgInBucket(BKTRepliedInvoices, invoice)
}

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

func (mgr *InvoiceManager) DeleteInt(bucketName []byte, key uint32) error {
	// for all those  buckets indexed by peeridx
	// or bukcets that use InvoiceMsg
	keyString := strconv.Itoa(int(key))
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		return bucket.Delete([]byte(keyString))
	})
	return err
}

func (mgr *InvoiceManager) DeleteString(bucketName []byte, key string) error {
	// for all those buckets indexed by invoiceId
	// or bukcets that use InvoiceReplyMsg
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		return bucket.Delete([]byte(key))
	})
	return err
}

// storeInvoiceReplyMsgInBucket is a common handler that is shared between all instances
// which want to write to a bucket bucketName
func (mgr *InvoiceManager) storeInvoiceReplyMsgInBucket(bucketName []byte, invoice *lnutil.InvoiceReplyMsg) error {
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

func (mgr *InvoiceManager) SaveRequestedInvoice(invoice *lnutil.InvoiceMsg) error {
	// if we requested an invoice from someone, all we need to know is their pubkley
	// and invoice id to check against teh reply they send us. Cointype and Amount aren;t
	// defined at this point yet
	// we sent someone an invoice request ie we want to pay them money
	// log invoices which we send to peers indexed by most recent added first
	return mgr.storeInvoiceMsgInBucket(BKTRequestedInvoices, invoice)
}

func (mgr *InvoiceManager) LoadRequestedInvoice(peerIdx uint32,
	invoiceId string) (lnutil.InvoiceMsg, error) {
	// retrieve the peerId attached to the given peerIdx. Hopefully there should be one
	// right now oncly returns the requested invoiceId
	var err error
	var finmsg lnutil.InvoiceMsg
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTRequestedInvoices)
		// since RequestedInvoices are indexed by peeridx
		b.ForEach(func(k, v []byte) error {
			fmt.Printf("key=%s, value=%s\n", k, v)
			// k is a byte slice which needs to be converted to an int
			// could by [51] or [xx, xx]
			intk, _ := strconv.Atoi(string(k))
			if err != nil {
				return err
			}
			if intk == int(peerIdx) && bytes.Equal(v, []byte(invoiceId)) {
				msg, err := InvoiceMsgFromBytes(v)
				if err != nil {
					return err
				}
				finmsg = msg
				finmsg.PeerIdx = peerIdx
				return nil
			}
			return nil
		})
		if len(finmsg.Id) == 0 {
			return fmt.Errorf("InvoiceId %s fo peer %d does not exist", invoiceId, peerIdx)
		}
		return nil
	})
	return finmsg, err
}

func (mgr *InvoiceManager) SavePendingInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	return mgr.storeInvoiceReplyMsgInBucket(BKTPendingInvoices, invoice)
}

func (mgr *InvoiceManager) LoadPendingInvoice(peerIdx uint32,
	invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	return mgr.loadFromBucket(BKTPendingInvoices, peerIdx, invoiceId)
}

func (mgr *InvoiceManager) SavePaidInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	return mgr.storeInvoiceReplyMsgInBucket(BKTPaidInvoices, invoice)
}

func (mgr *InvoiceManager) LoadPaidInvoice(peerIdx uint32,
	invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	return mgr.loadFromBucket(BKTPaidInvoices, peerIdx, invoiceId)
}

func (mgr *InvoiceManager) SaveGotPaidInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	// save the invocies we got paid for in the bucket
	return mgr.storeInvoiceReplyMsgInBucket(BKTPaidInvoices, invoice)
}

func (mgr *InvoiceManager) displayAllKeyVals(bucketName []byte) ([]lnutil.InvoiceReplyMsg, error) {
	// read through all the key value pairs in the bucket and return them as a slice
	// so that we can display them in a nice way
	// InvoiceReplyMsg params for easy reference
	//	PeerIdx  uint32
	//	Id       string
	//	CoinType string
	//	Amount   uint64
	var err error
	var finmsg []lnutil.InvoiceReplyMsg
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		b.ForEach(func(k, v []byte) error {
			fmt.Printf("key=%s, value=%s\n", k, v)
			msg, err := InvoiceReplyMsgFromBytes(v)
			if err != nil {
				return err
			}
			log.Println("APPENDING MSG", msg)
			finmsg = append(finmsg, msg)
			return nil
		})
		return nil
	})
	return finmsg, err
}

// helper functions to return stuff easily via the RPC
// might be repetitive, but avoids the need for people to call the displayAllKeyVals
// function each time on byte strings
func (mgr *InvoiceManager) GetAllGeneratedInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.displayAllKeyVals(BKTGeneratedInvoices)
}
func (mgr *InvoiceManager) GetAllRepliedInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.displayAllKeyVals(BKTRepliedInvoices)
}
func (mgr *InvoiceManager) GetAllRequestedInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.displayAllKeyVals(BKTRequestedInvoices)
}
func (mgr *InvoiceManager) GetAllPendingInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.displayAllKeyVals(BKTPendingInvoices)
}
func (mgr *InvoiceManager) GetAllPaidInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.displayAllKeyVals(BKTPaidInvoices)
}
func (mgr *InvoiceManager) GetAllGotPaidInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.displayAllKeyVals(BKTGotPaidInvoices)
}
