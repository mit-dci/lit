package invoice

import (
	"bytes"
	"encoding/binary"
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
func InvoiceMsgFromBytes(in []byte) (lnutil.InvoiceReplyMsg, error) {
	// the received merssage is something similar to [50 98 99 114 116 192 154 12]
	// for an invoice 0, 2, bcrt, 100000
	// But when reading throguh the byte slice, we do not know whether this is the
	// coinType or the amount. So..
	invoiceId := in[0]
	in = in[1:] // cut the invoice off
	var i int
	for i = len(in) - 1; i >= 0; i-- {
		if in[i] > 60 { // rough limit for int byte chars
			break
		}
	}
	var dummy lnutil.InvoiceReplyMsg
	amtSlice := in[i-1:]
	buf := bytes.NewBuffer(amtSlice)
	amount, err := binary.ReadVarint(buf)
	if err != nil {
		return dummy, fmt.Errorf("Unable to convert amount to uint64")
	}
	log.Println(amount)
	// now we have the coutner at which to slice
	constructedMessage := lnutil.InvoiceReplyMsg{
		PeerIdx:  uint32(60000),
		Id:       string(invoiceId),
		CoinType: string(in[0 : i-1]),
		Amount:   uint64(amount), // convery slice to uint64
	}
	log.Println("CONSTR", constructedMessage)
	return constructedMessage, nil
}
