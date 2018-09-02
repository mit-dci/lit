package invoice

import (
	"fmt"
	"github.com/mit-dci/lit/lnutil"
	"log"
)

// if you're sending an invoice out, you need to store amount, cointype, peerIdx and invoiceId

// if you received an invoice request, you need to see whether you have the invoice in
// the generated invoice bucket, send it out and delete it

// if you want to send out an invoice request, you need to store invoiceId, remotePeer address

// if you received a reply to your previously sent invoice request, you need to store it in
// some pending invoice struct and then ask the suer whether he wants to pay. If he
// does want to pay, you delete it from the pending bucket and move to the sent bucket.

// message type wise,
// there really is only a need for two types of messages
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

// BKTGeneratedInvoices -> should store InvoiceReplyMsg with Peeridx="60000"
// BKTSentInvoicesOut -> should store InvoiceMsg
// BKTSentInvoiceReq -> should store InvoiceReplyMsg
// BKTPendingInvoices -> should store InvoiceReplyMsg
// BKTPaidInvoices -> should store InvoiceReplyMsg

// so import everything from lnutil (and don't import this there)
// the Bytes() function is defined in lnutil, but the reverse isnt, which
// means that we have to do that here

type PaidInvoiceStorage struct {
	// use this struct in order to store the timestamps accurately
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

func InvoiceMsgFromBytes(in []byte) (lnutil.InvoiceMsg, error) {
	// InvoiceMsgFromBytes parses  only the value from the key value pair that we
	// get from the db. So you need to set the peerIdx separately if you use this
	// to load the value from the db at all
	var dummy lnutil.InvoiceMsg
	log.Println("calling InvoiceMsgFromBytes", string(in))
	dummy.Id = string(in)
	return dummy, nil
}
func InvoiceReplyMsgFromBytes(in []byte) (lnutil.InvoiceReplyMsg, error) {
	// the received merssage is something similar to [50 98 99 114 116 192 154 12]
	// for an invoice 0, 2, bcrt, 100000
	// But when reading throguh the byte slice, we do not know whether this is the
	// coinType or the amount. So..
	peerIdx := lnutil.BtU32((in[0:4]))
	invoiceId := in[4] // TODO: change this
	in = in[4:]        // cut the invoice and peeridx off
	// cutoff the last 8 bytes for the amount
	rsLength := len(in) // remaining Slice Length
	amount := in[rsLength-8:]
	coinType := string(in[1 : rsLength-8]) // should s tart from zero, edit
	// now we have the coutner at which to slice
	constructedMessage := lnutil.InvoiceReplyMsg{
		PeerIdx:  peerIdx, // why is this hardcoded?
		Id:       string(invoiceId),
		CoinType: coinType,
		Amount:   lnutil.BtU64(amount), // convert slice to uint64
	}
	fmt.Println("Decrypted message from bytes:", constructedMessage)
	return constructedMessage, nil
}

func PaidInvoiceStorageFromBytes(in []byte) (PaidInvoiceStorage, error) {
	log.Println("INCOMING", in)
	inL := len(in) // remaining Slice Length
	timestamp := in[inL-19:]
	in = in[:inL-19]
	peerIdx := lnutil.BtU32((in[0:4]))
	invoiceId := in[4]
	in = in[5:] // cut the invoice and peeridx off
	// cutoff the last 8 bytes for the amount
	rsL := len(in)
	amount := in[rsL-8:]
	in = in[:rsL-8]
	// the length of the timestamp is constant
	coinType := string(in)
	// now we have the coutner at which to slice
	constructedMessage := PaidInvoiceStorage{
		PeerIdx:   peerIdx, // why is this hardcoded?
		Id:        string(invoiceId),
		CoinType:  coinType,
		Amount:    lnutil.BtU64(amount), // convert slice to uint64
		Timestamp: string(timestamp),
	}
	return constructedMessage, nil
}
