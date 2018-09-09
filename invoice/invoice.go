package invoice

import (
	"github.com/mit-dci/lit/lnutil"
)

// if you're sending an invoice out, you need to store amount, cointype, peerIdx and invoiceId

// if you received an invoice request, you need to see whether you have the invoice in
// the generated invoice bucket, send it out and delete it

// if you want to send out an invoice request, you need to store invoiceId, remotePeer address

// if you received a reply to your previously sent invoice request, you need to store it in
// BKTPendingInvoices and then ask the user whether he wants to pay. If he
// does want to pay, you delete it from the pending bucket and move to the sent bucket.
// If he doesn't, remove it from Pending Invoices

// we really need only two types of messages
// type InvoiceReplyMsg struct {
// 	PeerIdx  uint32
// 	Id       string
// 	CoinType string
// 	Amount   uint64
// }
// and
// type InvoiceMsg struct {
// 	PeerIdx uint32
// 	Id      string
// }

// BKTGeneratedInvoices -> should store InvoiceReplyMsg with PeerIdx=""
// BKTSentInvoicesOut -> should store InvoiceMsg
// BKTSentInvoiceReq -> should store InvoiceReplyMsg
// BKTPendingInvoices -> should store InvoiceReplyMsg
// BKTPaidInvoices -> should store InvoiceReplyMsg with timestamp
// BKTGotPaidInvoices -> should store InvoiceReplyMsg with timestamp

// so import everything from lnutil (and don't import this there)
// the Bytes() function is defined in lnutil, but the reverse isnt, which
// means that we have to do that here

type PaidInvoiceStorage struct {
	// PaidInvoiceStorage is an InvoiceReplyMsg with a timestamp
	// spinoff from InvoiceReplyMsg. We could ideally extend InvoiceReplyMsg for this
	// but it would seem wasteful since this has only a singula application
	PeerIdx   uint32
	Id        string
	CoinType  string
	Amount    uint64
	Timestamp string
}

func (self PaidInvoiceStorage) Bytes() []byte {
	var msg []byte
	msg = append(msg, lnutil.U32tB(self.PeerIdx)...)
	msg = append(msg, self.Id...)
	msg = append(msg, self.CoinType...)
	msg = append(msg, lnutil.U64tB(self.Amount)...)
	msg = append(msg, self.Timestamp...)
	return msg
}

// IIdFromBytes parses only the value from the key value pair that we
// get from the db. You need to set the peerIdx separately if you use this
// to load the value from the db
// this is a duplicate of IMsgFromBytes, but has a slightly different application,
// so we need to be careful
func IIdFromBytes(in []byte) lnutil.InvoiceMsg {
	var dummy lnutil.InvoiceMsg
	dummy.Id = string(in)
	return dummy
}

// in PaidInvoiceStorageFromBytes, we first slice off the timestamp (which is
// its own custom format to standardize length) followed by the peerIdx, invoiceId
// and then the amount followed by cointype (which is variable in length). The
// last paramter to be sliced off is the coinType
func PaidInvoiceStorageFromBytes(in []byte) PaidInvoiceStorage {
	inLength := len(in)                // incoming slice Length
	timestamp := in[inLength-19:]      // 19 = length of the timestamp format that we use to encode
	in = in[:inLength-19]              // cut from slice permanently
	peerIdx := lnutil.BtU32((in[0:4])) // cut peerIdx since its a uint32
	invoiceId := in[4]                 // single character invoiceId cut off
	in = in[5:]                        // cut the invoice and peeridx off
	inLength = len(in)                 // rsLength = remaining slice length
	amount := in[inLength-8:]          // cutoff the last 8 bytes for amount
	in = in[:inLength-8]               // the length of the timestamp is constant
	coinType := string(in)             // the rest of the characters belong to the coinType
	constructedMessage := PaidInvoiceStorage{
		PeerIdx:   peerIdx,
		Id:        string(invoiceId),
		CoinType:  coinType,
		Amount:    lnutil.BtU64(amount), // convert slice to uint64
		Timestamp: string(timestamp),    // convert bytestring to string
	}
	return constructedMessage
}
