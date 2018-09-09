package litrpc

import (
	//"bufio"
	"fmt"
	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/consts"
	//"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/wire"
	"log"
	//"os"
	//"strings"
	//"time"
	invoice "github.com/mit-dci/lit/invoice"

	qrcode "github.com/skip2/go-qrcode"
)

type TxidsReply struct {
	Txids []string
}
type StatusReply struct {
	Status string
}

type NoArgs struct {
	// nothin
}

type CoinArgs struct {
	CoinType uint32
}

// ------------------------- balance
// BalReply is the reply when the user asks about their balance.
type CoinBalReply struct {
	CoinType    uint32
	SyncHeight  int32 // height this wallet is synced to
	ChanTotal   int64 // total balance in channels
	TxoTotal    int64 // all utxos
	MatureWitty int64 // confirmed, spendable and witness
	FeeRate     int64 // fee per byte
}

type BalanceReply struct {
	Balances []CoinBalReply
}

func (r *LitRPC) Balance(args *NoArgs, reply *BalanceReply) error {

	var allTxos portxo.TxoSliceByAmt

	// get all channels
	qcs, err := r.Node.GetAllQchans()
	if err != nil {
		return err
	}

	for cointype, wal := range r.Node.SubWallet {
		// will add the balance for this wallet to the full reply
		var cbr CoinBalReply

		cbr.CoinType = cointype
		// get wallet height
		cbr.SyncHeight = wal.CurrentHeight()
		// also current fee rate
		cbr.FeeRate = wal.Fee()

		allTxos, err = wal.UtxoDump()
		if err != nil {
			return err
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

		// I thought slices were pointery enough that I could put this line
		// near the top.  Guess not.
		reply.Balances = append(reply.Balances, cbr)
	}
	return nil
}

type TxoInfo struct {
	OutPoint string
	Amt      int64
	Height   int32
	Delay    int32
	CoinType string
	Witty    bool

	KeyPath string
}

type TxoListReply struct {
	Txos []TxoInfo
}

// TxoList sends back a list of all non-channel utxos
func (r *LitRPC) TxoList(args *NoArgs, reply *TxoListReply) error {

	for _, wal := range r.Node.SubWallet {

		walTxos, err := wal.UtxoDump()
		if err != nil {
			return err
		}

		syncHeight := wal.CurrentHeight()

		theseTxos := make([]TxoInfo, len(walTxos))
		for i, u := range walTxos {
			theseTxos[i].OutPoint = u.Op.String()
			theseTxos[i].Amt = u.Value
			theseTxos[i].Height = u.Height
			theseTxos[i].CoinType = wal.Params().Name
			// show delay before utxo can be spent
			if u.Seq != 0 {
				theseTxos[i].Delay = u.Height + int32(u.Seq) - syncHeight
			}
			theseTxos[i].Witty = u.Mode&portxo.FlagTxoWitness != 0
			theseTxos[i].KeyPath = u.KeyGen.String()
		}

		reply.Txos = append(reply.Txos, theseTxos...)
	}
	return nil
}

// ------------------------- send
type SendArgs struct {
	DestAddrs []string
	Amts      []int64
}

func (r *LitRPC) Send(args SendArgs, reply *TxidsReply) error {
	var err error

	nOutputs := len(args.DestAddrs)
	if nOutputs < 1 {
		return fmt.Errorf("No destination address specified")
	}
	if nOutputs != len(args.Amts) {
		return fmt.Errorf("%d addresses but %d amounts specified",
			nOutputs, len(args.Amts))
	}
	// get cointype for first address.
	coinType := CoinTypeFromAdr(args.DestAddrs[0])
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[coinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for address %s type %d",
			args.DestAddrs[0], coinType)
	}
	// All addresses must have the same cointype as they all
	// must to be in the same tx.
	for _, a := range args.DestAddrs {
		if CoinTypeFromAdr(a) != coinType {
			return fmt.Errorf("Coin type mismatch for address %s, %s",
				a, args.DestAddrs[0])
		}
	}

	txOuts := make([]*wire.TxOut, nOutputs)
	for i, s := range args.DestAddrs {
		if args.Amts[i] < consts.MinSendAmt {
			return fmt.Errorf("Amt %d less than minimum send amount %d", args.Amts[i], consts.MinSendAmt)
		}

		outScript, err := AdrStringToOutscript(s)
		if err != nil {
			return err
		}

		txOuts[i] = wire.NewTxOut(args.Amts[i], outScript)
	}

	// we don't care if it's witness or not
	ops, err := wal.MaybeSend(txOuts, false)
	if err != nil {
		return err
	}

	err = wal.ReallySend(&ops[0].Hash)
	if err != nil {
		return err
	}

	reply.Txids = append(reply.Txids, ops[0].Hash.String())
	return nil
}

// ------------------------- sweep
type SweepArgs struct {
	DestAdr string
	NumTx   uint32
	Drop    bool
}

func (r *LitRPC) Sweep(args SweepArgs, reply *TxidsReply) error {
	// get cointype for first address.
	coinType := CoinTypeFromAdr(args.DestAdr)
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[coinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for address %s type %d",
			args.DestAdr, coinType)
	}

	outScript, err := AdrStringToOutscript(args.DestAdr)
	if err != nil {
		return err
	}

	log.Printf("numtx: %d\n", args.NumTx)
	if args.NumTx < 1 {
		return fmt.Errorf("can't send %d txs", args.NumTx)
	}

	txids, err := wal.Sweep(outScript, args.NumTx)
	if err != nil {
		return err
	}

	for _, txid := range txids {
		reply.Txids = append(reply.Txids, txid.String())
	}

	return nil
}

// ------------------------- fanout
type FanArgs struct {
	DestAdr      string
	NumOutputs   uint32
	AmtPerOutput int64
}

func (r *LitRPC) Fanout(args FanArgs, reply *TxidsReply) error {
	if args.NumOutputs < 1 {
		return fmt.Errorf("Must have at least 1 output")
	}
	if args.AmtPerOutput < 5000 {
		return fmt.Errorf("Minimum 5000 per output")
	}

	// get cointype for first address.
	coinType := CoinTypeFromAdr(args.DestAdr)
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[coinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for address %s type %d",
			args.DestAdr, coinType)
	}

	outScript, err := AdrStringToOutscript(args.DestAdr)
	if err != nil {
		return err
	}

	txos := make([]*wire.TxOut, args.NumOutputs)

	for i, _ := range txos {
		txos[i] = new(wire.TxOut)
		txos[i].Value = args.AmtPerOutput + int64(i)
		txos[i].PkScript = outScript
	}

	// don't care if inputs are witty or not
	ops, err := wal.MaybeSend(txos, false)
	if err != nil {
		return err
	}
	err = wal.ReallySend(&ops[0].Hash)
	if err != nil {
		return err
	}

	reply.Txids = append(reply.Txids, ops[0].String())
	return nil
}

// set fee
type SetFeeArgs struct {
	Fee      int64
	CoinType uint32
}

// get fee
type FeeArgs struct {
	CoinType uint32
}
type FeeReply struct {
	CurrentFee int64
}

// SetFee allows you to set a fee rate for a wallet.
func (r *LitRPC) SetFee(args *SetFeeArgs, reply *FeeReply) error {
	// if cointype is 0, use the node's default coin
	if args.CoinType == 0 {
		args.CoinType = r.Node.DefaultCoin
	}
	if args.Fee < 0 {
		return fmt.Errorf("Invalid value for SetFee: %d", args.Fee)
	}
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[args.CoinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for coin type %d", args.CoinType)
	}
	reply.CurrentFee = wal.SetFee(args.Fee)
	return nil
}

// Fee gets the fee rate for a wallet.
func (r *LitRPC) GetFee(args *FeeArgs, reply *FeeReply) error {
	// if cointype is 0, use the node's default coin
	if args.CoinType == 0 {
		args.CoinType = r.Node.DefaultCoin
	}
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[args.CoinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for coin type %d", args.CoinType)
	}
	reply.CurrentFee = wal.Fee()
	return nil
}

// ------------------------- address
type AddressArgs struct {
	NumToMake uint32
	CoinType  uint32
}

// TODO Make this contain an array of structures not a structure of arrays.
type AddressReply struct {
	CoinTypes       []uint32
	WitAddresses    []string
	LegacyAddresses []string
}

func (r *LitRPC) Address(args *AddressArgs, reply *AddressReply) error {
	var allAdr [][20]byte
	var ctypesPerAdr []uint32

	// if cointype is 0, use the node's default coin
	if args.CoinType == 0 {
		args.CoinType = r.Node.DefaultCoin
	}

	// If you tell it to make 0 new addresses, it sends a list of all the old ones
	// (from every wallet)
	if args.NumToMake == 0 {
		// this gets 20 byte addresses; need to convert them to bech32 / base58
		// iterate through every wallet
		for cointype, wal := range r.Node.SubWallet {
			walAdr, err := wal.AdrDump()
			if err != nil {
				return err
			}

			for _, _ = range walAdr {
				ctypesPerAdr = append(ctypesPerAdr, cointype)
			}
			allAdr = append(allAdr, walAdr...)
		}
	} else {
		// if you have non-zero NumToMake, then cointype matters
		wal, ok := r.Node.SubWallet[args.CoinType]
		if !ok {
			return fmt.Errorf("No wallet of cointype %d linked", args.CoinType)
		}

		// call NewAdr a bunch of times
		remaining := args.NumToMake
		for remaining > 0 {
			adr, err := wal.NewAdr()
			if err != nil {
				return err
			}
			allAdr = append(allAdr, adr)
			ctypesPerAdr = append(ctypesPerAdr, args.CoinType)
			remaining--
		}
	}

	reply.CoinTypes = make([]uint32, len(allAdr))
	reply.WitAddresses = make([]string, len(allAdr))
	reply.LegacyAddresses = make([]string, len(allAdr))

	for i, a := range allAdr {

		// Store the cointype
		reply.CoinTypes[i] = ctypesPerAdr[i]

		// convert 20 byte array to old address
		param := r.Node.SubWallet[ctypesPerAdr[i]].Params()

		oldadr := lnutil.OldAddressFromPKH(a, param.PubKeyHashAddrID)
		reply.LegacyAddresses[i] = oldadr

		// convert 20-byte PKH to a bech32 segwit v0 address
		bech32adr, err := bech32.SegWitV0Encode(param.Bech32Prefix, a[:])

		if err != nil {
			return err
		}
		reply.WitAddresses[i] = bech32adr
	}

	return nil
}

// More human-readable replies
func (r *LitRPC) GetAddresses(args *NoArgs, reply *AddressReply) error {

	// return index
	ri := 0

	cts := make([]uint32, 0)
	was := make([]string, 0)
	las := make([]string, 0)

	for cointype, wal := range r.Node.SubWallet {

		walAdr, err := wal.AdrDump()
		if err != nil {
			panic("this should never happen, I don't think")
		}

		for _, pubkey := range walAdr {

			param := r.Node.SubWallet[cointype].Params()
			cts = append(cts, cointype)

			b32, err := bech32.SegWitV0Encode(param.Bech32Prefix, pubkey[:])
			if err != nil {
				panic("error encoding bech32 address")
			}

			was = append(was, b32)
			las = append(las, lnutil.OldAddressFromPKH(pubkey, param.PubKeyHashAddrID))

			ri++
		}

	}

	reply.CoinTypes = cts
	reply.WitAddresses = was
	reply.LegacyAddresses = las

	return nil
}

//func oldAddressPubKeyHash(pkHash []byte, netID byte) (string, error) {
//	// Check for a valid pubkey hash length.
//	if len(pkHash) != ripemd160.Size {
//		return "", errors.New("pkHash must be 20 bytes")
//	}
//	return base58.CheckEncode(pkHash, netID), nil
//}

type ClaimHTLCArgs struct {
	R [16]byte
}

func (r *LitRPC) ClaimHTLC(args *ClaimHTLCArgs, reply *TxidsReply) error {
	txids, err := r.Node.ClaimHTLC(args.R)
	if err != nil {
		return err
	}

	reply.Txids = make([]string, 0)
	for _, txid := range txids {
		reply.Txids = append(reply.Txids, fmt.Sprintf("%x", txid))
	}

	return nil
}

// Gen Invoice
type GenInvoiceArgs struct {
	CoinType string
	Amount   uint64
}

type GenInvoiceReply struct {
	Invoice string
}

// Pay Invoice
type PayInvoiceArgs struct {
	Invoice string
}

type PayInvoiceReply struct {
	Txid     string
	StateIdx uint64
}

type PayInvoiceHandlerReply struct {
	Invoices []lnutil.InvoiceReplyMsg
}

func (r *LitRPC) GenInvoice(args *GenInvoiceArgs, reply *GenInvoiceReply) error {
	idPriv := r.Node.IdKey()
	var idPub [33]byte
	copy(idPub[:], idPriv.PubKey().SerializeCompressed())
	adr := lnutil.LitAdrFromPubkey(idPub)
	// now we have the listening address that the peer has to connect to
	// generate Invoice Id here
	// general stuff - it can be any single byte character
	// but restricting to alphanumeric. That gives us 36 concurrent payments
	// which is still good (36 / 3  = 12 tps) (3 s is for the wait delay in response)

	// so we need a new invoices.db file
	invoiceId, err := r.Node.GenInvoiceId(args.CoinType, args.Amount)
	if err != nil {
		return err
	}

	// check here if passed cointype really exists or not
	existingCoins := [...]string{"tb", "bcrt"}
	// find a nicer way to collect this information
	validCoin := false
	for _, coin := range existingCoins {
		if args.CoinType == coin {
			validCoin = true
		}
	}
	if !validCoin {
		return fmt.Errorf("Coin not yet supported. Add support!")
	}
	log.Printf("Generated invoice: %s1%s", adr, invoiceId)
	// 1 is the identifier
	// store this generated invoice in the db that we have
	var invoiceStorage lnutil.InvoiceReplyMsg
	invoiceStorage.CoinType = args.CoinType
	invoiceStorage.Amount = args.Amount
	invoiceStorage.PeerIdx = uint32(60000) // some random peerid for generated
	// invoices since we don't need them. Could leave them empty, but that might
	// affect error handling stuff later down the road somewhere. Something
	// to visit at the end I guess
	invoiceStorage.Id = invoiceId

	err = r.Node.InvoiceManager.SaveGeneratedInvoice(&invoiceStorage)
	if err != nil {
		return err
	}
	// generate a qr code for the invoice so that people cna print it out or something
	qrString := fmt.Sprintf("%s1%s", adr, invoiceId)
	err = qrcode.WriteFile(qrString, qrcode.Highest, -1, "qr_"+invoiceId+".png")
	// 30% error recovery qr codes, could maybe decrease it, but better to have this
	// in case some part of a qr gets burnt or something
	// create a qr code in png format
	if err != nil {
		log.Println("Error while generating a qr code")
		// don't return this error since its an addition
	}
	// If size is too small then a larger image is silently written
	// usign a negative value for variable sized images
	reply.Invoice = qrString
	return nil
}

func (r *LitRPC) PayInvoiceHandler(args *NoArgs, reply *PayInvoiceHandlerReply) error {
	// we need to go through all the endpoitns present in BKTRequestedInvoices and then
	// see whether they are in BKTPendingInvoices. If tehy are in pending invoices,
	// we also need to send the user a message asking him if he wants to pay the
	// particular invoice
	// the problem with this endpoitn right now is that it r eturns a single invoice
	// we need to return a list of invocies and then the handler on the client side
	// should make sure that it iterates over all the invoices in this list
	// but that's a bit ugly and not all clients may follow the same rules, etc,
	// so whwt's a good move?
	reqInvoices, err := r.Node.InvoiceManager.GetAllRequestedInvoices()
	if err != nil {
		log.Println("Unable ot fetch all requested invoices")
		return err
	}
	pendingInvoices, err := r.Node.InvoiceManager.GetAllPendingInvoices()
	if err != nil {
		log.Println("Unable ot fetch all pending invoices")
		return err
	}
	// now we have both pending and requested invoices. N eed to get the common
	// elements in them so that we can pay those paritcular invoices
	// n^2 but no other option. Also n is low (max 36 elements), so tis okay
	for _, rInvoice := range reqInvoices {
		// check against each pendignInvoice
		for _, pInvoice := range pendingInvoices {
			// need to check each element since this is a struct
			// rInvoice is an InvoiceMsg whereas pInvoice is an InvoiceReplyMsg
			if rInvoice.Id == pInvoice.Id &&
				rInvoice.PeerIdx == pInvoice.PeerIdx {
				log.Println("This is an invoice we must pay", rInvoice)
				// before adding, we must check if this invoice is already in reply.Invoices
				// but how do we do this?
				found := false
				for _, dInvoice := range reply.Invoices {
					if dInvoice.Id == pInvoice.Id && dInvoice.PeerIdx == pInvoice.PeerIdx {
						// invoie alreayd exists, do nothing
						found = true
					}
				}
				if !found {
					reply.Invoices = append(reply.Invoices, pInvoice) // since pinvoice is an InvoiceReplyMsg
				}
			}
		}
	}
	return nil
}

type PayInvoiceConfirmArgs struct {
	// same as lnutil.InvoiceReplyMsg
	Invoice lnutil.InvoiceReplyMsg
}

type PayInvoiceConfirmReply struct {
	success bool
}

// CleanInvoiceAsyncHandler cleans the invocei from the relevant databases since
// the user didn't want to make the payment
func (r *LitRPC) CleanInvoiceAsyncHandler(args *PayInvoiceConfirmArgs,
	reply *PayInvoiceConfirmReply) error {
	err := r.Node.DeleteInvoicePayer(args.Invoice)
	if err != nil {
		log.Println("Couldn't clear up the database after paying the peer, flush manually!")
		return err
	}
	// now we saved this to the list of invoices that we've paid. We should delete the
	// invoice from pending, requested
	return nil
}

func (r *LitRPC) PayInvoiceConfirm(args *PayInvoiceConfirmArgs, reply *PayInvoiceConfirmReply) error {
	// got confirmation from the user to pay the invoice, so pay
	// where do we get destAdr from?
	// we need destAdr i ncase we aren't connected to the peer, get it from lndb somehow
	// and then pass it on to PayInvoice where it should check for a connection
	// and if one doesn't exist, it s hould create a connection.

	// so I know the peerIdx, how can I get the remote node's address from here?
	err := r.Node.PayInvoice(args.Invoice)
	if err != nil {
		return err
	}
	// if everything goes well until here, means we paid the invoice. Add the invoice
	// to the paidInvoice bucket and delete it from the generated address bucket
	// to make the invoiceId free for other invoices to take up
	err = r.Node.InvoiceManager.SavePaidInvoice(&args.Invoice)
	// maybe need to store timestamp as well
	if err != nil {
		log.Println("Paid invoice, couldn't store it in the database")
		return err
	}
	// now we saved this to the list of invoices that we've paid. We should delete the
	// invoice from pending, requested
	err = r.Node.DeleteInvoicePayer(args.Invoice)
	if err != nil {
		log.Println("Couldn't clear up the database after paying the peer, flush manually!")
		return err
	}
	return nil
}

func (r *LitRPC) PayInvoice(args *PayInvoiceArgs, reply *PayInvoiceReply) error {
	var err error
	// send a message out to the peer asking for details
	// parse the recieved message
	destAdr, invoiceId, err := r.Node.SplitInvoiceId(args.Invoice)
	if err != nil {
		return err
	}
	log.Printf("Parsed invoice with destination address: %s and invoice id: %s", destAdr, invoiceId)
	// we must look at the tracker here, but this works for now..
	invoiceRequester := destAdr + "@:2448" // for testing

	rpx, err := r.Node.InvoiceDial(invoiceRequester)
	if err != nil {
		return err
	}
	// now I have the remote Peer
	// send it a byte message
	msgString := "I" + invoiceId
	testStr := []byte(msgString) // this should be the actual invoice preceeded by I
	_, err = rpx.Con.Write(testStr)
	if err != nil {
		log.Println("Error while writing to remote peer")
		return err
	}
	// store this invoice in our list of sent invoices
	// store it in the sentinvocies database so that we can retrieve it later
	var sentInvoice lnutil.InvoiceMsg
	sentInvoice.Id = invoiceId
	sentInvoice.PeerIdx = rpx.Idx

	err = r.Node.InvoiceManager.SaveRequestedInvoice(&sentInvoice)
	if err != nil {
		log.Println("Error while saving to requested invocies. returning")
		return err
	}

	// BUT I may have changed my mind on whether to pay this invoice (eg the messages
	// took more than 10 minutes and I paid cofee by cash instead.)
	// An additional concern here is that this is run by the node and not by the clietnt, so
	// the ndoe needs to ask the alient whether he wants to pay x amount
	// how do we do this?

	// simple way is we return here and then fire up an async handler on lit-af
	// which would then alert us if we have stuff to pay
	return nil
}

type LsInvoiceReplyMsg struct {
	Invoices []lnutil.InvoiceReplyMsg
}

type LsInvoiceMsg struct {
	Invoices []lnutil.InvoiceMsg
}

type LsPaidInvoiceStorageMsg struct {
	Invoices []invoice.PaidInvoiceStorage
}

/*
func (mgr *InvoiceManager) GetAllRepliedInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.displayAllKeyValsDup(BKTRepliedInvoices)
}
func (mgr *InvoiceManager) GetAllRequestedInvoices() ([]lnutil.InvoiceMsg, error) {
	return mgr.displayAllKeyValsDup(BKTRequestedInvoices)
}

// need invoicereplymsges to work
func (mgr *InvoiceManager) GetAllGeneratedInvoices() ([]lnutil.InvoiceReplyMsg, error) {
	return mgr.displayAllKeyValsDup(BKTGeneratedInvoices)
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
*/
func (r *LitRPC) ListAllGeneratedInvoices(args *NoArgs, reply *LsInvoiceReplyMsg) error {
	temp, err := r.Node.InvoiceManager.GetAllGeneratedInvoices()
	if err != nil {
		return err
	}
	reply.Invoices = temp
	return nil
}

func (r *LitRPC) ListAllPendingInvoices(args *NoArgs, reply *LsInvoiceReplyMsg) error {
	temp, err := r.Node.InvoiceManager.GetAllPendingInvoices()
	if err != nil {
		return err
	}
	reply.Invoices = temp
	return nil
}

func (r *LitRPC) ListAllPaidInvoices(args *NoArgs, reply *LsPaidInvoiceStorageMsg) error {
	temp, err := r.Node.InvoiceManager.GetAllPaidInvoices()
	if err != nil {
		return err
	}
	reply.Invoices = temp
	return nil
}

func (r *LitRPC) ListAllGotPaidInvoices(args *NoArgs, reply *LsPaidInvoiceStorageMsg) error {
	temp, err := r.Node.InvoiceManager.GetAllGotPaidInvoices()
	if err != nil {
		return err
	}
	reply.Invoices = temp
	return nil
}

func (r *LitRPC) ListAllRepliedInvoices(args *NoArgs, reply *LsInvoiceMsg) error {
	temp, err := r.Node.InvoiceManager.GetAllRepliedInvoices()
	if err != nil {
		return err
	}
	reply.Invoices = temp
	return nil
}
func (r *LitRPC) ListAllRequestedInvoices(args *NoArgs, reply *LsInvoiceMsg) error {
	temp, err := r.Node.InvoiceManager.GetAllRequestedInvoices()
	if err != nil {
		return err
	}
	reply.Invoices = temp
	return nil
}
