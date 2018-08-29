package qln

import (
	"fmt"
	"log"
	//"math"
	"net"
	"regexp"
	//"strconv"
	//"strings"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/crypto/fastsha256"
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
	// so now we have to generate invoiceIds and store them in GenInvoiceReq
	// InvoiceId string
	// CoinType string
	// Amount uint64
	var chars = "123456789abcdefghijklmnopqrstuvwxyz"
	_, exists := nd.GenInvoiceReq[chars[0:1]]
	i := 0
	for exists { // the invoice exists, so need a different invoiceId
		i++
		_, exists = nd.GenInvoiceReq[chars[i:i+1]]
	}
	return chars[i : i+1], nil
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
			msg.PeerIdx = peer.Idx // overwrite the remote node's peer id before
			// storing it in PendingInvoiceReq so that litrpc can take it up
			nd.PendingInvoiceReq = append(nd.PendingInvoiceReq, msg)
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
	maxInvoiceLength := 43 // 1 extra for the separator + 1 for the invoice
	// Min invoice Length: 21+1
	// Next step after verifying ids: 1+2+5=8
	// (id + currency ticker + min sat amount)
	// 5 = (int)(math.Log10(float64(consts.MinSendAmt))+1)
	minInvoicelength := 23 // 1 extra for the separator + 1 for the invoice

	invoiceLength := len(invoice)
	if invoiceLength > maxInvoiceLength || invoiceLength < minInvoicelength {
		// having 1 as invoice length in order to check whether this works
		// error printed by appropriate handlers, no need to print it here.
		return "", "", fmt.Errorf("Invalid invoice length")
	}

	if invoice[(invoiceLength-2):invoiceLength-1] != "1"  {
		// check whether the invoice has a valid invoice identifier
		log.Println("Contains spam data. Exiting")
		return "", "", fmt.Errorf("Invalid Invoice, doesn't contain identifier. Exiting")
	}

	invoiceId := invoice[invoiceLength-1]
	destAdr := invoice[0:len(invoice)-4] // 111 + invoiceId
	// check if destAdr is valid here
	if !IsBech32String(destAdr[3:]) { // cut off the starting ln1
		log.Println("Payee address invalid. Quitting!")
	}

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

func (nd *LitNode) PayInvoiceBkp(req lnutil.InvoiceReplyMsg,
	destAdr string, invoice string) (uint64, error) {

	// Two ways we can pay this invoice
	// 1. Pay through an existing channel ie push funds through an existing channel
	// 2. Two alternatives:
	// 		a. Open a channel directly with the peer and push the amount
	// 		b. Find a route to the peer using multi hop and then pay that user.
	//				i. But this is only if the guy who wants to pay has some channel open
	// 			 ii. If he doesn't, then we're better off opening a new channel with him?

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

	log.Println("paying for the invoice now")
	// check if I have sufficient  money to pay for this invoice

	var coinType uint32
	switch req.CoinType {
	case "tb":
		coinType = uint32(1) // the numbers are what we define, not SLIP stuff
	case "bcrt":
		coinType = uint32(257)
	default:
		coinType = uint32(257)
	}
	balance, err := nd.GetBalancesOfCoinType(uint32(coinType))
	// get coinbalances of the daemon running
	// We could have some structure for storing other balances as well.
	// Need it for multi hop anyway?
	if err != nil {
		// either the daemon isn't running or some other weird error.
		return 0, nil
	}

	if balance.MatureWitty < int64(req.Amount) {
		// only witness balance since the cheapest option is to push funds through
		// an existing channel
		log.Println("Insufficient balance to pay this invoice")
		return 0, fmt.Errorf("Insufficient balance to pay this invoice")
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
		return 0, fmt.Errorf("no pubkey found. quitting!")
	}

	nd.RemoteMtx.Lock()
	_, connected := nd.RemoteCons[req.PeerIdx]
	// see if the peer is connected. Don't store the remote peer here.
	nd.RemoteMtx.Unlock()

	idHash := fastsha256.Sum256(pubKey[:])
	adr := bech32.Encode("ln", idHash[:20])
	if adr == destAdr { // sanity check regarding addresses
		log.Println("Addresses match")
		if connected {
			log.Println("We are connected to this peer")
			conExists = true
			rpx = *nd.RemoteCons[req.PeerIdx] // store the remote peer here
		}
	} else {
		// the remote PKH doesn't match what we have, some weird case. exit
		log.Println("remote address doesn't match. quitting!")
		return 0, fmt.Errorf("remote address doesn't match. quitting!")
	}
	if !conExists {
		// per disconnected in the 3s that we waited. No money for them.
		log.Println("Not connected to peer, not paying invoice!")
		return 0, fmt.Errorf("Not connected to peer, not paying invoice!")
	}
	chanExists := false
	qcs, err := nd.GetAllQchans()
	if err != nil {
		return 0, err
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
			return 0, fmt.Errorf("Insufficient funds to start a new channel")
		}

		var err error
		if nd.InProg != nil && nd.InProg.PeerIdx != 0 {
			return 0, fmt.Errorf("channel with peer %d not done yet", nd.InProg.PeerIdx)
		}

		if fundParams.Capacity > balance.MatureWitty-consts.SafeFee {
			return 0, fmt.Errorf("Wanted %d but %d available for channel creation",
				fundParams.Capacity, balance.MatureWitty-consts.SafeFee)
		}

		fundParams.InitialSend = fundParams.InitialSend + int64(req.Amount)
		idx, err := nd.FundChannel(
			fundParams.Peer, fundParams.CoinType, fundParams.Capacity, fundParams.InitialSend, fundParams.Data)
		if err != nil {
			return 0, err
		}
		log.Printf("Opened channel %d with peer %d", idx, req.PeerIdx)
		// Now we have a channel to push funds into, fall through to the case below
		// sleep a bit
		return uint64(idx), nil
	}
	log.Println("Pushinf funds in existing / created channel")
	var data [32]byte

	qc, ok := rpx.QCs[qChannel.Idx()]
	if !ok {
		return 0, fmt.Errorf("peer %d doesn't have channel %d",
			qChannel.Peer(), qChannel.Idx())
	}
	qc.Height = qChannel.Height

	log.Println("Paying %d towards invoice %s", req.Amount, invoice)
	err = nd.PushChannel(qc, uint32(req.Amount), data)
	if err != nil {
		log.Println("ERROR WHILE PUSHING FUNDS!!", err)
		return 0, fmt.Errorf("ERROR WHILE PUSHING FUNDS!!")
	}

	return qc.State.StateIdx, nil
}
