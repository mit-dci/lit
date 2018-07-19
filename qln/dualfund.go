package qln

import (
	"fmt"
	"log"

	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/txsort"
	"github.com/mit-dci/lit/elkrem"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/wire"
)

const (
	DUALFUND_DECLINE_REASON_USER                    = 0x01 // User manually declined
	DUALFUND_DECLINE_REASON_INSUFFICIENT_BALANCE    = 0x02 // Not enough balance to accept the request
	DUALFUND_DECLINE_REASON_UNSUPPORTED_COIN        = 0x03 // We don't have that coin type, so we declined automatically
	DUALFUND_DECLINE_REASON_ALREADY_PENDING_REQUEST = 0x04 // Current design only supports single dualfund request at a time. So decline when we have one in progress.

)

/* Dual funding process.

Allows a peer to initiate funding with its counterparty, requesting that party
to also commit funds into the channel.

Parameters:

Peer		:	The peer index to dual fund with
MyAmount	:	The amount i am funding myself
TheirAmount	:	The amount we request the peer to fund

Initial send and data seems unnecessary here, since both parties are funding,
neither is sending the other side money (the initial state will be equal
to the amounts funded)

The process that's followed:

A -> B      Request funding UTXOs and change address for TheirAmount

B -> A      Respond with UTXOs and change address for TheirAmount
(or) B -> A Respond with a decline

A -> B      Respond with UTXOs and change address for MyAmount

A and B     Compose the funding TX

Then (regular funding messages):
A -> B      Requests channel point and refund pubkey, sending along its own

            A channel point (33) (channel pubkey for now)
            A refund (33)

B -> A      Replies with channel point and refund pubkey

            B channel point (32) (channel pubkey for now)
            B refund (33)

A -> B      Sends channel Description:
            ---
            outpoint (36)
            capacity (8)
            initial push (8)
            B's HAKD pub #1 (33)
            signature (~70)
            ---

B -> A      Acknowledges Channel:
            ---
            A's HAKD pub #1 (33)
            signature (~70)
            ---

Then (dual funding specific, exchange of signatures for the funding TX):

A -> B      Requests signatures for the funding TX, sending along its own

B -> A      Sends signatures for the funding TX

A           Publishes funding TX

=== time passes, fund tx gets in a block ===

A -> B      SigProof
            SPV proof of the outpoint (block height, tree depth, tx index, hashes)
            signature (~70)

*/

// DualFundChannel requests a peer to do dual funding. The remote peer can decline.

func (nd *LitNode) DualFundChannel(
	peerIdx, cointype uint32, ourAmount int64, theirAmount int64) (*DualFundingResult, error) {

	nullFundingResult := new(DualFundingResult)
	nullFundingResult.Error = true

	wal, ok := nd.SubWallet[cointype]
	if !ok {
		return nullFundingResult, fmt.Errorf("No wallet of type %d connected", cointype)
	}

	changeAddr, err := wal.NewAdr()
	if err != nil {
		return nullFundingResult, err
	}

	nd.InProgDual.mtx.Lock()
	//	defer nd.InProg.mtx.Lock()
	if nd.InProgDual.PeerIdx != 0 {
		nd.InProgDual.mtx.Unlock()
		return nullFundingResult, fmt.Errorf("dual fund with peer %d not done yet", nd.InProgDual.PeerIdx)
	}

	if theirAmount <= 0 || ourAmount <= 0 {
		nd.InProgDual.mtx.Unlock()
		return nullFundingResult, fmt.Errorf("dual funding requires both parties to commit funds")
	}

	if theirAmount+ourAmount < 1000000 { // limit for now
		nd.InProgDual.mtx.Unlock()
		return nullFundingResult, fmt.Errorf("Min channel capacity 1M sat")
	}

	// TODO - would be convenient if it auto connected to the peer huh
	if !nd.ConnectedToPeer(peerIdx) {
		nd.InProgDual.mtx.Unlock()
		return nullFundingResult, fmt.Errorf("Not connected to peer %d. Do that yourself.", peerIdx)
	}

	cIdx, err := nd.NextChannelIdx()
	if err != nil {
		nd.InProgDual.mtx.Unlock()
		return nullFundingResult, err
	}

	// Find UTXOs to use
	utxos, _, err := wal.PickUtxos(ourAmount, 500, wal.Fee(), true) //TODO Fee calculation
	if err != nil {
		return nil, err
	}

	ourInputs := make([]lnutil.DualFundingInput, len(utxos))
	for i := 0; i < len(utxos); i++ {
		ourInputs[i] = lnutil.DualFundingInput{utxos[i].Op, utxos[i].Value}
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = cointype | 1<<31
	kg.Step[2] = UseChannelFund
	kg.Step[3] = peerIdx | 1<<31
	kg.Step[4] = cIdx | 1<<31

	myChanPub, _ := nd.GetUsePub(kg, UseChannelFund)
	myRefundPub, _ := nd.GetUsePub(kg, UseChannelRefund)
	myHAKDbase, err := nd.GetUsePub(kg, UseChannelHAKDBase)
	if err != nil {
		return nil, err
	}

	nd.InProgDual.ChanIdx = cIdx
	nd.InProgDual.PeerIdx = peerIdx
	nd.InProgDual.CoinType = cointype
	nd.InProgDual.OurAmount = ourAmount
	nd.InProgDual.TheirAmount = theirAmount
	nd.InProgDual.OurChangeAddress = changeAddr
	nd.InProgDual.InitiatedByUs = true
	nd.InProgDual.OurPub = myChanPub
	nd.InProgDual.OurRefundPub = myRefundPub
	nd.InProgDual.OurHAKDBase = myHAKDbase
	nd.InProgDual.OurInputs = ourInputs
	nd.InProgDual.mtx.Unlock() // switch to defer

	outMsg := lnutil.NewDualFundingReqMsg(peerIdx, cointype, ourAmount, theirAmount, myChanPub, myRefundPub, myHAKDbase, changeAddr, ourInputs)

	nd.OmniOut <- outMsg

	// wait until it's done!
	result := <-nd.InProgDual.done

	return result, nil
}

// Declines the in progress (received) dual funding request
func (nd *LitNode) DualFundDecline(reason uint8) {
	outMsg := lnutil.NewDualFundingDeclMsg(nd.InProgDual.PeerIdx, reason)
	nd.OmniOut <- outMsg
	nd.InProgDual.mtx.Lock()
	nd.InProgDual.Clear()
	nd.InProgDual.mtx.Unlock()
	return
}

func (nd *LitNode) DualFundAccept() *DualFundingResult {
	wal, ok := nd.SubWallet[nd.InProgDual.CoinType]
	if !ok {
		log.Printf("DualFundingReqHandler err no wallet for type %d", nd.InProgDual.CoinType)
		return nil
	}

	changeAddr, err := wal.NewAdr()
	if err != nil {
		log.Printf("Error creating change address: %s", err.Error())
		return nil
	}

	// Find UTXOs to use
	utxos, _, err := wal.PickUtxos(nd.InProgDual.OurAmount, 500, wal.Fee(), true) //TODO Fee calculation
	if err != nil {
		log.Printf("Error fetching UTXOs to use for funding: %s", err.Error())
		return nil
	}

	ourInputs := make([]lnutil.DualFundingInput, len(utxos))
	for i := 0; i < len(utxos); i++ {
		ourInputs[i] = lnutil.DualFundingInput{utxos[i].Op, utxos[i].Value}
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = nd.InProgDual.CoinType | 1<<31
	kg.Step[2] = UseChannelFund
	kg.Step[3] = nd.InProgDual.PeerIdx | 1<<31
	kg.Step[4] = nd.InProgDual.ChanIdx | 1<<31

	myChanPub, _ := nd.GetUsePub(kg, UseChannelFund)
	myRefundPub, _ := nd.GetUsePub(kg, UseChannelRefund)
	myHAKDbase, err := nd.GetUsePub(kg, UseChannelHAKDBase)
	if err != nil {
		log.Printf("Error fetching UTXOs to use for funding: %s", err.Error())
		return nil
	}

	var keyGen portxo.KeyGen
	keyGen.Depth = 5
	keyGen.Step[0] = 44 | 1<<31
	keyGen.Step[1] = nd.InProgDual.CoinType | 1<<31
	keyGen.Step[2] = UseHTLCBase
	keyGen.Step[3] = 0 | 1<<31
	keyGen.Step[4] = nd.InProgDual.ChanIdx | 1<<31

	MyNextHTLCBase, err := nd.GetUsePub(keyGen, UseHTLCBase)
	if err != nil {
		log.Printf("error generating NextHTLCBase %v", err)
		return nil
	}

	keyGen.Step[3] = 1 | 1<<31
	MyN2HTLCBase, err := nd.GetUsePub(keyGen, UseHTLCBase)
	if err != nil {
		log.Printf("error generating N2HTLCBase %v", err)
		return nil
	}

	nd.InProgDual.mtx.Lock()
	nd.InProgDual.OurChangeAddress = changeAddr
	nd.InProgDual.OurInputs = ourInputs
	nd.InProgDual.OurPub = myChanPub
	nd.InProgDual.OurRefundPub = myRefundPub
	nd.InProgDual.OurHAKDBase = myHAKDbase
	nd.InProgDual.OurNextHTLCBase = MyNextHTLCBase
	nd.InProgDual.OurN2HTLCBase = MyN2HTLCBase
	nd.InProgDual.mtx.Unlock()

	outMsg := lnutil.NewDualFundingAcceptMsg(nd.InProgDual.PeerIdx, nd.InProgDual.CoinType, myChanPub, myRefundPub, myHAKDbase, changeAddr, ourInputs, MyNextHTLCBase, MyN2HTLCBase)

	nd.OmniOut <- outMsg

	// wait until it's done!
	result := <-nd.InProgDual.done

	return result
}

// RECIPIENT
// DualFundingReqHandler gets a request with the funding data of the remote peer, along with the
// amount of funding requested to return data for.
func (nd *LitNode) DualFundingReqHandler(msg lnutil.DualFundingReqMsg) {

	cIdx, err := nd.NextChannelIdx()
	if err != nil {
		log.Printf("DualFundingReqHandler err %s", err.Error())
		return
	}

	wal, ok := nd.SubWallet[msg.CoinType]
	if !ok {
		log.Printf("DualFundingReqHandler err no wallet for type %d", msg.CoinType)
		log.Printf("Auto declining dual fund request. We don't handle coin type %d\n", msg.CoinType)
		nd.DualFundDecline(DUALFUND_DECLINE_REASON_UNSUPPORTED_COIN)
		return
	}

	nd.InProgDual.mtx.Lock()
	if nd.InProgDual.PeerIdx != 0 {
		log.Printf("DualFundingReqHandler already have a pending request. Declining.")
		nd.DualFundDecline(DUALFUND_DECLINE_REASON_ALREADY_PENDING_REQUEST)
		nd.InProgDual.mtx.Unlock()
		return
	}

	nd.InProgDual.ChanIdx = cIdx
	nd.InProgDual.PeerIdx = msg.Peer()
	nd.InProgDual.CoinType = msg.CoinType
	nd.InProgDual.OurAmount = msg.TheirAmount
	nd.InProgDual.TheirInputs = msg.OurInputs
	nd.InProgDual.TheirAmount = msg.OurAmount
	nd.InProgDual.TheirChangeAddress = msg.OurChangeAddressPKH
	nd.InProgDual.TheirPub = msg.OurPub
	nd.InProgDual.TheirRefundPub = msg.OurRefundPub
	nd.InProgDual.TheirHAKDBase = msg.OurHAKDBase
	nd.InProgDual.InitiatedByUs = false
	nd.InProgDual.mtx.Unlock()

	// Check if we have the requested amount, otherwise auto-decline
	var allPorTxos portxo.TxoSliceByAmt
	allPorTxos, err = wal.UtxoDump()
	if err != nil {
		log.Printf("DualFundingReqHandler err %s", err.Error())
		return
	}

	nowHeight := wal.CurrentHeight()
	spendable := allPorTxos.SumWitness(nowHeight)
	if msg.TheirAmount > spendable-50000 {
		log.Printf("Auto declining dual fund request. Requested %d, but we only have %d for funding\n", msg.TheirAmount, spendable-50000)
		nd.DualFundDecline(DUALFUND_DECLINE_REASON_INSUFFICIENT_BALANCE)
		return
	}

	return
}

// RECIPIENT
// DualFundingDeclHandler gets a message where the remote party is declining the
// request to dualfund.
func (nd *LitNode) DualFundingDeclHandler(msg lnutil.DualFundingDeclMsg) {

	result := new(DualFundingResult)

	result.DeclineReason = msg.Reason

	nd.InProgDual.mtx.Lock()
	nd.InProgDual.done <- result
	nd.InProgDual.Clear()
	nd.InProgDual.mtx.Unlock()

	return
}

// DualFundingAcceptHandler gets a message where the remote party is accepting the
// request to dualfund.
func (nd *LitNode) DualFundingAcceptHandler(msg lnutil.DualFundingAcceptMsg) {

	nd.InProgDual.mtx.Lock()
	nd.InProgDual.TheirInputs = msg.OurInputs
	nd.InProgDual.TheirChangeAddress = msg.OurChangeAddressPKH
	nd.InProgDual.TheirPub = msg.OurPub
	nd.InProgDual.TheirRefundPub = msg.OurRefundPub
	nd.InProgDual.TheirHAKDBase = msg.OurHAKDBase

	// make channel (not in db) just for keys / elk
	q := new(Qchan)

	q.Height = -1

	q.Value = nd.InProg.Amt

	q.KeyGen.Depth = 5
	q.KeyGen.Step[0] = 44 | 1<<31
	q.KeyGen.Step[1] = nd.InProgDual.CoinType | 1<<31
	q.KeyGen.Step[2] = UseChannelFund
	q.KeyGen.Step[3] = nd.InProgDual.PeerIdx | 1<<31
	q.KeyGen.Step[4] = nd.InProgDual.ChanIdx | 1<<31

	q.MyPub, _ = nd.GetUsePub(q.KeyGen, UseChannelFund)
	q.MyRefundPub, _ = nd.GetUsePub(q.KeyGen, UseChannelRefund)
	q.MyHAKDBase, _ = nd.GetUsePub(q.KeyGen, UseChannelHAKDBase)

	// chop up incoming message, save points to channel struct
	copy(q.TheirPub[:], nd.InProgDual.TheirPub[:])
	copy(q.TheirRefundPub[:], nd.InProgDual.TheirRefundPub[:])
	copy(q.TheirHAKDBase[:], nd.InProgDual.TheirHAKDBase[:])

	// make sure their pubkeys are real pubkeys
	_, err := btcec.ParsePubKey(q.TheirPub[:], btcec.S256())
	if err != nil {
		nd.InProgDual.mtx.Unlock()
		log.Printf("PubRespHandler TheirPub err %s", err.Error())
		return
	}
	_, err = btcec.ParsePubKey(q.TheirRefundPub[:], btcec.S256())
	if err != nil {
		nd.InProgDual.mtx.Unlock()
		log.Printf("PubRespHandler TheirRefundPub err %s", err.Error())
		return
	}
	_, err = btcec.ParsePubKey(q.TheirHAKDBase[:], btcec.S256())
	if err != nil {
		nd.InProgDual.mtx.Unlock()
		log.Printf("PubRespHandler TheirHAKDBase err %s", err.Error())
		return
	}

	_, err = btcec.ParsePubKey(msg.OurNextHTLCBase[:], btcec.S256())
	if err != nil {
		nd.InProgDual.mtx.Unlock()
		log.Printf("PubRespHandler NextHTLCBase err %s", err.Error())
		return
	}
	_, err = btcec.ParsePubKey(msg.OurN2HTLCBase[:], btcec.S256())
	if err != nil {
		nd.InProgDual.mtx.Unlock()
		log.Printf("PubRespHandler N2HTLCBase err %s", err.Error())
		return
	}

	nd.InProgDual.TheirNextHTLCBase = msg.OurNextHTLCBase
	nd.InProgDual.TheirN2HTLCBase = msg.OurN2HTLCBase

	// derive elkrem sender root from HD keychain
	elkRoot, _ := nd.GetElkremRoot(q.KeyGen)
	q.ElkSnd = elkrem.NewElkremSender(elkRoot)

	// Build the funding transaction
	tx, _ := nd.BuildDualFundingTransaction()

	outPoint := wire.OutPoint{tx.TxHash(), 0}
	nd.InProgDual.OutPoint = &outPoint
	q.Op = *nd.InProgDual.OutPoint

	// create initial state for elkrem points
	q.State = new(StatCom)
	q.State.StateIdx = 0
	q.State.MyAmt = nd.InProgDual.OurAmount
	// get fee from sub wallet.  Later should make fee per channel and update state
	// based on size
	q.State.Fee = nd.SubWallet[q.Coin()].Fee() * 1000
	q.Value = nd.InProgDual.OurAmount + nd.InProgDual.TheirAmount

	q.State.NextHTLCBase = msg.OurNextHTLCBase
	q.State.N2HTLCBase = msg.OurN2HTLCBase
	q.State.MyNextHTLCBase = nd.InProgDual.OurNextHTLCBase
	q.State.MyN2HTLCBase = nd.InProgDual.OurN2HTLCBase

	// save channel to db
	err = nd.SaveQChan(q)
	if err != nil {
		log.Printf("PointRespHandler SaveQchanState err %s", err.Error())
		return
	}

	// when funding a channel, give them the first *3* elkpoints.
	elkPointZero, err := q.ElkPoint(false, 0)
	if err != nil {
		log.Printf("PointRespHandler ElkpointZero err %s", err.Error())
		return
	}
	elkPointOne, err := q.ElkPoint(false, 1)
	if err != nil {
		log.Printf("PointRespHandler ElkpointOne err %s", err.Error())
		return
	}

	elkPointTwo, err := q.N2ElkPointForThem()
	if err != nil {
		log.Printf("PointRespHandler ElkpointTwo err %s", err.Error())
		return
	}

	outMsg := lnutil.NewChanDescMsg(
		nd.InProgDual.PeerIdx, *nd.InProgDual.OutPoint, q.MyPub, q.MyRefundPub, q.MyHAKDBase, nd.InProgDual.OurNextHTLCBase,
		nd.InProgDual.OurN2HTLCBase,
		nd.InProgDual.CoinType, nd.InProgDual.OurAmount+nd.InProgDual.TheirAmount, nd.InProgDual.TheirAmount,
		elkPointZero, elkPointOne, elkPointTwo, q.State.Data)

	nd.InProgDual.mtx.Unlock()
	nd.OmniOut <- outMsg

	return
}

func (nd *LitNode) BuildDualFundingTransaction() (*wire.MsgTx, error) {
	// make the tx
	tx := wire.NewMsgTx()

	w, ok := nd.SubWallet[nd.InProgDual.CoinType]
	if !ok {
		err := fmt.Errorf("BuildDualFundingTransaction err no wallet for type %d", nd.InProgDual.CoinType)
		return tx, err
	}

	// set version 2, for op_csv
	tx.Version = 2
	// set the time, the way core does.
	tx.LockTime = uint32(w.CurrentHeight())

	// add all the txins
	var ourInputTotal int64
	var theirInputTotal int64

	for _, u := range nd.InProgDual.OurInputs {
		tx.AddTxIn(wire.NewTxIn(&u.Outpoint, nil, nil))
		ourInputTotal += u.Value
	}
	for _, u := range nd.InProgDual.TheirInputs {
		tx.AddTxIn(wire.NewTxIn(&u.Outpoint, nil, nil))
		theirInputTotal += u.Value
	}

	var initiatorPub [33]byte
	var counterPartyPub [33]byte
	if nd.InProgDual.InitiatedByUs {
		initiatorPub = nd.InProgDual.OurPub
		counterPartyPub = nd.InProgDual.TheirPub
	} else {
		initiatorPub = nd.InProgDual.TheirPub
		counterPartyPub = nd.InProgDual.OurPub
	}

	// get txo for channel
	txo, err := lnutil.FundTxOut(initiatorPub, counterPartyPub, nd.InProgDual.OurAmount+nd.InProgDual.TheirAmount)
	if err != nil {
		return tx, err
	}
	tx.AddTxOut(txo)

	// add the change outputs
	var initiatorChange int64
	var counterPartyChange int64
	var initiatorChangeAddress [20]byte
	var counterPartyChangeAddress [20]byte

	if nd.InProgDual.InitiatedByUs {
		initiatorChangeAddress = nd.InProgDual.OurChangeAddress
		initiatorChange = ourInputTotal - nd.InProgDual.OurAmount - 500
		counterPartyChangeAddress = nd.InProgDual.TheirChangeAddress
		counterPartyChange = theirInputTotal - nd.InProgDual.TheirAmount - 500

	} else {
		initiatorChangeAddress = nd.InProgDual.TheirChangeAddress
		initiatorChange = theirInputTotal - nd.InProgDual.TheirAmount - 500
		counterPartyChangeAddress = nd.InProgDual.OurChangeAddress
		counterPartyChange = ourInputTotal - nd.InProgDual.OurAmount - 500
	}

	changeScriptInitiator := lnutil.DirectWPKHScriptFromPKH(initiatorChangeAddress)
	tx.AddTxOut(wire.NewTxOut(initiatorChange, changeScriptInitiator))

	changeScriptCounterParty := lnutil.DirectWPKHScriptFromPKH(counterPartyChangeAddress)
	tx.AddTxOut(wire.NewTxOut(counterPartyChange, changeScriptCounterParty))

	txsort.InPlaceSort(tx)

	return tx, nil
}

// RECIPIENT
// QChanDescHandler takes in a description of a channel output.  It then
// saves it to the local db, and returns a channel acknowledgement
func (nd *LitNode) DualFundChanDescHandler(msg lnutil.ChanDescMsg) {

	log.Printf("DualFundChanDescHandler\n")

	wal, ok := nd.SubWallet[msg.CoinType]
	if !ok {
		log.Printf("DualFundChanDescHandler err no wallet for type %d", msg.CoinType)
		return
	}

	// deserialize desc
	op := msg.Outpoint
	opArr := lnutil.OutPointToBytes(op)
	amt := msg.Capacity

	cIdx, err := nd.NextChannelIdx()
	if err != nil {
		log.Printf("DualFundChanDescHandler err %s", err.Error())
		return
	}

	qc := new(Qchan)

	qc.Height = -1
	qc.KeyGen.Depth = 5
	qc.KeyGen.Step[0] = 44 | 1<<31
	qc.KeyGen.Step[1] = msg.CoinType | 1<<31
	qc.KeyGen.Step[2] = UseChannelFund
	qc.KeyGen.Step[3] = msg.Peer() | 1<<31
	qc.KeyGen.Step[4] = cIdx | 1<<31
	qc.Value = amt
	qc.Mode = portxo.TxoP2WSHComp
	qc.Op = op

	qc.TheirPub = msg.PubKey
	qc.TheirRefundPub = msg.RefundPub
	qc.TheirHAKDBase = msg.HAKDbase
	qc.MyPub, _ = nd.GetUsePub(qc.KeyGen, UseChannelFund)
	qc.MyRefundPub, _ = nd.GetUsePub(qc.KeyGen, UseChannelRefund)
	qc.MyHAKDBase, _ = nd.GetUsePub(qc.KeyGen, UseChannelHAKDBase)

	_, err = btcec.ParsePubKey(msg.NextHTLCBase[:], btcec.S256())
	if err != nil {
		fmt.Errorf("QChanDescHandler NextHTLCBase err %s", err.Error())
		return
	}
	_, err = btcec.ParsePubKey(msg.N2HTLCBase[:], btcec.S256())
	if err != nil {
		fmt.Errorf("QChanDescHandler N2HTLCBase err %s", err.Error())
		return
	}

	// it should go into the next bucket and get the right key index.
	// but we can't actually check that.
	//	qc, err := nd.SaveFundTx(
	//		op, amt, peerArr, theirPub, theirRefundPub, theirHAKDbase)
	//	if err != nil {
	//		log.Printf("QChanDescHandler SaveFundTx err %s", err.Error())
	//		return
	//	}
	log.Printf("got multisig output %s amt %d\n", op.String(), amt)

	// create initial state
	qc.State = new(StatCom)
	// similar to SIGREV in pushpull

	// TODO assumes both parties use same fee
	qc.State.Fee = wal.Fee() * 1000
	qc.State.MyAmt = msg.InitPayment

	qc.State.Data = msg.Data

	qc.State.StateIdx = 0
	// use new ElkPoint for signing
	qc.State.ElkPoint = msg.ElkZero
	qc.State.NextElkPoint = msg.ElkOne
	qc.State.N2ElkPoint = msg.ElkTwo

	qc.State.MyNextHTLCBase = nd.InProgDual.OurNextHTLCBase
	qc.State.MyN2HTLCBase = nd.InProgDual.OurN2HTLCBase

	qc.State.NextHTLCBase = msg.NextHTLCBase
	qc.State.N2HTLCBase = msg.N2HTLCBase

	// save new channel to db
	err = nd.SaveQChan(qc)
	if err != nil {
		log.Printf("DualFundChanDescHandler err %s", err.Error())
		return
	}

	// load ... the thing I just saved.  why?
	qc, err = nd.GetQchan(opArr)
	if err != nil {
		log.Printf("DualFundChanDescHandler GetQchan err %s", err.Error())
		return
	}

	// when funding a channel, give them the first *2* elkpoints.
	theirElkPointZero, err := qc.ElkPoint(false, 0)
	if err != nil {
		log.Printf("DualFundChanDescHandler err %s", err.Error())
		return
	}
	theirElkPointOne, err := qc.ElkPoint(false, 1)
	if err != nil {
		log.Printf("DualFundChanDescHandler err %s", err.Error())
		return
	}

	theirElkPointTwo, err := qc.N2ElkPointForThem()
	if err != nil {
		log.Printf("DualFundChanDescHandler err %s", err.Error())
		return
	}

	sig, _, err := nd.SignState(qc)
	if err != nil {
		log.Printf("DualFundChanDescHandler SignState err %s", err.Error())
		return
	}

	log.Printf("Acking channel...\n")

	fundingTx, err := nd.BuildDualFundingTransaction()
	if err != nil {
		log.Printf("DualFundChanDescHandler BuildDualFundingTransaction err %s", err.Error())
		return
	}

	wal.SignMyInputs(fundingTx)

	outMsg := lnutil.NewDualFundingChanAckMsg(
		msg.Peer(), op,
		theirElkPointZero, theirElkPointOne, theirElkPointTwo,
		sig, fundingTx)

	nd.OmniOut <- outMsg

	return
}

// FUNDER
// QChanAckHandler takes in an acknowledgement multisig description.
// when a multisig outpoint is ackd, that causes the funder to sign and broadcast.
func (nd *LitNode) DualFundChanAckHandler(msg lnutil.DualFundingChanAckMsg, peer *RemotePeer) {
	opArr := lnutil.OutPointToBytes(msg.Outpoint)
	sig := msg.Signature

	// load channel to save their refund address
	qc, err := nd.GetQchan(opArr)
	if err != nil {
		log.Printf("DualFundChanAckHandler GetQchan err %s", err.Error())
		return
	}

	//	err = qc.IngestElkrem(revElk)
	//	if err != nil { // this can't happen because it's the first elk... remove?
	//		log.Printf("QChanAckHandler IngestElkrem err %s", err.Error())
	//		return
	//	}
	qc.State.ElkPoint = msg.ElkZero
	qc.State.NextElkPoint = msg.ElkOne
	qc.State.N2ElkPoint = msg.ElkTwo

	err = qc.VerifySigs(sig, nil)
	if err != nil {
		log.Printf("DualFundChanAckHandler VerifySig err %s", err.Error())
		return
	}

	// verify worked; Save state 1 to DB
	err = nd.SaveQchanState(qc)
	if err != nil {
		log.Printf("DualFundChanAckHandler SaveQchanState err %s", err.Error())
		return
	}

	// Make sure everything works & is saved, then clear InProg.

	// sign their com tx to send
	sig, _, err = nd.SignState(qc)
	if err != nil {
		log.Printf("DualFundChanAckHandler SignState err %s", err.Error())
		return
	}

	// OK to fund.
	tx, err := nd.BuildDualFundingTransaction()
	if err != nil {
		log.Printf("DualFundChanAckHandler BuildDualFundingTransaction err %s", err.Error())
		return
	}
	err = nd.SubWallet[qc.Coin()].SignMyInputs(tx)
	if err != nil {
		log.Printf("DualFundChanAckHandler SignMyInputs err %s", err.Error())
		return
	}

	// Add signatures from peer
	for i := range msg.SignedFundingTx.TxIn {
		if (msg.SignedFundingTx.TxIn[i].Witness != nil || msg.SignedFundingTx.TxIn[i].SignatureScript != nil) && tx.TxIn[i].Witness == nil && tx.TxIn[i].SignatureScript == nil {
			tx.TxIn[i].Witness = msg.SignedFundingTx.TxIn[i].Witness
			tx.TxIn[i].SignatureScript = msg.SignedFundingTx.TxIn[i].SignatureScript
		}
	}

	nd.SubWallet[qc.Coin()].DirectSendTx(tx)

	err = nd.SubWallet[qc.Coin()].WatchThis(qc.Op)
	if err != nil {
		log.Printf("DualFundChanAckHandler WatchThis err %s", err.Error())
		return
	}

	log.Printf("Registering refund address in the wallet\n")
	// tell base wallet about watcher refund address in case that happens
	// TODO this is weird & ugly... maybe have an export keypath func?
	nullTxo := new(portxo.PorTxo)
	nullTxo.Value = 0 // redundant, but explicitly show that this is just for adr
	nullTxo.KeyGen = qc.KeyGen
	nullTxo.KeyGen.Step[2] = UseChannelWatchRefund
	nd.SubWallet[qc.Coin()].ExportUtxo(nullTxo)

	// channel creation is ~complete, clear InProg.
	// We may be asked to re-send the sig-proof
	result := new(DualFundingResult)

	result.Accepted = true
	result.ChannelId = qc.KeyGen.Step[4] & 0x7fffffff

	log.Printf("Built result with channel ID %d, committing...\n", result.ChannelId)

	nd.InProgDual.mtx.Lock()
	nd.InProgDual.done <- result
	nd.InProgDual.Clear()
	nd.InProgDual.mtx.Unlock()

	peer.QCs[qc.Idx()] = qc
	peer.OpMap[opArr] = qc.Idx()

	// sig proof should be sent later once there are confirmations.
	// it'll have an spv proof of the fund tx.
	// but for now just send the sig.
	log.Printf("Sending sigproof\n")

	outMsg := lnutil.NewSigProofMsg(msg.Peer(), msg.Outpoint, sig)

	nd.OmniOut <- outMsg

	return
}

// RECIPIENT
// SigProofHandler saves the signature the recipient stores.
// In some cases you don't need this message.
func (nd *LitNode) DualFundSigProofHandler(msg lnutil.SigProofMsg, peer *RemotePeer) {

	op := msg.Outpoint
	opArr := lnutil.OutPointToBytes(op)

	qc, err := nd.GetQchan(opArr)
	if err != nil {
		log.Printf("DualFundSigProofHandler err %s", err.Error())
		return
	}

	wal, ok := nd.SubWallet[qc.Coin()]
	if !ok {
		log.Printf("Not connected to coin type %d\n", qc.Coin())
		return
	}

	err = qc.VerifySigs(msg.Signature, nil)
	if err != nil {
		log.Printf("DualFundSigProofHandler err %s", err.Error())
		return
	}

	// sig OK, save
	err = nd.SaveQchanState(qc)
	if err != nil {
		log.Printf("DualFundSigProofHandler err %s", err.Error())
		return
	}

	err = wal.WatchThis(op)

	if err != nil {
		log.Printf("DualFundSigProofHandler err %s", err.Error())
		return
	}

	// tell base wallet about watcher refund address in case that happens
	nullTxo := new(portxo.PorTxo)
	nullTxo.Value = 0 // redundant, but explicitly show that this is just for adr
	nullTxo.KeyGen = qc.KeyGen
	nullTxo.KeyGen.Step[2] = UseChannelWatchRefund
	wal.ExportUtxo(nullTxo)

	peer.QCs[qc.Idx()] = qc
	peer.OpMap[opArr] = qc.Idx()

	// sig OK; in terms of UI here's where you can say "payment received"
	// "channel online" etc

	result := new(DualFundingResult)

	result.Accepted = true
	result.ChannelId = qc.KeyGen.Step[4] & 0x7fffffff
	nd.InProgDual.mtx.Lock()
	nd.InProgDual.done <- result
	nd.InProgDual.Clear()
	nd.InProgDual.mtx.Unlock()

	return
}
