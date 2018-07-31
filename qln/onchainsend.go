package qln

import (
	"bytes"
	"fmt"

	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/wire"
)

func (nd *LitNode) OnChainSend(lnAdr string, coinType uint32, amt int64, data [32]byte) error {

	// Check if we have a wallet of the requested coin type
	wal, ok := nd.SubWallet[coinType]
	if !ok {
		return fmt.Errorf("No wallet for coinType [%d]", coinType)
	}

	// OK, so pushing over a channel was not possible. Let's check if
	// we still have enough funds in our "on-chain wallet"
	_, _, err := wal.PickUtxos(amt, 500, wal.Fee(), false)
	if err != nil {
		return err
	}

	inFlight := new(InFlightSend)
	inFlight.PeerIdx = nd.GetPeerIdxFromAdr(lnAdr)

	if inFlight.PeerIdx == 0 {
		// Connect first!
		err := nd.DialPeer(lnAdr)
		if err != nil {
			return err
		}
		inFlight.PeerIdx = nd.GetPeerIdxFromAdr(lnAdr)
	}

	inFlight.Amt = amt
	inFlight.CoinType = coinType
	inFlight.Data = data

	nd.RemoteMtx.Lock()
	nd.InFlightSends = append(nd.InFlightSends, inFlight)
	nd.RemoteMtx.Unlock()

	msg := lnutil.NewOnChainPaymentRequestMsg(inFlight.PeerIdx, coinType, data)
	nd.OmniOut <- msg
	return nil
}

func (nd *LitNode) OnChainPaymentRequestMsgHandler(msg lnutil.OnChainPaymentRequestMsg, peer *RemotePeer) error {
	wal, ok := nd.SubWallet[msg.CoinType]
	if !ok {
		return fmt.Errorf("Someone requested a payment address for cointype [%d] but we have no wallet for that.", msg.CoinType)
	}

	pkh, err := wal.NewAdr()
	if err != nil {
		return err
	}

	outMsg := lnutil.NewOnChainPaymentReplyMsg(msg.PeerIdx, msg.Data, pkh)
	nd.OmniOut <- outMsg
	return nil
}

func (nd *LitNode) OnChainPaymentReplyMsgHandler(msg lnutil.OnChainPaymentReplyMsg, peer *RemotePeer) error {
	// Received address. Find inflight send
	var inFlight *InFlightSend

	for _, ifs := range nd.InFlightSends {
		if ifs.PeerIdx == msg.PeerIdx && bytes.Equal(ifs.Data[:], msg.Data[:]) {
			inFlight = ifs
		}
	}

	if inFlight == nil {
		return fmt.Errorf("Someone sent us a payment address, but we don't have a matching inflight send waiting for them.")
	}

	wal, ok := nd.SubWallet[inFlight.CoinType]
	if !ok {
		// This should never happen
		return fmt.Errorf("Somehow we have an inflight send for a cointype %d we don't support", inFlight.CoinType)
	}

	txOuts := make([]*wire.TxOut, 1)
	txOuts[0] = wire.NewTxOut(inFlight.Amt, lnutil.DirectWPKHScriptFromPKH(msg.PKH))

	// we don't care if it's witness or not
	ops, err := wal.MaybeSend(txOuts, false)
	if err != nil {
		return err
	}

	err = wal.ReallySend(&ops[0].Hash)
	if err != nil {
		return err
	}

	return nil
}
