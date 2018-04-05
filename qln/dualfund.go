package qln

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/adiabat/btcd/wire"
	"github.com/adiabat/btcutil/txsort"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

const (
	DUALFUND_DECLINE_REASON_USER                 = 0x01 // User manually declined
	DUALFUND_DECLINE_REASON_INSUFFICIENT_BALANCE = 0x02 // Not enough balance to accept the request
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
	idx := <-nd.InProgDual.done
	return idx, nil
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

func (nd *LitNode) DualFundAccept() {
	wal, ok := nd.SubWallet[nd.InProgDual.CoinType]
	if !ok {
		fmt.Printf("DualFundingReqHandler err no wallet for type %d", nd.InProgDual.CoinType)
		return
	}

	changeAddr, err := wal.NewAdr()
	if err != nil {
		fmt.Printf("Error creating change address: %s", err.Error())
		return
	}

	// Find UTXOs to use
	utxos, _, err := wal.PickUtxos(nd.InProgDual.OurAmount, 500, wal.Fee(), true) //TODO Fee calculation
	if err != nil {
		fmt.Printf("Error fetching UTXOs to use for funding: %s", err.Error())
		return
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
		fmt.Printf("Error fetching UTXOs to use for funding: %s", err.Error())
		return
	}

	nd.InProgDual.mtx.Lock()
	nd.InProgDual.OurChangeAddress = changeAddr
	nd.InProgDual.OurInputs = ourInputs
	nd.InProgDual.OurPub = myChanPub
	nd.InProgDual.OurRefundPub = myRefundPub
	nd.InProgDual.OurHAKDBase = myHAKDbase
	nd.InProgDual.mtx.Unlock()

	outMsg := lnutil.NewDualFundingAcceptMsg(nd.InProgDual.PeerIdx, nd.InProgDual.CoinType, myChanPub, myRefundPub, myHAKDbase, changeAddr, ourInputs)

	nd.OmniOut <- outMsg

	return
}

// RECIPIENT
// DualFundingReqHandler gets a request with the funding data of the remote peer, along with the
// amount of funding requested to return data for.
func (nd *LitNode) DualFundingReqHandler(msg lnutil.DualFundingReqMsg) {

	cIdx, err := nd.NextChannelIdx()
	if err != nil {
		fmt.Printf("DualFundingReqHandler err %s", err.Error())
		return
	}

	wal, ok := nd.SubWallet[msg.CoinType]
	if !ok {
		fmt.Printf("DualFundingReqHandler err no wallet for type %d", msg.CoinType)
		return
	}

	nd.InProgDual.mtx.Lock()
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
		fmt.Printf("DualFundingReqHandler err %s", err.Error())
		return
	}

	nowHeight := wal.CurrentHeight()
	spendable := allPorTxos.SumWitness(nowHeight)
	if msg.TheirAmount > spendable-50000 {
		fmt.Printf("Auto declining dual fund request. Requested %d, but we only have %d for funding\n", msg.TheirAmount, spendable-50000)
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
	nd.InProgDual.mtx.Unlock()

	// Since we haven't gotten more than this - this is where we end. Build the funding TX and print it out
	tx, _ := nd.BuildDualFundingTransaction()
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	tx.Serialize(w)
	w.Flush()
	fmt.Printf("Built the funding TX:\n%x\n", b.Bytes())

	result := new(DualFundingResult)

	result.Accepted = true

	nd.InProgDual.mtx.Lock()
	nd.InProgDual.done <- result
	nd.InProgDual.Clear()
	nd.InProgDual.mtx.Unlock()

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
