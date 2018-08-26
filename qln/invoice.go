package qln

import (
	"fmt"
	"log"
	//"math"
	"net"
	"regexp"
	//"strconv"
	"strings"

	"github.com/mit-dci/lit/bech32"
	//"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

func (nd *LitNode) ReturnQchan() (*Qchan) {
	var q *Qchan
	return q
}
func IsBech32String(in string) bool {
	const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	var IsBech32 = regexp.MustCompile(`^[qpzry9x8gf2tvdw0s3jn54khce6mua7l]+$`).MatchString
	return IsBech32(in)
}

// GenInvoice generates an invoiceid and then stores the extra stuff in a map
// to send out to the peer in the case it is required in the future.
func GenInvoice() error {
	log.Println("Generating invoice for requested payment")
	return nil
}

func RetrieveInvoiceInfo() (lnutil.InvoiceReplyMsg, error) {
	// maybe have a custom data structure for invoices? idk
	log.Println("Retrieving invoice info from storage")
	// what we really have to do here on the invoice requester's side
	// is that we need to look through our storage and get the invoice details
	// related to the invoice identifier
	// Right now, we'll skip that and hardcode stuff to test stuff
	// 1bcrt100
	var msg lnutil.InvoiceReplyMsg
	msg.Id = "1"
	msg.CoinType = "bcrt"
	msg.Amount = 10000
	return msg, nil
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

func (nd *LitNode) DummyFunc() *RemotePeer {
	log.Println("WORKS?", nd.RemoteCons[4])
	return nd.RemoteCons[4]
}
func (nd *LitNode) GetInvoiceReplyInfo(msg lnutil.InvoiceReplyMsg,
	peer *RemotePeer) (lnutil.InvoiceReplyMsg, error) {
	// so someone sent me details of their invoice, cool. If I requested that,
	// great I'll store it in my map. Else, random spammer trying to trick me,
	// so not going to store.
	// how do I know its not a random spammer? Check my sentInvoiceReq map. If
	// the guy is in that, great, lets just return conns and pay him. Else, spammer
	var dummy lnutil.InvoiceReplyMsg
	log.Println("RECV MSG", msg, peer.Idx)
	for _, invoices := range nd.SentInvoiceReq {
		if invoices.PeerIdx == peer.Idx && invoices.Id == msg.Id {
			// this is an invoice we sent and and invoice that has the correct id
			log.Println("WORKSINSIDE?", nd.RemoteCons[4])
			log.Println("Both match, paying now")
			msg.PeerIdx = peer.Idx // overwrite the remote node's peer id before
			// passing it
			nd.PendingInvoiceReq = append(nd.PendingInvoiceReq, msg)
			// _, err := nd.PayInvoice(msg, peer)
			// if err != nil {
			// 	return dummy, err
			// }
			// call the pay handler here
		}
	}
	log.Println("LIST OF SENT INVOICES", nd.SentInvoiceReq)
	return dummy, nil
}

func (nd *LitNode) GetInvoiceInfo(msg lnutil.InvoiceMsg, peer *RemotePeer) (lnutil.InvoiceReplyMsg, error) {
	// call the tracker to find the ip address
	// dial the address and then do a custom handshake with the peer to get the
	// invoice
	log.Printf("Retrieving details for invoice id: %s requested by peer: %d", msg.Id, msg.PeerIdx)
	invoice, err := RetrieveInvoiceInfo()
	if err != nil {
		log.Println("Error while retrieving invoice. Exiting")
		return invoice, fmt.Errorf("Error while retrieving invoice. Exiting")
	}
	invoice.PeerIdx = peer.Idx
	var replyStr []byte
	replyStr = append(replyStr, []byte("H")...)
	replyStr = append(replyStr, invoice.Bytes()...)
	//testStr := []byte(reply.Id) // this should be the actual invoice preceeded by I
	_, err = peer.Con.Write(replyStr)
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
	// eg invoice:
	// ln1jad29zj4lq43klxyvs5eyx7cp656pnegjfamsg_1tb100000
	// address: 1
	// incoice identifier: 1
	// currency: tb
	// amount (in satoshi): 100000
	// Maximum amount will be fixed the same as lit (1G sat)
	// Length limits
	// Address length range: 21-41 (short addresses included)
	// Invoice number range: 1 (0-35)
	// Currently suppor only 35 simultaneous payments - highly doubt if we need
	// more than this concurrently.
	// Currency identifier range: 2-4 (tb, vtc, bcrt, etc)
	// note: currency identifiers are defined in accordance with SLIP0173
	// Amount Length Range: 5-9 (10000 - 100000000)
	// invoice

	// Max Invoice Length: 41+ 1 = 42
	// Next step after verifying ids: 1+4+9=14
	// (id + curr ticker + amount)
	// 9 = (int)(math.Log10(float64(consts.MaxChanCapacity))+1)
	maxInvoiceLength := 43 // 1 extra for the separator
	// Min invoice Length: 21+1
	// Next step after verifying ids: 1+2+5=8
	// (id + currency ticker + min sat amount)
	// 5 = (int)(math.Log10(float64(consts.MinSendAmt))+1)
	minInvoicelength := 23 // 1 extra for the separator

	log.Println(maxInvoiceLength, minInvoicelength)
	invoiceLength := len(invoice)
	if invoiceLength > maxInvoiceLength || invoiceLength < minInvoicelength {
		// having 1 as invoice length in order to check whether this works
		// error printed by appropriate handlers, no need to print it here.
		return "", "", fmt.Errorf("Invalid invoice length")
	}
	separatorPosition := strings.Index(invoice, "_")
	destAdr := invoice[0:separatorPosition]
	// check if destAdr is valid here
	if !IsBech32String(destAdr[3:]) { // cut off the starting ln1
		log.Println("Payee address invalid. Quitting!")
	}
	rightSeparator := invoice[separatorPosition+1 : invoiceLength]
	return destAdr, rightSeparator, nil
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
func (nd *LitNode) PayInvoiceHandler(peer *RemotePeer,
	invoiceAmount uint64, cointype string) error {
	// Now we have the destination address and the amount to be paid the that address
	// There are two ways we can pay this invoice
	// 1. Open a channel directly with the peer and push the amount
	// 2. Find a route to the peer using multi hop and then pay that user.
	// First, lets do 1.
	// In order to do 1, the minimum fund must be Minoutput + fee()*1000
	// with the default values that's 180k sat

	// Define Channel capacity to be 100 times the amount the user is planning
	// to send via the invoice. 100 times is reasonable? I guess..

	// type PeerInfo struct {
	// 	PeerNumber uint32
	//	RemoteHost string
	// 	LitAdr 	   string
	// 	Nickname   string
	// }

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

	chanExists := false
	var chanIdx uint32

	qcs, err := nd.GetAllQchans()
	if err != nil {
		return err
	}
	for _, q := range qcs {
		if q.KeyGen.Step[3]&0x7fffffff == peer.Idx && !q.CloseData.Closed &&
			uint64(q.Value) > invoiceAmount && uint64(q.State.MyAmt) > invoiceAmount {
			// we have a channel which is open, has capacity greater than the
			// invoice AMount and we have a balance greater than invoiceAmount
			log.Printf("Found a channel %d to push funds from", q.KeyGen.Step[4]&0x7fffffff)
			chanExists = true
			chanIdx = q.KeyGen.Step[4] & 0x7fffffff
			break
		}
	}

	if chanExists {
		var data [32]byte
		dummyqc, err := nd.GetQchanByIdx(chanIdx)
		if err != nil {
			return err
		}
		// map read, need mutex...?
		nd.RemoteMtx.Lock()
		peer1, ok := nd.RemoteCons[dummyqc.Peer()]
		nd.RemoteMtx.Unlock()
		if !ok {
			return fmt.Errorf("not connected to peer %d for channel %d",
				dummyqc.Peer(), dummyqc.Idx())
		}
		qc, ok := peer1.QCs[dummyqc.Idx()]
		if !ok {
			return fmt.Errorf("peer %d doesn't have channel %d",
				dummyqc.Peer(), dummyqc.Idx())
		}

		log.Printf("channel %s\n", qc.Op.String())

		if qc.CloseData.Closed {
			// check for one last time if the channel is closed
			return fmt.Errorf("Channel %d already closed by tx %s",
				chanIdx, qc.CloseData.CloseTxid.String())
		}

		qc.Height = dummyqc.Height
		err = nd.PushChannel(qc, uint32(invoiceAmount), data)
		if err != nil {
			log.Println("ERROR", err)
			return err
		}
	} else {
		// this case is the same as connecting to a random peer. But we know
		// this guy already. So just print some stuff and then fall through
		log.Println("we need to weigh the option between creating a new channel and multi hop")
	}

	// err := nd.DialPeer(destAdr)
	// if err != nil {
	// 	return err
	// }
	// Assign peer, cointype, capacity, initialsend, data and then
	// nd.FundChannel(args.Peer, args.CoinType, args.Capacity, args.InitialSend, args.Data)
	log.Println("Paying your invoice ultra securely")
	return nil
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

func (nd *LitNode) PayInvoice(invoiceMsg lnutil.InvoiceReplyMsg, peer *RemotePeer) (string, error) {
	// Now after this point, we know the invoice id, so we need to dial the peer
	// and then give the invoice id so that we recieve the other information
	// which we then collect
	// call GetInvoiceInfo()
	// 	PeerIdx  uint32
	// 	Id       string
	// 	CoinType string
	// 	Amount   uint64

	invoiceId := invoiceMsg.Id
	coinType := invoiceMsg.CoinType
	invoiceAmount := invoiceMsg.Amount
	log.Printf("Sending %d satoshi equivalent to peer: %s with invoice ID %d\n", invoiceAmount, invoiceId, invoiceId)

	// here we should ocnnect to hte peer to know details about the invoice
	// after cononecting to the guy, send him a InvoiceMsg telling him we have
	// na invoice id. Based on his repsonse, set all the values below.

	// write an invoiceMsg to the peer saying that you want details for a specific
	// invoice
	// You need to wait for the reply now in order to get details on the specific
	// invoice that you want to pay
	// But how?
	//log.Println("WRITTEN BYTES", bytesWritten)
	var coinTypeInt uint32
	switch coinType {
	case "tb":
		coinTypeInt = uint32(1) // the numbers are what we define, not SLIP stuff
	case "bcrt":
		coinTypeInt = uint32(257)
	default:
		coinTypeInt = uint32(257)
	}
	log.Println("COIN TYPE INT:", coinTypeInt)
	balance, err := nd.GetBalancesOfCoinType(uint32(coinTypeInt))
	if err != nil {
		return "", nil
	}

	// now have the balance of all running coin daemons. Need to check whether
	// amount + fees exist on our account to pay for the invoice
	defaultFeePerByte := balance.FeeRate // Replace this with a fee estimation RPC
	log.Println("Default fee for this coin is:", balance.FeeRate)
	avgTxSize := int64(200)

	balNeeded := invoiceAmount + uint64(defaultFeePerByte*avgTxSize)
	balHave := uint64(balance.MatureWitty)

	if balHave < balNeeded {
		log.Println("have insufficient witness amount, need to send on chain" +
			" or weight between sending via different coins")
		// TODO: check other coins' balances
	}
	log.Println("WE HAVE THE BALANCE TO PAY!!", balHave, balNeeded)
	// check for multiple coin balances here as well
	// might need rates for conversion, etc if we don't have required funds in one
	// coin
	// We have to
	// find someone who is willing to take our coin x and exchange it for
	// requested coin y with multi hop.

	// invoice stuff should end here, just pay using hte hadnler and then store
	//  the payment in PaidInvoiceReq once done
	err = nd.PayInvoiceHandler(peer, balNeeded, coinType)
	if err != nil {
		return "", err
	}
	return "0x00000000000000000000000000000000", nil
}

func (nd *LitNode) PayInvoiceBkp(req lnutil.InvoiceReplyMsg, destAdr string, invoice string) error {
	log.Println("paying for the invoice now")
	conExists := false
	var empty [33]byte
	var rpx RemotePeer
	var qChannel *Qchan

	pubKey, _ := nd.GetPubHostFromPeerIdx(req.PeerIdx)
	if pubKey == empty {
		// we've reached the end of our list of peers which we ahve connected to
		// in the past. break.
		log.Println("no pubkey found. quitting!")
		fmt.Errorf("no pubkey found. quitting!")
	}

	nd.RemoteMtx.Lock()
	_, connected := nd.RemoteCons[req.PeerIdx]
	nd.RemoteMtx.Unlock()

	idHash := fastsha256.Sum256(pubKey[:])
	adr := bech32.Encode("ln", idHash[:20])
	if adr == destAdr {
		log.Println("Addresses match")
		if connected {
			log.Println("We are connected to this peer")
			conExists = true
			rpx = *nd.RemoteCons[req.PeerIdx]
		}
	} else {
		log.Println("remote address doesn't match. quitting!")
		fmt.Errorf("remote address doesn't match. quitting!")
	}
	if !conExists {
		log.Println("Not connected to peer")
		return fmt.Errorf("Not connected to peer")
	}
	chanExists := false
	qcs, err := nd.GetAllQchans()
	if err != nil {
		return err
	}
	for _, q := range qcs {
		if q.KeyGen.Step[3]&0x7fffffff == req.PeerIdx && !q.CloseData.Closed {
			// this means I have / had a channel with him
			log.Println("We have / had a channel, this is cool")
			chanExists = true
			qChannel = q
		}
	}

	if chanExists {
		log.Println("Just push funds in the channel if it has capacity")
		var data [32]byte

		qc, ok := rpx.QCs[qChannel.Idx()]
		if !ok {
			return fmt.Errorf("peer %d doesn't have channel %d",
				qChannel.Peer(), qChannel.Idx())
		}
		qc.Height = qChannel.Height

		log.Println("Paying %d towards invoice %s", req.Amount, invoice)
		err := nd.PushChannel(qc, uint32(req.Amount), data)
		if err != nil {
			//qChannel.ClearToSend <- true
			log.Println("ERROR WHILE PUSHING FUNDS!!", err)
			return fmt.Errorf("ERROR WHILE PUSHING FUNDS!!")
		}
	} else {
		log.Println("we need to weigh the option between creating a new channel and multi hop")
	}
	return nil
}
