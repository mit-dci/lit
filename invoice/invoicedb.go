package invoice

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"time"

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

// DoesKeyExist works for all buckets indexed by invoiceId
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

// buckets that use InvoiceMsg / peerIdx
func (mgr *InvoiceManager) DeletePeerIdx(bucketName []byte, key uint32) error {
	keyString := strconv.Itoa(int(key))
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		return bucket.Delete([]byte(keyString))
	})
	return err
}

// buckets that use InvoiceReplyMsg / invoiceId
func (mgr *InvoiceManager) DeleteInvoiceId(bucketName []byte, key string) error {
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		return bucket.Delete([]byte(key))
	})
	return err
}

// storeInvoiceMsgInBucket is indexed by PeerIdx
func (mgr *InvoiceManager) storeInvoiceMsgInBucket(bucketName []byte, invoice *lnutil.InvoiceMsg) error {
	// log invoices which we send to peers indexed by most recent added first
	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		var writeBuffer bytes.Buffer
		binary.Write(&writeBuffer, binary.BigEndian, invoice.PeerIdx)
		temp := strconv.Itoa(int(invoice.PeerIdx)) // indexing by peeridx
		err := b.Put([]byte(temp), invoice.Bytes())
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

// storeIRMsgInBucket is a common handler that is shared between all instances
// which want to write to a bucket bucketName
func (mgr *InvoiceManager) storeIRMsgInBucket(bucketName []byte, invoice *lnutil.InvoiceReplyMsg) error {
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

// getAllFromBucket is a common handler for all methods which want to go through
// the key value pairs in the buckets and then retrieve the invoice information
// form the respective buckets.
func (mgr *InvoiceManager) getAllFromBucket(bucketName []byte, peerIdx uint32,
	invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	var err error
	var retmsg lnutil.InvoiceReplyMsg
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		v := b.Get([]byte(invoiceId))
		b.ForEach(func(k, v []byte) error {
			if k[0] == invoiceId[0] {
				msg, err := lnutil.IRMsgFromBytes(v)
				if err != nil {
					return err
				}
				if msg.PeerIdx != peerIdx {
					log.Println("Invoice Ids don't match, probably another guy's invoice!")
				} else {
					retmsg = msg
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
		log.Println(err)
	}
	return retmsg, err
}

func (mgr *InvoiceManager) getInvoicesWTimestamp(bucketName []byte) ([]PaidInvoiceStorage, error) {
	// read through all the key value pairs in the bucket and return them as a slice
	// so that we can display them in a nice way
	var err error
	var retmsg []PaidInvoiceStorage
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		b.ForEach(func(k, v []byte) error {
			msg := PaidInvoiceStorageFromBytes(v)
			retmsg = append(retmsg, msg)
			return nil
		})
		return nil
	})
	return retmsg, err
}

func (mgr *InvoiceManager) getAllInvoiceMsgs(bucketName []byte) ([]lnutil.InvoiceMsg, error) {
	var err error
	var retmsg []lnutil.InvoiceMsg
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		b.ForEach(func(k, v []byte) error {
			msg := IIdFromBytes(v)
			// convert k to uint32
			kuint, err := strconv.Atoi(string(k))
			if err != nil {
				return err
			}
			msg.PeerIdx = uint32(kuint) // because the invociemsgfrombytes function doesn't parse peeridx
			retmsg = append(retmsg, msg)
			return nil
		})
		return nil
	})
	return retmsg, err
}

func (mgr *InvoiceManager) getAllInvoiceReplyMsgs(bucketName []byte) ([]lnutil.InvoiceReplyMsg, error) {
	// read through all the key value pairs in the bucket and return them as a slice
	// so that we can display them in a nice way
	var err error
	var retmsg []lnutil.InvoiceReplyMsg
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		b.ForEach(func(k, v []byte) error {
			//fmt.Printf("key=%s, value=%s\n", k, v)
			msg, err := lnutil.IRMsgFromBytes(v)
			if err != nil {
				return err
			}
			retmsg = append(retmsg, msg)
			return nil
		})
		return nil
	})
	return retmsg, err
}

// save the invoices we got paid for in the bucket
func (mgr *InvoiceManager) saveInvoicesWTimestamp(
	invoice *lnutil.InvoiceReplyMsg, bucketName []byte) error {
	var temp PaidInvoiceStorage
	temp.PeerIdx = invoice.PeerIdx
	temp.Id = invoice.Id
	temp.CoinType = invoice.CoinType
	temp.Amount = invoice.Amount
	temp.Timestamp = time.Now().Format("01/02/2006 15:04:05")
	// have our own format so that we can be sure about the size of the timestamp

	err := mgr.InvoiceDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		var writeBuffer bytes.Buffer
		binary.Write(&writeBuffer, binary.BigEndian, temp.Timestamp)
		// index with timestamp since its deemde to be unique (with known quantum physics ie)
		err := b.Put([]byte(temp.Timestamp), temp.Bytes())
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

func (mgr *InvoiceManager) SaveGeneratedInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	return mgr.storeIRMsgInBucket(BKTGeneratedInvoices, invoice)
}

// LoadGeneratedInvoice loads a given invoice based on its invoiceId from memory
// there is no need for peerIDx here because this is retrieved from our storage
// and while creating an invoice, we don't know who / what is going to pay us
// LoadGeneratedInvoice is indexed by invoiceId
func (mgr *InvoiceManager) LoadGeneratedInvoice(invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	var msg lnutil.InvoiceReplyMsg
	var err error
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTGeneratedInvoices)
		v := b.Get([]byte(invoiceId)) // indexed by invoiceId
		if v == nil {
			return fmt.Errorf("InvoiceId %d does not exist", invoiceId)
		}
		msg, err = lnutil.IRMsgFromBytes(v)
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
		msg = IIdFromBytes(v)
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

// we sent someone an invoice request ie we want to pay them money
func (mgr *InvoiceManager) SaveRequestedInvoice(invoice *lnutil.InvoiceMsg) error {
	// if we requested an invoice from someone, all we need to know is their pubkley
	// and invoice id to check against the reply they send us. Cointype and Amount aren't
	// defined at this point yet
	// log invoices which we send to peers indexed by most recent added first
	return mgr.storeInvoiceMsgInBucket(BKTRequestedInvoices, invoice)
}

func (mgr *InvoiceManager) LoadRequestedInvoice(peerIdx uint32,
	invoiceId string) (lnutil.InvoiceMsg, error) {
	// retrieve the peerId attached to the given peerIdx. Hopefully there should be one
	// right now only returns the requested invoiceId
	var err error
	var retmsg lnutil.InvoiceMsg
	err = mgr.InvoiceDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BKTRequestedInvoices)
		// since RequestedInvoices are indexed by peeridx
		b.ForEach(func(k, v []byte) error {
			// k is a byte slice which needs to be converted to an int
			// could by [51] or [xx, xx] or [xx,yy,zz] and so on
			intk, _ := strconv.Atoi(string(k))
			if err != nil {
				return err
			}
			if intk == int(peerIdx) && bytes.Equal(v, []byte(invoiceId)) {
				retmsg = IIdFromBytes(v)
				retmsg.PeerIdx = peerIdx // set peerIdx separately since IIdFromBytes doesn't handle that
				return nil
			}
			return nil
		})
		if len(retmsg.Id) == 0 {
			return fmt.Errorf("InvoiceId %s fo peer %d does not exist", invoiceId, peerIdx)
		}
		return nil
	})
	return retmsg, err
}

func (mgr *InvoiceManager) SavePendingInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	return mgr.storeIRMsgInBucket(BKTPendingInvoices, invoice)
}

func (mgr *InvoiceManager) LoadPendingInvoice(peerIdx uint32,
	invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	return mgr.getAllFromBucket(BKTPendingInvoices, peerIdx, invoiceId)
}

func (mgr *InvoiceManager) SavePaidInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	return mgr.saveInvoicesWTimestamp(invoice, BKTPaidInvoices)
}

func (mgr *InvoiceManager) LoadPaidInvoice(peerIdx uint32,
	invoiceId string) (lnutil.InvoiceReplyMsg, error) {
	return mgr.getAllFromBucket(BKTPaidInvoices, peerIdx, invoiceId)
}

func (mgr *InvoiceManager) SaveGotPaidInvoice(invoice *lnutil.InvoiceReplyMsg) error {
	return mgr.saveInvoicesWTimestamp(invoice, BKTGotPaidInvoices)
}

// helper functions to return stuff easily via the RPC, could rmeove them, but
// nice to have easy to remmeber / call RPCs (instead of the uglier displayAllKeyVals)
// might be repetitive, but avoids the need for people to call the displayAllKeyVals
// function each time on byte strings
// need invoicemsges to work
func (mgr *InvoiceManager) GetAllRepliedInvoices() ([]lnutil.InvoiceMsg, error) {
	return mgr.getAllInvoiceMsgs(BKTRepliedInvoices)
}

func (mgr *InvoiceManager) GetAllRequestedInvoices() ([]lnutil.InvoiceMsg, error) {
	return mgr.getAllInvoiceMsgs(BKTRequestedInvoices)
}

func (mgr *InvoiceManager) GetAllGeneratedInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.getAllInvoiceReplyMsgs(BKTGeneratedInvoices)
}

func (mgr *InvoiceManager) GetAllPendingInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.getAllInvoiceReplyMsgs(BKTPendingInvoices)
}

func (mgr *InvoiceManager) GetAllPaidInvoices() ([]PaidInvoiceStorage, error) {
	return mgr.getInvoicesWTimestamp(BKTPaidInvoices)
}

func (mgr *InvoiceManager) GetAllGotPaidInvoices() ([]PaidInvoiceStorage, error) {
	return mgr.getInvoicesWTimestamp(BKTGotPaidInvoices)
}
