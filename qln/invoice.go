package qln

import (
	"fmt"
	"log"
	//"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/portxo"
)

func IsBech32String(in string) bool {
	const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	var IsBech32 = regexp.MustCompile(`^[qpzry9x8gf2tvdw0s3jn54khce6mua7l]+$`).MatchString
	return IsBech32(in)
}

// ParseInvoice parses the received invoice in step 2 after dialling the peer
// with the given invoiceid
func ParseInvoice(rightSeparator string) (string, string, uint64, error) {
	// invoiceId, coinType, uint64(amount), nil
	log.Println("Parsing invoiceid")

	// Give this invoice to people. Let them dial you back up to know more info
	// on this
	invoiceId := rightSeparator[:1]
	rightSeparator = rightSeparator[1:]

	var i int
	var chars rune
	var coinType string
	log.Println("ABSORBING chars", rightSeparator)
	for i, chars = range rightSeparator {
		if chars > 64 && chars < 123 { // 0 - 9
			// convert ascii to int
			coinType += string(int(chars))
			continue
		}
		break
	}
	rightSeparator = rightSeparator[i:]

	var amountStr string
	var tmp bool
	for i, chars = range rightSeparator {
		if chars > 47 && chars < 58 { // 0 - 9
			// convert ascii to int
			amountStr += string(int(chars))
			continue
		}
		tmp = true // if the thing in the last part is not a number,
		// means someone has attached spam to that and we must quit
		break
	}
	// rightSeparator = rightSeparator[i:] we don't bother about its value
	// since we already have amountStr
	if tmp {
		// the zero check is to cover for cases where rightSeparator is empty
		// but for some reason go wants to return 0 for that as well
		log.Println("Extra data added at the end. Invalid invoice!")
		return "", "", 0, fmt.Errorf("Extra data added. Invalid invoice!")
	}
	log.Printf("CoinType: %s, "+
		"Amount: %s", coinType, amountStr)

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return "", "", 0, fmt.Errorf("Error while parsing amounts")
	}
	if amount > consts.MaxChanCapacity {
		log.Println("Requested amount is greater than max channel capacity. Failed to send")
	}
	log.Println("Printing invoice id", invoiceId)
	return invoiceId, coinType, uint64(amount), nil
}

type InvoiceReply struct {
	Id       string // alphanumeric / bech32?
	CoinType string // SLIP0173 cointypes
	Amount   uint64 // Amoutn in uint64
}

// GenInvoice generates an invoiceid and then stores the extra stuff in a map
// to send out to the peer in the case it is required in the future.
func GenInvoice() error {
	log.Println("Generating invoice for requested payment")
	return nil
}

func RetrieveInvoiceInfo() (InvoiceReply, error) {
	// maybe have a custom data structure for invoices? idk
	log.Println("Retrieving invoice info from storage")
	// what we really have to do here on the invoice requester's side
	// is that we need to look through our storage and get the invoice details
	// related to the invoice identifier
	// Right now, we'll skip that and hardcode stuff to test stuff
	// 1bcrt100
	var reply InvoiceReply
	reply.Id = "1"
	reply.CoinType = "bcrt"
	reply.Amount = 100
	return reply, nil
}
func GetInvoiceInfo(destAdr string) (InvoiceReply, error) {
	// call the tracker to find the ip address
	// dial the address and then do a custom handshake with the peer to get the
	// invoice
	invoice, err := RetrieveInvoiceInfo()
	if err != nil {
		log.Println("Error while retrieving invoice. Exiting")
		return invoice, fmt.Errorf("Error while retrieving invoice. Exiting")
	}
	log.Println("INVOICE:", invoice)
	return invoice, nil
}

func SplitInvoiceId(invoice string) (string, string, error) {
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
func (nd *LitNode) PayInvoiceHandler(destAdr string, invoiceAmount uint64, cointype string) error {
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

	connectedEarlier := false
	conExists := false
	localPeerIdx := uint32(0)
	var rp RemotePeer
	ctr := uint32(1)
	var empty [33]byte

	log.Println("COINTYPE", cointype)
	for {
		pubKey, _ := nd.GetPubHostFromPeerIdx(ctr)
		if pubKey == empty {
			// we've reached the end of our list of peers which we ahve connected to
			// in the past. break.
			break
		}

		nd.RemoteMtx.Lock()
		_, connected := nd.RemoteCons[ctr]
		nd.RemoteMtx.Unlock()

		idHash := fastsha256.Sum256(pubKey[:])
		adr := bech32.Encode("ln", idHash[:20])
		if adr == destAdr {
			log.Println("We have the address in our past history")
			localPeerIdx = ctr
			connectedEarlier = true
			if connected {
				log.Println("We are connected to this peer")
				conExists = true
				rp = *nd.RemoteCons[ctr]
			}
			break
		}
		ctr++
	}
	chanExists := false
	var qChannel *Qchan
	if connectedEarlier { // || conExists is omitted here
		qcs, err := nd.GetAllQchans()
		if err != nil {
			return err
		}
		for _, q := range qcs {
			if q.KeyGen.Step[3]&0x7fffffff == localPeerIdx && !q.CloseData.Closed {
				// this means I have / had a channel with him
				log.Println("We have / had a channel, this is cool")
				chanExists = true
				qChannel = q
			}
		}
	}

	if chanExists {
		if !conExists {
			// connect and see what to do
			log.Println("Channel exists but we aren't connected. So connecting.")
			conExists = true
		}
		log.Println("Just push funds in the channel if it has capacity")
		var data [32]byte

		qc, ok := rp.QCs[qChannel.Idx()]
		if !ok {
			return fmt.Errorf("peer %d doesn't have channel %d",
				qChannel.Peer(), qChannel.Idx())
		}
		qc.Height = qChannel.Height

		err := nd.PushChannel(qc, uint32(invoiceAmount), data)
		if err != nil {
			qChannel.ClearToSend <- true
			log.Println("err", err)
			log.Println("ERROR WHILE PUSHING FUNDS!!")
			return fmt.Errorf("ERROR WHILE PUSHING FUNDS!!")
		}
		// get a list of all channels
		// check which one ahs capacity and can be closed safely
		// push funds in that
		log.Println("Active connection:", rp)
		// in case no single channel has capacity, push funds through
		// multiple channles is possible. Else just break
	} else if connectedEarlier && !chanExists {
		// this case is the same as connectign to a random peer. But we know
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
func (nd *LitNode) PayInvoice(invoice string) (string, error) {
	// log.Println("Calls litrpc. Cool, QUitting")
	//destAdr, invoiceAmount, invoiceId, err := SplitInvoiceId(invoice)
	destAdr, invoiceId, err := SplitInvoiceId(invoice)
	if err != nil {
		return "", err
	}
	log.Printf("Parsed invoice with destination address: %s and invoice id: %s", destAdr, invoiceId)
	// Now after this point, we know the invoice id, so we need to dial the peer
	// and then gice the invoice id so that we recieve the other infromation
	// which we then collect
	// call GetInvoiceInfo()
	invoiceId, coinType, invoiceAmount, err := ParseInvoice("1bcrt100")
	log.Printf("parsed invoiceId: %s, coinType: %s, invoice Amoint: %d ",
		invoiceId, coinType, invoiceAmount)

	invoiceDetails, err := GetInvoiceInfo(destAdr)
	if err != nil {
		return "", err
	}
	log.Println("INVOICE DETAILS", invoiceDetails)
	//log.Printf("Sending %d satoshi equivalent to address: %s with invoice ID %d\n", invoiceAmount, destAdr, invoiceId)
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

	log.Println(invoiceDetails.Amount, defaultFeePerByte*avgTxSize)
	balNeeded := invoiceDetails.Amount + uint64(defaultFeePerByte*avgTxSize)
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
	log.Printf("Paying invoice: %s with destination address: %s, amount: %d"+
		" and cointype: %s", invoice, destAdr, balNeeded, coinType)
	err = nd.PayInvoiceHandler(destAdr, balNeeded, invoiceDetails.CoinType)
	if err != nil {
		return "", err
	}
	return "0x00000000000000000000000000000000", nil
}
