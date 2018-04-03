package qln

import (
	"fmt"

	"github.com/mit-dci/lit/lnutil"
)

const (
	DECLINE_REASON_USER = 0x01 // User manually declined
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

	nd.InProgDual.ChanIdx = cIdx
	nd.InProgDual.PeerIdx = peerIdx
	nd.InProgDual.CoinType = cointype
	nd.InProgDual.OurAmount = ourAmount
	nd.InProgDual.TheirAmount = theirAmount
	nd.InProgDual.OurChangeAddress = changeAddr
	nd.InProgDual.InitiatedByUs = true

	nd.InProgDual.mtx.Unlock() // switch to defer

	// Find UTXOs to use
	utxos, _, err := wal.PickUtxos(ourAmount, 500, wal.Fee(), false)
	if err != nil {
		return nil, err
	}

	outMsg := lnutil.NewDualFundingReqMsg(peerIdx, cointype, ourAmount, theirAmount, changeAddr, utxos)

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

	_, ok := nd.SubWallet[msg.CoinType]
	if !ok {
		fmt.Printf("DualFundingReqHandler err no wallet for type %d", msg.CoinType)
		return
	}

	nd.InProgDual.mtx.Lock()
	nd.InProgDual.ChanIdx = cIdx
	nd.InProgDual.PeerIdx = msg.Peer()
	nd.InProgDual.CoinType = msg.CoinType
	nd.InProgDual.OurAmount = msg.TheirAmount
	nd.InProgDual.TheirAmount = msg.OurAmount
	nd.InProgDual.TheirChangeAddress = msg.OurChangeAddressPKH
	nd.InProgDual.InitiatedByUs = false
	nd.InProgDual.mtx.Unlock()
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
