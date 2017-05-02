package qln

import (
	"fmt"

	"github.com/adiabat/btcd/btcec"
	"github.com/adiabat/btcd/wire"
	"github.com/mit-dci/lit/elkrem"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

/*
right now fund makes a channel without actually building commit
transactions before signing and broadcasting the fund transaction.
Once state update push/pull messages work that will be added on to
this process

Note that the first elkrem exchange revokes state 0, which was never actually
commited to  (there are no HAKDpubs for state 0; those start at state 1.)
So it's kindof pointless, but you still have to send the right one, because
elkrem 2 is the parent of elkrems 0 and 1, so that checks 0.

*/

/*
New funding process.
No fancy curve stuff.  Just ask the other node for a point on a curve, which
will be their channel pubkey.
There are 2 ways to then change to a channel-proof-y method later.
One way is to construct the channel pubkey FROM that point, by having
ID:A
Random point:B
Channel point:C
and set C = hash(A, B)*(A + B)
or you could do it 4-way so that
C = hash(A1, B1, A2, B2)*(A + B)
to commit to both sides' creation process.  This is a little more complex,
but the proof of ownership for the channel just consists of the point B so
it's compact.

Another way to do it, after the fact with arbitrary points.
ID:A, Channel pub:C
B = A + C
sign B with b.  That signature is proof that someone / something knew
a and c at the same time.

Either of these can be added later without changing much.  The messages
don't have to change at all, and in the first case you'd change the channel
pubkey calculation.  In the second it's independant of the fund process.

For now though:
funding --
A -> B point request

A channel point (33) (channel pubkey for now)
A refund (33)

B -> A point response
B replies with channel point and refund pubkey

B channel point (32) (channel pubkey for now)
B refund (33)

A -> B Channel Description:
---
outpoint (36)
capacity (8)
initial push (8)
B's HAKD pub #1 (33)
signature (~70)
---

add next:
timeout (2)
fee? fee can
(fee / timeout...?  hardcoded for now)

B -> A  Channel Acknowledge:
A's HAKD pub #1 (33)
signature (~70)

=== time passes, fund tx gets in a block ===

A -> B SigProof
SPV proof of the outpoint (block height, tree depth, tx index, hashes)
signature (~70)


B knows the channel is open and he got paid when he receives the sigproof.
A's got B's signature already.  So "payment happened" is sortof the same as
bitcoin now; wait for confirmations.

Alternatively A can open a channel with no initial funding going to B, then
update the state once the channel is open.  If for whatever reason you want
an exact timing for the payment.

*/

// FundChannel opens a channel with a peer.  Doesn't return until the channel
// has been created.  Maybe timeout if it takes too long?
func (nd *LitNode) FundChannel(peerIdx uint32, ccap, initSend int64) (uint32, error) {

	nd.InProg.mtx.Lock()
	//	defer nd.InProg.mtx.Lock()
	if nd.InProg.PeerIdx != 0 {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("fund with peer %d not done yet", nd.InProg.PeerIdx)
	}

	if initSend < 0 || ccap < 0 {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("Can't have negative send or capacity")
	}
	if ccap < 1000000 { // limit for now
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("Min channel capacity 1M sat")
	}
	if initSend > ccap {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("Cant send %d in %d capacity channel", initSend, ccap)
	}

	// TODO - would be convenient if it auto connected to the peer huh
	if !nd.ConnectedToPeer(peerIdx) {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("Not connected to peer %d. Do that yourself.", peerIdx)
	}

	cIdx, err := nd.NextChannelIdx()
	if err != nil {
		nd.InProg.mtx.Unlock()
		return 0, err
	}

	nd.InProg.ChanIdx = cIdx
	nd.InProg.PeerIdx = peerIdx
	nd.InProg.Amt = ccap
	nd.InProg.InitSend = initSend
	nd.InProg.mtx.Unlock() // switch to defer

	cointype := uint32(0)
	outMsg := lnutil.NewPointReqMsg(peerIdx, cointype)

	nd.OmniOut <- outMsg

	// wait until it's done!
	idx := <-nd.InProg.done
	return idx, nil
}

// RECIPIENT
// PubReqHandler gets a (content-less) pubkey request.  Respond with a pubkey
// and a refund pubkey hash. (currently makes pubkey hash, need to only make 1)
// so if someone sends 10 pubkeyreqs, they'll get the same pubkey back 10 times.
// they have to provide an actual tx before the next pubkey will come out.
func (nd *LitNode) PointReqHandler(msg lnutil.PointReqMsg) {

	/* shouldn't be possible to get this error...
	if nd.RemoteCon == nil || nd.RemoteCon.RemotePub == nil {
		fmt.Printf("Not connected to anyone\n")
		return
	}*/

	// pub req; check that idx matches next idx of ours and create pubkey
	// peerArr, _ := nd.GetPubHostFromPeerIdx(msg.Peer())

	cIdx, err := nd.NextChannelIdx()
	if err != nil {
		fmt.Printf("PointReqHandler err %s", err.Error())
		return
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = 0 | 1<<31
	kg.Step[2] = UseChannelFund
	kg.Step[3] = msg.Peer() | 1<<31
	kg.Step[4] = cIdx | 1<<31

	myChanPub := nd.GetUsePub(kg, UseChannelFund)
	myRefundPub := nd.GetUsePub(kg, UseChannelRefund)
	myHAKDbase := nd.GetUsePub(kg, UseChannelHAKDBase)
	fmt.Printf("Generated channel pubkey %x\n", myChanPub)

	outMsg := lnutil.NewPointRespMsg(msg.Peer(), myChanPub, myRefundPub, myHAKDbase)
	nd.OmniOut <- outMsg
	outMsg.Bytes()

	return
}

// FUNDER
// PointRespHandler takes in a point response, and returns a channel description
func (nd LitNode) PointRespHandler(msg lnutil.PointRespMsg) error {

	nd.InProg.mtx.Lock()
	defer nd.InProg.mtx.Unlock()

	if nd.InProg.PeerIdx == 0 {
		return fmt.Errorf("Got point response but no channel creation in progress")
	}

	if nd.InProg.PeerIdx != msg.Peer() {
		return fmt.Errorf(
			"making channel with peer %d but got PointResp from %d",
			nd.InProg.PeerIdx, msg.Peer())
	}

	// make channel (not in db) just for keys / elk
	qc := new(Qchan)

	qc.Height = -1

	qc.Value = nd.InProg.Amt

	qc.KeyGen.Depth = 5
	qc.KeyGen.Step[0] = 44 | 1<<31
	qc.KeyGen.Step[1] = 0 | 1<<31
	qc.KeyGen.Step[2] = UseChannelFund
	qc.KeyGen.Step[3] = nd.InProg.PeerIdx | 1<<31
	qc.KeyGen.Step[4] = nd.InProg.ChanIdx | 1<<31

	qc.MyPub = nd.GetUsePub(qc.KeyGen, UseChannelFund)
	qc.MyRefundPub = nd.GetUsePub(qc.KeyGen, UseChannelRefund)
	qc.MyHAKDBase = nd.GetUsePub(qc.KeyGen, UseChannelHAKDBase)

	// chop up incoming message, save points to channel struct
	copy(qc.TheirPub[:], msg.ChannelPub[:])
	copy(qc.TheirRefundPub[:], msg.RefundPub[:])
	copy(qc.TheirHAKDBase[:], msg.HAKDbase[:])

	// make sure their pubkeys are real pubkeys
	_, err := btcec.ParsePubKey(qc.TheirPub[:], btcec.S256())
	if err != nil {
		return fmt.Errorf("PubRespHandler TheirPub err %s", err.Error())
	}
	_, err = btcec.ParsePubKey(qc.TheirRefundPub[:], btcec.S256())
	if err != nil {
		return fmt.Errorf("PubRespHandler TheirRefundPub err %s", err.Error())
	}
	_, err = btcec.ParsePubKey(qc.TheirHAKDBase[:], btcec.S256())
	if err != nil {
		return fmt.Errorf("PubRespHandler TheirHAKDBase err %s", err.Error())
	}

	// derive elkrem sender root from HD keychain
	qc.ElkSnd = elkrem.NewElkremSender(nd.GetElkremRoot(qc.KeyGen))

	// get txo for channel
	txo, err := lnutil.FundTxOut(qc.MyPub, qc.TheirPub, nd.InProg.Amt)
	if err != nil {
		return err
	}

	// call MaybeSend, freezing inputs and learning the txid of the channel
	// here, we require only witness inputs
	outPoints, err := nd.SubWallet.MaybeSend([]*wire.TxOut{txo}, true)
	if err != nil {
		return err
	}

	// should only have 1 txout index from MaybeSend, which we use
	if len(outPoints) != 1 {
		return fmt.Errorf("got %d OPs from MaybeSend (expect 1)", len(outPoints))
	}

	// save fund outpoint to inProg
	nd.InProg.op = outPoints[0]
	// also set outpoint in channel
	qc.Op = *nd.InProg.op

	// create initial state for elkrem points
	qc.State = new(StatCom)
	qc.State.StateIdx = 0
	qc.State.MyAmt = nd.InProg.Amt - nd.InProg.InitSend

	// save channel to db
	err = nd.SaveQChan(qc)
	if err != nil {
		return fmt.Errorf("PointRespHandler SaveQchanState err %s", err.Error())
	}

	// when funding a channel, give them the first *3* elkpoints.
	elkPointZero, err := qc.ElkPoint(false, 0)
	if err != nil {
		return err
	}
	elkPointOne, err := qc.ElkPoint(false, 1)
	if err != nil {
		return err
	}

	elkPointTwo, err := qc.N2ElkPointForThem()
	if err != nil {
		return err
	}

	// description is outpoint (36), mypub(33), myrefund(33),
	// myHAKDbase(33), capacity (8),
	// initial payment (8), ElkPoint0,1,2 (99)
	// total length 250

	outMsg := lnutil.NewChanDescMsg(msg.Peer(), *nd.InProg.op, qc.MyPub, qc.MyRefundPub,
		qc.MyHAKDBase, nd.InProg.Amt, nd.InProg.InitSend, elkPointZero, elkPointOne, elkPointTwo)

	nd.OmniOut <- outMsg

	return nil
}

// RECIPIENT
// QChanDescHandler takes in a description of a channel output.  It then
// saves it to the local db, and returns a channel acknowledgement
func (nd *LitNode) QChanDescHandler(msg lnutil.ChanDescMsg) {

	// deserialize desc
	op := msg.Outpoint
	opArr := lnutil.OutPointToBytes(op)
	amt := msg.Capacity

	cIdx, err := nd.NextChannelIdx()
	if err != nil {
		fmt.Printf("QChanDescHandler err %s", err.Error())
		return
	}

	qc := new(Qchan)

	qc.Height = -1
	qc.KeyGen.Depth = 5
	qc.KeyGen.Step[0] = 44 | 1<<31
	qc.KeyGen.Step[1] = 0 | 1<<31
	qc.KeyGen.Step[2] = UseChannelFund
	qc.KeyGen.Step[3] = msg.Peer() | 1<<31
	qc.KeyGen.Step[4] = cIdx | 1<<31
	qc.Value = amt
	qc.Mode = portxo.TxoP2WSHComp
	qc.Op = op

	qc.MyPub = nd.GetUsePub(qc.KeyGen, UseChannelFund)
	qc.TheirPub = msg.PubKey
	qc.TheirRefundPub = msg.RefundPub
	qc.TheirHAKDBase = msg.HAKDbase
	qc.MyRefundPub = nd.GetUsePub(qc.KeyGen, UseChannelRefund)
	qc.MyHAKDBase = nd.GetUsePub(qc.KeyGen, UseChannelHAKDBase)

	// it should go into the next bucket and get the right key index.
	// but we can't actually check that.
	//	qc, err := nd.SaveFundTx(
	//		op, amt, peerArr, theirPub, theirRefundPub, theirHAKDbase)
	//	if err != nil {
	//		fmt.Printf("QChanDescHandler SaveFundTx err %s", err.Error())
	//		return
	//	}
	fmt.Printf("got multisig output %s amt %d\n", op.String(), amt)

	// create initial state
	qc.State = new(StatCom)
	// similar to SIGREV in pushpull
	qc.State.MyAmt = msg.InitPayment
	qc.State.StateIdx = 0
	// use new ElkPoint for signing
	qc.State.ElkPoint = msg.ElkZero
	qc.State.NextElkPoint = msg.ElkOne
	qc.State.N2ElkPoint = msg.ElkTwo

	// create empty elkrem receiver to save
	//	qc.ElkRcv = new(elkrem.ElkremReceiver)
	//	err = qc.IngestElkrem(revElk)
	//	if err != nil { // this can't happen because it's the first elk... remove?
	//		fmt.Printf("QChanDescHandler err %s", err.Error())
	//		return
	//	}

	// save new channel to db
	err = nd.SaveQChan(qc)
	if err != nil {
		fmt.Printf("QChanDescHandler err %s", err.Error())
		return
	}

	// load ... the thing I just saved.  why?
	qc, err = nd.GetQchan(opArr)
	if err != nil {
		fmt.Printf("QChanDescHandler GetQchan err %s", err.Error())
		return
	}

	// when funding a channel, give them the first *2* elkpoints.
	theirElkPointZero, err := qc.ElkPoint(false, 0)
	if err != nil {
		fmt.Printf("QChanDescHandler err %s", err.Error())
		return
	}
	theirElkPointOne, err := qc.ElkPoint(false, 1)
	if err != nil {
		fmt.Printf("QChanDescHandler err %s", err.Error())
		return
	}

	theirElkPointTwo, err := qc.N2ElkPointForThem()
	if err != nil {
		fmt.Printf("QChanDescHandler err %s", err.Error())
		return
	}

	sig, err := nd.SignState(qc)
	if err != nil {
		fmt.Printf("QChanDescHandler SignState err %s", err.Error())
		return
	}

	//	elk, err := qc.ElkSnd.AtIndex(qc.State.StateIdx - 1) // which is 0
	//	if err != nil {
	//		fmt.Printf("QChanDescHandler ElkSnd err %s", err.Error())
	//		return
	//	}
	// ACK the channel address, which causes the funder to sign / broadcast
	// ACK is outpoint (36), ElkPoint0,1,2 (99) and signature (64)

	outMsg := lnutil.NewChanAckMsg(msg.Peer(), op, theirElkPointZero, theirElkPointOne, theirElkPointTwo, sig)
	outMsg.Bytes()

	nd.OmniOut <- outMsg

	return
}

// FUNDER
// QChanAckHandler takes in an acknowledgement multisig description.
// when a multisig outpoint is ackd, that causes the funder to sign and broadcast.
func (nd *LitNode) QChanAckHandler(msg lnutil.ChanAckMsg, peer *RemotePeer) {
	opArr := lnutil.OutPointToBytes(msg.Outpoint)
	sig := msg.Signature

	// load channel to save their refund address
	qc, err := nd.GetQchan(opArr)
	if err != nil {
		fmt.Printf("QChanAckHandler GetQchan err %s", err.Error())
		return
	}

	//	err = qc.IngestElkrem(revElk)
	//	if err != nil { // this can't happen because it's the first elk... remove?
	//		fmt.Printf("QChanAckHandler IngestElkrem err %s", err.Error())
	//		return
	//	}
	qc.State.ElkPoint = msg.ElkZero
	qc.State.NextElkPoint = msg.ElkOne
	qc.State.N2ElkPoint = msg.ElkTwo

	err = qc.VerifySig(sig)
	if err != nil {
		fmt.Printf("QChanAckHandler VerifySig err %s", err.Error())
		return
	}

	// verify worked; Save state 1 to DB
	err = nd.SaveQchanState(qc)
	if err != nil {
		fmt.Printf("QChanAckHandler SaveQchanState err %s", err.Error())
		return
	}

	// Make sure everything works & is saved, then clear InProg.

	// sign their com tx to send
	sig, err = nd.SignState(qc)
	if err != nil {
		fmt.Printf("QChanAckHandler SignState err %s", err.Error())
		return
	}

	// OK to fund.
	err = nd.SubWallet.ReallySend(&qc.Op.Hash)
	if err != nil {
		fmt.Printf("QChanAckHandler ReallySend err %s", err.Error())
		return
	}

	err = nd.SubWallet.WatchThis(qc.Op)
	if err != nil {
		fmt.Printf("QChanAckHandler WatchThis err %s", err.Error())
		return
	}

	// tell base wallet about watcher refund address in case that happens
	nullTxo := new(portxo.PorTxo)
	nullTxo.Value = 0 // redundant, but explicitly show that this is just for adr
	nullTxo.KeyGen = qc.KeyGen
	nullTxo.KeyGen.Step[2] = UseChannelWatchRefund
	nd.SubWallet.ExportUtxo(nullTxo)

	// channel creation is ~complete, clear InProg.
	// We may be asked to re-send the sig-proof

	nd.InProg.mtx.Lock()
	nd.InProg.done <- qc.KeyGen.Step[4] & 0x7fffffff
	nd.InProg.Clear()
	nd.InProg.mtx.Unlock()

	peer.QCs[qc.Idx()] = qc
	peer.OpMap[opArr] = qc.Idx()

	// sig proof should be sent later once there are confirmations.
	// it'll have an spv proof of the fund tx.
	// but for now just send the sig.

	outMsg := lnutil.NewSigProofMsg(msg.Peer(), msg.Outpoint, sig)

	nd.OmniOut <- outMsg

	return
}

// RECIPIENT
// SigProofHandler saves the signature the recipent stores.
// In some cases you don't need this message.
func (nd *LitNode) SigProofHandler(msg lnutil.SigProofMsg, peer *RemotePeer) {

	op := msg.Outpoint
	opArr := lnutil.OutPointToBytes(op)

	qc, err := nd.GetQchan(opArr)
	if err != nil {
		fmt.Printf("SigProofHandler err %s", err.Error())
		return
	}

	err = qc.VerifySig(msg.Signature)
	if err != nil {
		fmt.Printf("SigProofHandler err %s", err.Error())
		return
	}

	// sig OK, save
	err = nd.SaveQchanState(qc)
	if err != nil {
		fmt.Printf("SigProofHandler err %s", err.Error())
		return
	}

	err = nd.SubWallet.WatchThis(op)
	if err != nil {
		fmt.Printf("SigProofHandler err %s", err.Error())
		return
	}

	// tell base wallet about watcher refund address in case that happens
	nullTxo := new(portxo.PorTxo)
	nullTxo.Value = 0 // redundant, but explicitly show that this is just for adr
	nullTxo.KeyGen = qc.KeyGen
	nullTxo.KeyGen.Step[2] = UseChannelWatchRefund
	nd.SubWallet.ExportUtxo(nullTxo)

	peer.QCs[qc.Idx()] = qc
	peer.OpMap[opArr] = qc.Idx()

	// sig OK; in terms of UI here's where you can say "payment received"
	// "channel online" etc
	return
}
