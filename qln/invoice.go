package qln

import (
	"fmt"
	"log"
	//"math"
	"net"
	"regexp"
	"time"
	//"strconv"
	//"strings"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/crypto/fastsha256"
	invoice "github.com/mit-dci/lit/invoice"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

func IsBech32String(in string) bool {
	const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	var IsBech32 = regexp.MustCompile(`^[qpzry9x8gf2tvdw0s3jn54khce6mua7l]+$`).MatchString
	return IsBech32(in)
}

// GenInvoice generates an invoiceid and then stores the extra stuff in a map
// to send out to the peer in the case it is required in the future.
func (nd *LitNode) GenInvoiceId(cointype string, amount uint64) (string, error) {
	// so now we have to generate invoiceIds and store them in the database
	// InvoiceId string
	// CoinType string
	// Amount uint64
	var chars = "123456789abcdefghijklmnopqrstuvwxyz"
	i := 0
	free, err := nd.InvoiceManager.DoesKeyExist(invoice.BKTGeneratedInvoices, string(chars[i]))
	if err != nil {
		return "", err
	}
	for !free { // the invoice exists, so need a different invoiceId
		i++
		free, err = nd.InvoiceManager.DoesKeyExist(invoice.BKTGeneratedInvoices, string(chars[i]))
		if err != nil {
			return "", err
		}
	}
	log.Println("GOT A FREE INVOICEID", chars[i:i+1])
	return chars[i : i+1], nil
}

// type InvoiceReplyMsg struct {
// 	PeerIdx  uint32
// 	Id       string
// 	CoinType string
// 	Amount   uint64
// }

// type InvoiceMsg struct {
// 	PeerIdx uint32
// 	Id      string
// }

func (nd *LitNode) GetInvoiceReplyInfo(msg lnutil.InvoiceReplyMsg,
	peer *RemotePeer) (lnutil.InvoiceReplyMsg, error) {
	// so someone sent me details of their invoice, cool. If I requested that,
	// great I'll store it in my map. Else, random spammer trying to trick me,
	// so not going to store.
	// how do I know its not a random spammer? Check my sentInvoiceReq map. If
	// the guy is in that, great, lets just return conns and pay him. Else, spammer
	var dummy lnutil.InvoiceReplyMsg
	log.Println("RECV MSG", msg, peer.Idx)
	// change this such that it reads from the database
	// check the RequestedInvoices bucket
	savedInvoice, err := nd.InvoiceManager.LoadRequestedInvoice(peer.Idx, msg.Id)
	if err != nil {
		log.Println("Error while retrieving invocie from storage")
		return dummy, fmt.Errorf("Error while retrieving invocie from storage")
	}
	// pass the peeridx, chan id and receive a invoicemsg
	if savedInvoice.PeerIdx == peer.Idx && savedInvoice.Id == msg.Id {
		// this is an invoice we sent and and invoice that has the correct id
		msg.PeerIdx = peer.Idx // overwrite the remote node's peer id before
		// storing it in the database so taht litrpc can pick it up later
		log.Printf("Remote peer replied to an invoice %s. Storing in SavePendingInvoice", msg.Id)
		err := nd.InvoiceManager.SavePendingInvoice(&msg)
		if err != nil {
			return dummy, err
		}
		return msg, nil
	}
	return dummy, nil
}

func (nd *LitNode) GetInvoiceInfo(msg lnutil.InvoiceMsg, peer *RemotePeer) (lnutil.InvoiceReplyMsg, error) {
	// call the tracker to find the ip address
	// dial the address and then do a custom handshake with the peer to get the
	// invoice
	invoice, err := nd.InvoiceManager.LoadGeneratedInvoice(msg.Id)
	if err != nil {
		log.Println("Error while retrieving invoice. Exiting")
		return invoice, fmt.Errorf("Error while retrieving invoice. Exiting")
	}
	log.Printf("Peer %d requested invoice: %s", peer.Idx, invoice.Id)
	invoice.PeerIdx = peer.Idx
	var replyStr []byte
	replyStr = append(replyStr, []byte("H")...)
	replyStr = append(replyStr, invoice.Bytes()...)
	//testStr := []byte(reply.Id) // this should be the actual invoice preceeded by I
	_, err = peer.Con.Write(replyStr)
	if err != nil {
		log.Println("Error while retrieving invoice. Exiting!", err)
		return invoice, err
	}
	// now that we've sent out a message to the peer, we need to store it in
	// BKTRepliedInvoices so that we can keep track of whether this has been paid
	// or not
	err = nd.InvoiceManager.SaveRepliedInvoice(&msg)
	if err != nil {
		log.Println("Error while saving replied invoice", err)
		// no need to exit here since we can get paid even if this fails
	}

	// keep track of this invoice to see if we get paid sometime soon. One go routune
	// func (nd *LitNode) MonitorInvoice(peerIdx uint32, invoiceAmount uint64, coinType string) error {
	go nd.MonitorInvoice(invoice)
	// this is the invoicereplymsg that we get from generated invoices
	return invoice, nil
}

func (nd *LitNode) SplitInvoiceId(invoice string) (string, string, error) {
	// Invoice format:
	// An invoice majorly consists of three parts - the address you want to pay
	// to, and then a separator followed by the invoice identifier.
	// Everything else comes only after you dial up the peer
	// and then tell it about the invoice identifier
	// Once the remote node receives the invoice identifier, it needs to
	// send it the currency ticker and the amoutn in satoshi along with the
	// invoice identifier, the currency ticker as defined by SLIP0173 and the
	// amount in satoshis that the invoice is responsible for
	// Maximum amount will be fixed the same as lit (1G sat)
	// Currently supports only 35 simultaneous payments due to the single character
	// invoiceId- highly doubt if we need more than this.
	// note: currency identifiers are defined in accordance with SLIP0173
	// Amount Length Range: 5-9 (10000 - 100000000)
	// invoice

	// Max Invoice Length: 41+ 1 = 42
	maxInvoiceLength := 43 // 1 extra for the separator + 1 for the invoice
	// Min invoice Length: 21+1
	minInvoicelength := 23 // 1 extra for the separator + 1 for the invoice

	invoiceLength := len(invoice)
	if invoiceLength > maxInvoiceLength || invoiceLength < minInvoicelength {
		// having 1 as invoice length in order to check whether this works
		// error printed by appropriate handlers, no need to print it here.
		return "", "", fmt.Errorf("Invalid invoice length")
	}

	if invoice[(invoiceLength-2):invoiceLength-1] != "1" {
		// check whether the invoice has a valid invoice identifier
		log.Println("Contains spam data. Exiting")
		return "", "", fmt.Errorf("Invalid Invoice, doesn't contain identifier. Exiting")
	}

	invoiceId := invoice[invoiceLength-1]
	destAdr := invoice[0 : len(invoice)-2] // 111 + invoiceId
	// check if destAdr is valid here
	if !IsBech32String(destAdr[3:]) { // cut off the starting ln1
		log.Println("Payee address invalid. Quitting!")
	}
	log.Println("DEST AFDR", destAdr)
	return destAdr, string(invoiceId), nil
}

type CoinBalReply struct {
	CoinType    uint32
	SyncHeight  int32 // height this wallet is synced to
	ChanTotal   int64 // total balance in channels
	TxoTotal    int64 // all utxos
	MatureWitty int64 // confirmed, spendable and witness
	FeeRate     int64 // fee per byte
}

// GetBalancesOfCoinType gets the balance of the specific cointype. There's a small
// problem of how it works though, it gets only the cointypes of running coin daemons?
// got to sovle this or else inform user of this so he can restart the daemon or
// something
func (nd *LitNode) GetBalancesOfCoinType(checkCoinType uint32) (CoinBalReply, error) {
	var allTxos portxo.TxoSliceByAmt
	var empty CoinBalReply

	// get all channels
	qcs, err := nd.GetAllQchans()
	if err != nil {
		return empty, err
	}

	for cointype, wal := range nd.SubWallet {
		if cointype != checkCoinType {
			continue
		}
		// will add the balance for this wallet to the full reply
		var cbr CoinBalReply
		cbr.CoinType = cointype
		// get wallet height
		cbr.SyncHeight = wal.CurrentHeight()
		// also current fee rate
		cbr.FeeRate = wal.Fee()

		allTxos, err = wal.UtxoDump()
		if err != nil {
			return empty, err
		}

		// ask sub-wallet for balance
		cbr.TxoTotal = allTxos.Sum()
		cbr.MatureWitty = allTxos.SumWitness(cbr.SyncHeight)

		// iterate through channels to figure out how much we have
		for _, q := range qcs {
			if q.Coin() == cointype && !q.CloseData.Closed {
				cbr.ChanTotal += q.State.MyAmt
			}
		}
		return cbr, nil
	}
	return empty, fmt.Errorf("No coin daemon running, can't pay!")
}

func (nd *LitNode) InvoiceDial(invoiceRequester string) (RemotePeer, error) {
	var temp RemotePeer
	var err error
	// parse address and get pkh / host / port
	who, where := splitAdrString(invoiceRequester)

	// If we couldn't deduce a URL, look it up on the tracker
	if where == "" {
		where, _, err = Lookup(who, nd.TrackerURL, nd.ProxyURL)
		if err != nil {
			return temp, err
		}
	}

	// get my private ID key
	idPriv := nd.IdKey()

	// Assign remote connection
	newConn, err := lndc.Dial(idPriv, where, who, net.Dial)
	if err != nil {
		return temp, err
	}

	peerIdx, err := nd.GetPeerIdx(newConn.RemotePub(), newConn.RemoteAddr().String())
	if err != nil {
		return temp, err
	}

	// also retrieve their nickname, if they have one
	nickname := nd.GetNicknameFromPeerIdx(uint32(peerIdx))

	nd.RemoteMtx.Lock()
	temp.Con = newConn
	temp.Idx = peerIdx
	temp.Nickname = nickname
	nd.RemoteCons[peerIdx] = &temp
	nd.RemoteMtx.Unlock()
	// each connection to a peer gets its own LNDCReader
	go nd.LNDCReader(&temp)
	return temp, nil
}

type FundReply struct {
	Status     string
	ChanIdx    uint32
	FundHeight int32
}

type FundArgs struct {
	Peer        uint32 // who to make the channel with
	CoinType    uint32 // what coin to use
	Capacity    int64  // later can be minimum capacity
	Roundup     int64  // ignore for now; can be used to round-up capacity
	InitialSend int64  // Initial send of -1 means "ALL"
	Data        [32]byte
}

func (nd *LitNode) GetCoinBalances() (error) {
	// get coin balances of all running peers
	// is there a way to get the balances of coins whose daemons aren't running as well?
	// TODOX
	return nil
}
func (nd *LitNode) PayInvoice(req lnutil.InvoiceReplyMsg) (error) {

	// Two ways we can pay this invoice
	// 1. Pay through an existing channel ie push funds through an existing channel
	// 2. Two alternatives:
	// 		a. Open a channel directly with the peer and push the amount
	// 		b. Find a route to the peer using multi hop and then pay that user.
	//				i. But this is only if the guy who wants to pay has some channel open
	// 			 ii. If he doesn't, then we're better off opening a new channel with the remote peer directly

	// First, lets do 1.
	// Lets check balances before falling throguh to the channel code. It helps
	// save time if we know that there is no suitable channel that can pay the amount
	// After checking balances, check if the connection still exists. The remote peer
	// may have disconnected in the 3 seconds that we wait for it to respond. If
	// the connection is sitll alive, get teh appropriate channel id and then push
	// funds in that channel.

	// On 2,
	// Lets define Channel capacity to be 100 times the amount the user is planning
	// to send via the invoice. 100 times is reasonable? I guess..
	// If someone wants to pay the same guy for more than 100 times, just open a new
	// channel
	// We need to weigh between on chain fees for opening a new channel and  the multi
	// hop fees. As it stands right now, the multi hop fees is way lesser and as a
	// result, we see if we have ANY open channel and hten try multi hop through it.
	// if that doesn't work, fall back to the default case of opening a new channel.

	// this is the remotePeer struct defined elsewhere. Easy reference.
	// type RemotePeer struct {
	// 	Idx      uint32 // the peer index
	// 	Nickname string
	// 	Con      *lndc.Conn
	// 	QCs      map[uint32]*Qchan   // keep map of all peer's channels in ram
	// 	OpMap    map[[36]byte]uint32 // quick lookup for channels
	// }

	// Dial Peer
	// go through list of open channels
	// if there is a channel with this peer and we already have a connection, good
	// if there is a channel and we aren't connected, connect.
	// else do nothing, proceed to open a new channel

	// check if I have sufficient money to pay for this invoice

	var coinType uint32
	switch req.CoinType {
		// maybe we could have a map and then parse this stufff from there
		// but for now, this seems easiest
	case "tb":
		coinType = uint32(1) // the numbers are what we define, not SLIP stuff
	case "bcrt":
		coinType = uint32(257)
	case "tltc": // test litecoin
		coinType = uint32(65537)
	case "rltc": // regtest litecoin
		coinType = uint32(258)
	case "tvtc":
		coinType = uint32(65536)
	case "vtc":
		coinType = uint32(28)
	default:
		coinType = uint32(257)
	}
	balance, err := nd.GetBalancesOfCoinType(uint32(coinType))
	// get coinbalances of the daemon running
	// We could have some structure for storing other balances as well.
	// Need it for multi hop anyway?
	if err != nil {
		// either the daemon isn't running or some other weird error.
		return err
	}

	if balance.MatureWitty < int64(req.Amount) {
		// only witness balance since the cheapest option is to push funds through
		// an existing channel
		log.Println("Insufficient balance to pay this invoice")
		return fmt.Errorf("Insufficient balance to pay this invoice")
	}
	log.Printf("We have %d, paying %d from that", balance, req.Amount)

	conExists := false
	var empty [33]byte
	var rpx RemotePeer
	var qChannel *Qchan

	pubKey, _ := nd.GetPubHostFromPeerIdx(req.PeerIdx)
	if pubKey == empty {
		// we've reached the end of our list of peers which we ahve connected to
		// in the past. break.
		log.Println("no pubkey found. quitting!")
		return fmt.Errorf("no pubkey found. quitting!")
	}

	nd.RemoteMtx.Lock()
	_, connected := nd.RemoteCons[req.PeerIdx]
	// see if the peer is connected. Don't store the remote peer here.
	// skip checking rmeote addresses
	if connected {
		conExists = true
	}
	nd.RemoteMtx.Unlock()

	if !conExists {
		// we have to connect to them because we want to pay them
		// this case occrus when you sened out an invoice request, receive it but then
		// exit before you get to complete the payment. IN this canse, the payment is
		// still pending but with no connection
		idHash := fastsha256.Sum256(pubKey[:])
		adr := bech32.Encode("ln", idHash[:20])
		err := nd.DialPeer(adr + "@:2448") // TODOXX: Change this to a tracker absed lookup

		if err != nil {
			log.Printf("Could not connect to remote peerIdx %d", req.PeerIdx)
			log.Println(err)
			return err
		}
	}
	chanExists := false
	qcs, err := nd.GetAllQchans()
	if err != nil {
		return err
	}
	for _, q := range qcs {
		if q.KeyGen.Step[3]&0x7fffffff == req.PeerIdx && !q.CloseData.Closed &&
			q.State.MyAmt > int64(req.Amount) && q.Value > int64(req.Amount) {
			// get an open channel with required capacity
			// do we check for confirmation height as well?
			chanExists = true
			qChannel = q
		}
	}

	// check if chanExists
	if !chanExists {
		// First need to check for multi hop fees. Then need to check if we have funds
		// to open a new channel
		log.Println("we need to weigh the option between creating a new channel and multi hop")
		defaultFeePerByte := balance.FeeRate
		// Needs a good fee estimation RPC since fees may be wild
		log.Println("set fee for this coin is:", balance.FeeRate)
		avgTxSize := int64(200) // maybe have a function for this as well for accuracy

		balNeeded := req.Amount + uint64(defaultFeePerByte*avgTxSize) + uint64(consts.MinOutput) + uint64(balance.FeeRate*1000)
		// sum of required amount +  minOutput + justicetx Fee
		balHave := uint64(balance.MatureWitty)

		if balHave < balNeeded {
			log.Println("have insufficient witness amount, need to send on chain" +
				" or weight between sending via different coins")
			// TODO: check other coins' balances
		}

		var fundParams FundArgs
		var data [32]byte
		fundParams.Peer = req.PeerIdx
		fundParams.CoinType = coinType
		fundParams.Capacity = int64(100 * req.Amount)
		if int64(100*req.Amount) < consts.MinChanCapacity {
			fundParams.Capacity = consts.MinChanCapacity
		}
		fundParams.InitialSend = consts.MinOutput + balance.FeeRate*1000
		if fundParams.InitialSend > fundParams.Capacity {
			fundParams.Capacity = fundParams.InitialSend * 2
		}
		if fundParams.Capacity > int64(consts.MaxChanCapacity) {
			fundParams.Capacity = consts.MaxChanCapacity
		}
		fundParams.Data = data
		if fundParams.Capacity > int64(balHave) {
			return fmt.Errorf("Insufficient funds to start a new channel")
		}

		var err error
		if nd.InProg != nil && nd.InProg.PeerIdx != 0 {
			return fmt.Errorf("channel with peer %d not done yet", nd.InProg.PeerIdx)
		}

		if fundParams.Capacity > balance.MatureWitty-consts.SafeFee {
			return fmt.Errorf("Wanted %d but %d available for channel creation",
				fundParams.Capacity, balance.MatureWitty-consts.SafeFee)
		}

		fundParams.InitialSend = fundParams.InitialSend + int64(req.Amount)
		idx, err := nd.FundChannel(
			fundParams.Peer, fundParams.CoinType, fundParams.Capacity, fundParams.InitialSend, fundParams.Data)
		if err != nil {
			return err
		}
		log.Printf("Opened channel %d with peer %d", idx, req.PeerIdx)
		// Now we have a channel to push funds into, fall through to the case below
		// sleep a bit
		return nil
	}

	nd.RemoteMtx.Lock()
	rpx = *nd.RemoteCons[req.PeerIdx] // store the remote peer here
	nd.RemoteMtx.Unlock()

	log.Println("Pushing funds in existing / created channel")
	var data [32]byte
	qc, ok := rpx.QCs[qChannel.Idx()]
	if !ok {
		fmt.Printf("peer %d doesn't have channel %d",
			qChannel.Peer(), qChannel.Idx())
		return fmt.Errorf("peer %d doesn't have channel %d",
			qChannel.Peer(), qChannel.Idx())
	}
	qc.Height = qChannel.Height

	log.Println("Paying %d towards previously confirmed invoice", req.Amount)
	err = nd.PushChannel(qc, uint32(req.Amount), data)
	if err != nil {
		log.Println("ERROR WHILE PUSHING FUNDS!!", err)
		return err
	}
	// at this point, we've paid this peer and can store it in the lsit of paid invoices
	return nil
}

func (nd *LitNode) DeleteInvoicePayer(invoice lnutil.InvoiceReplyMsg) error {
	var err error
	BKTRequestedInvoices := []byte("RequestedInvoices")
	BKTPendingInvoices := []byte("PendingInvoices")
	err = nd.InvoiceManager.DeletePeerIdx(BKTRequestedInvoices, invoice.PeerIdx)
	// requested invoices are indexed by peeridx and not invoiceId
	if err != nil {
		log.Println("Couldn't delete val from Requested invocies db")
		return fmt.Errorf("Couldn't delete val from Requested invocies db")
	}
	err = nd.InvoiceManager.DeleteInvoiceId(BKTPendingInvoices, invoice.Id)
	// pending invoices are indexed by their invoiceIds rather than peerid's
	if err != nil {
		log.Println("Couldn't delete val from Pending invocies db")
		return fmt.Errorf("Couldn't delete val from Pending invocies db")
	}
	// deletion complete, we can safely return without any issues now
	// but the remote peer must delete its generated invoice and repliedinvoices
	// and store in its GotPaidInvoices Bucket
	return nil
}

func (nd *LitNode) DeleteInvoiceReceiver(invoice lnutil.InvoiceReplyMsg) error {
	// delete the invoice from GeneratedInvoices and RepliedInvoices and add it to
	// GotPaidInvoices so that we can keep a track of all invoices we got paid for
	var err error
	BKTGeneratedInvoices := []byte("GeneratedInvoices")
	BKTRepliedInvoices := []byte("RepliedInvoices")

	err = nd.InvoiceManager.DeleteInvoiceId(BKTGeneratedInvoices, invoice.Id)
	// generated invoices are indexed by invoiceId
	if err != nil {
		log.Println("Couldn't delete val from Generated invocies db")
		return fmt.Errorf("Couldn't delete val from Generated invocies db")
	}
	err = nd.InvoiceManager.DeletePeerIdx(BKTRepliedInvoices, invoice.PeerIdx)
	// replied invoices are indexed by their PeerIdx
	if err != nil {
		log.Println("Couldn't delete val from Replied invocies db")
		return fmt.Errorf("Couldn't delete val from Replied invocies db")
	}
	// deletion complete, we can safely return without any issues now
	// but the remote peer must delete its generated invoice and repliedinvoices
	// and store in its GotPaidInvoices Bucket
	return nil
}

func (nd *LitNode) MonitorInvoice(invoice lnutil.InvoiceReplyMsg) error {
	// MonitorInvoice will open a go routine that keeps track of all the payments
	// made by a specific user towards a receiver. in case this balance increases,
	// this payment is assumed to be part of the invoice. If the payment amount is less,
	// we mark the payment as failed and don't delete the invoice. If the payment is
	// more or equal we mark the payment as completed and delete the invoice.

	// this won't work if the remote peer disconnects after we reply to him
	// how do we solve this?
	// convenience handlers
	peerIdx := invoice.PeerIdx
	invoiceAmount := invoice.Amount
	coinType := invoice.CoinType

	var err error
	var qcs []*Qchan
	var currBalance, oldBalance uint64
	var coinTypeInt uint32
	switch coinType {
	case "tn3": // should be tb, check and fix
		coinTypeInt = uint32(1) // the numbers are what we define, not SLIP stuff
	case "bcrt":
		coinTypeInt = uint32(257)
	case "tltc": // test litecoin
		coinTypeInt = uint32(65537)
	case "rltc": // regtest litecoin
		coinTypeInt = uint32(258)
	case "tvtc":
		coinTypeInt = uint32(65536)
	case "vtc":
		coinTypeInt = uint32(28)
	default:
		coinTypeInt = 1 // tn3
	}

	qcs, err = nd.GetAllQchans()
	if err != nil {
		return err
	}

	for _, q := range qcs {
		if q.KeyGen.Step[3]&0x7fffffff == peerIdx && !q.CloseData.Closed && q.Coin() == coinTypeInt {
			// this is the peer we're looking for. We need to track balances for this guy
			oldBalance += uint64(q.State.MyAmt)
		}
	}

	currBalance = oldBalance
	for currBalance < oldBalance+invoiceAmount {
		// we need to get a list of all channels again because the peer may create
		// a new channel
		log.Println("Old Balance: %d", oldBalance)
		currBalance = 0
		// reset currentBalance because we calculate that again in each run
		qcs, err = nd.GetAllQchans()
		for _, q := range qcs {
			if q.KeyGen.Step[3]&0x7fffffff == peerIdx && !q.CloseData.Closed && q.Coin() == coinTypeInt {
				// this is the peer we're looking for. We need to track balances for this guy
				currBalance += uint64(q.State.MyAmt)
			}
		}
		time.Sleep(3 * time.Second) // 3s time polling interval to see if we got paid
		// is 3s too less? idk
	}
	// paid = true // need this bool for prompts maybe. else delete
	log.Printf("We got paid for invoice: %s", invoice.Id)
	// if we do come here, it means that we got paid, so delete the invoice from
	// generated invoices to free up the invoiceId for future use.
	err = nd.DeleteInvoiceReceiver(invoice)
	if err != nil {
		log.Println("Couldn't delete invoice from db, manually flush")
		// don't exit here since we already got paid
	}
	// now add this invoice to the GotPaidInvoices db soi that we can keep a track
	// of all those invoices we got paid for. This could also be used to alert the user
	// that he got paid simply by checking the invoice Id against the GotPaidInvoices db.
	// create an obejct of type PaidInvoiceStorage
	err = nd.InvoiceManager.SaveGotPaidInvoice(&invoice)
	if err != nil {
		log.Println("Couldn't save invoice to GotPaidInvoices")
		// don't exit here because this doesn't affect anything
		// but this would mean that we don't display the invoice to the user when he
		// wants to see the list of invoices he's paid.
	}
	// now delete the invoice from GeneratedInvoices, RepliedInvoices
	return nil
}
