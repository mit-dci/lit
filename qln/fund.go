package qln

import (
	"fmt"
	"log"

	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/wire"
	"github.com/mit-dci/lit/consts"
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
committed to  (there are no HAKDpubs for state 0; those start at state 1.)
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
pubkey calculation.  In the second it's independent of the fund process.

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
func (nd *LitNode) FundChannel(
	peerIdx, cointype uint32, ccap, initSend int64, data [32]byte) (uint32, error) {

	_, ok := nd.SubWallet[cointype]
	if !ok {
		return 0, fmt.Errorf("No wallet of type %d connected", cointype)
	}

	nd.InProg.mtx.Lock()
	//	defer nd.InProg.mtx.Lock()

	_, ok = nd.ConnectedCoinTypes[cointype] ; if !ok {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("No daemon of type %d connected. Can't fund, only receive", cointype)
	}

	if nd.InProg.PeerIdx != 0 {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("fund with peer %d not done yet", nd.InProg.PeerIdx)
	}

	if initSend < 0 || ccap < 0 {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("Can't have negative send or capacity")
	}
	if ccap < consts.MinChanCapacity { // limit for now
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("Min channel capacity 1M sat")
	}
	if initSend > ccap {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("Can't send %d in %d capacity channel", initSend, ccap)
	}

	if initSend < consts.MinOutput {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("Can't send %d as initial send because MinOutput is %d", initSend, consts.MinOutput)
	}

	if ccap-initSend < consts.MinOutput {
		nd.InProg.mtx.Unlock()
		return 0, fmt.Errorf("Can't send %d as initial send because MinOutput is %d and you would only have %d", initSend, consts.MinOutput, ccap-initSend)
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
	nd.InProg.Data = data

	nd.InProg.Coin = cointype
	nd.InProg.mtx.Unlock() // switch to defer

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
		log.Printf("Not connected to anyone\n")
		return
	}*/

	// pub req; check that idx matches next idx of ours and create pubkey
	// peerArr, _ := nd.GetPubHostFromPeerIdx(msg.Peer())

	cIdx, err := nd.NextChannelIdx()
	if err != nil {
		log.Printf("PointReqHandler err %s", err.Error())
		return
	}

	_, ok := nd.SubWallet[msg.Cointype]
	if !ok {
		log.Printf("PointReqHandler err no wallet for type %d", msg.Cointype)
		return
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = msg.Cointype | 1<<31
	kg.Step[2] = UseChannelFund
	kg.Step[3] = msg.Peer() | 1<<31
	kg.Step[4] = cIdx | 1<<31

	myChanPub, _ := nd.GetUsePub(kg, UseChannelFund)
	myRefundPub, _ := nd.GetUsePub(kg, UseChannelRefund)
	myHAKDbase, err := nd.GetUsePub(kg, UseChannelHAKDBase)
	if err != nil {
		log.Printf("PointReqHandler err %s", err.Error())
		return
	}

	log.Printf("Generated channel pubkey %x\n", myChanPub)

	var keyGen portxo.KeyGen
	keyGen.Depth = 5
	keyGen.Step[0] = 44 | 1<<31
	keyGen.Step[1] = msg.Cointype | 1<<31
	keyGen.Step[2] = UseHTLCBase
	keyGen.Step[3] = 0 | 1<<31
	keyGen.Step[4] = cIdx | 1<<31

	myNextHTLCBase, err := nd.GetUsePub(keyGen, UseHTLCBase)
	if err != nil {
		log.Printf("error generating NextHTLCBase %v", err)
		return
	}

	keyGen.Step[3] = 1 | 1<<31
	myN2HTLCBase, err := nd.GetUsePub(keyGen, UseHTLCBase)
	if err != nil {
		log.Printf("error generating N2HTLCBase %v", err)
		return
	}

	outMsg := lnutil.NewPointRespMsg(msg.Peer(), myChanPub, myRefundPub, myHAKDbase,
		myNextHTLCBase, myN2HTLCBase)
	nd.OmniOut <- outMsg

	return
}

// FUNDER
// PointRespHandler takes in a point response, and returns a channel description
func (nd *LitNode) PointRespHandler(msg lnutil.PointRespMsg) error {
	log.Printf("Got PointResponse")

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

	if nd.SubWallet[nd.InProg.Coin] == nil {
		return fmt.Errorf("Not connected to coin type %d\n", nd.InProg.Coin)
	}

	// make channel (not in db) just for keys / elk
	q := new(Qchan)

	q.Height = -1

	q.Value = nd.InProg.Amt

	q.KeyGen.Depth = 5
	q.KeyGen.Step[0] = 44 | 1<<31
	q.KeyGen.Step[1] = nd.InProg.Coin | 1<<31
	q.KeyGen.Step[2] = UseChannelFund
	q.KeyGen.Step[3] = nd.InProg.PeerIdx | 1<<31
	q.KeyGen.Step[4] = nd.InProg.ChanIdx | 1<<31

	q.MyPub, _ = nd.GetUsePub(q.KeyGen, UseChannelFund)
	q.MyRefundPub, _ = nd.GetUsePub(q.KeyGen, UseChannelRefund)
	q.MyHAKDBase, _ = nd.GetUsePub(q.KeyGen, UseChannelHAKDBase)

	// chop up incoming message, save points to channel struct
	copy(q.TheirPub[:], msg.ChannelPub[:])
	copy(q.TheirRefundPub[:], msg.RefundPub[:])
	copy(q.TheirHAKDBase[:], msg.HAKDbase[:])

	// make sure their pubkeys are real pubkeys
	_, err := btcec.ParsePubKey(q.TheirPub[:], btcec.S256())
	if err != nil {
		return fmt.Errorf("PubRespHandler TheirPub err %s", err.Error())
	}
	_, err = btcec.ParsePubKey(q.TheirRefundPub[:], btcec.S256())
	if err != nil {
		return fmt.Errorf("PubRespHandler TheirRefundPub err %s", err.Error())
	}
	_, err = btcec.ParsePubKey(q.TheirHAKDBase[:], btcec.S256())
	if err != nil {
		return fmt.Errorf("PubRespHandler TheirHAKDBase err %s", err.Error())
	}

	// derive elkrem sender root from HD keychain
	elkRoot, _ := nd.GetElkremRoot(q.KeyGen)
	q.ElkSnd = elkrem.NewElkremSender(elkRoot)

	// get txo for channel
	txo, err := lnutil.FundTxOut(q.MyPub, q.TheirPub, nd.InProg.Amt)
	if err != nil {
		return err
	}

	// call MaybeSend, freezing inputs and learning the txid of the channel
	// here, we require only witness inputs
	outPoints, err := nd.SubWallet[q.Coin()].MaybeSend([]*wire.TxOut{txo}, true)
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
	q.Op = *nd.InProg.op

	// create initial state for elkrem points
	q.State = new(StatCom)
	q.State.StateIdx = 0
	q.State.MyAmt = nd.InProg.Amt - nd.InProg.InitSend
	// get fee from sub wallet.  Later should make fee per channel and update state
	// based on size
	q.State.Fee = nd.SubWallet[q.Coin()].Fee() * 1000

	q.State.Data = nd.InProg.Data

	_, err = btcec.ParsePubKey(msg.NextHTLCBase[:], btcec.S256())
	if err != nil {
		return fmt.Errorf("PubRespHandler NextHTLCBase err %s", err.Error())
	}
	_, err = btcec.ParsePubKey(msg.N2HTLCBase[:], btcec.S256())
	if err != nil {
		return fmt.Errorf("PubRespHandler N2HTLCBase err %s", err.Error())
	}

	var keyGen portxo.KeyGen
	keyGen.Depth = 5
	keyGen.Step[0] = 44 | 1<<31
	keyGen.Step[1] = nd.InProg.Coin | 1<<31
	keyGen.Step[2] = UseHTLCBase
	keyGen.Step[3] = 0 | 1<<31
	keyGen.Step[4] = nd.InProg.ChanIdx | 1<<31

	q.State.MyNextHTLCBase, err = nd.GetUsePub(keyGen, UseHTLCBase)
	if err != nil {
		return fmt.Errorf("error generating NextHTLCBase %v", err)
	}

	keyGen.Step[3] = 1 | 1<<31
	q.State.MyN2HTLCBase, err = nd.GetUsePub(keyGen, UseHTLCBase)
	if err != nil {
		return fmt.Errorf("error generating N2HTLCBase %v", err)
	}

	q.State.NextHTLCBase = msg.NextHTLCBase
	q.State.N2HTLCBase = msg.N2HTLCBase

	// save channel to db
	err = nd.SaveQChan(q)
	if err != nil {
		return fmt.Errorf("PointRespHandler SaveQchanState err %s", err.Error())
	}

	// when funding a channel, give them the first *3* elkpoints.
	elkPointZero, err := q.ElkPoint(false, 0)
	if err != nil {
		return err
	}
	elkPointOne, err := q.ElkPoint(false, 1)
	if err != nil {
		return err
	}

	elkPointTwo, err := q.N2ElkPointForThem()
	if err != nil {
		return err
	}

	// description is outpoint (36), mypub(33), myrefund(33),
	// myHAKDbase(33), capacity (8),
	// initial payment (8), ElkPoint0,1,2 (99)

	outMsg := lnutil.NewChanDescMsg(
		msg.Peer(), *nd.InProg.op, q.MyPub, q.MyRefundPub, q.MyHAKDBase,
		q.State.MyNextHTLCBase, q.State.MyN2HTLCBase,
		nd.InProg.Coin, nd.InProg.Amt, nd.InProg.InitSend,
		elkPointZero, elkPointOne, elkPointTwo, nd.InProg.Data)

	nd.OmniOut <- outMsg

	return nil
}

// RECIPIENT
// QChanDescHandler takes in a description of a channel output.  It then
// saves it to the local db, and returns a channel acknowledgement
func (nd *LitNode) QChanDescHandler(msg lnutil.ChanDescMsg) {

	wal, ok := nd.SubWallet[msg.CoinType]
	if !ok {
		log.Printf("QChanDescHandler err no wallet for type %d", msg.CoinType)
		return
	}

	// deserialize desc
	op := msg.Outpoint
	opArr := lnutil.OutPointToBytes(op)
	amt := msg.Capacity

	cIdx, err := nd.NextChannelIdx()
	if err != nil {
		log.Printf("QChanDescHandler err %s", err.Error())
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

	var keyGen portxo.KeyGen
	keyGen.Depth = 5
	keyGen.Step[0] = 44 | 1<<31
	keyGen.Step[1] = msg.CoinType | 1<<31
	keyGen.Step[2] = UseHTLCBase
	keyGen.Step[3] = 0 | 1<<31
	keyGen.Step[4] = cIdx | 1<<31

	qc.State.MyNextHTLCBase, err = nd.GetUsePub(keyGen, UseHTLCBase)
	if err != nil {
		fmt.Printf("error generating NextHTLCBase %v", err)
		return
	}

	keyGen.Step[3] = 1 | 1<<31
	qc.State.MyN2HTLCBase, err = nd.GetUsePub(keyGen, UseHTLCBase)
	if err != nil {
		fmt.Printf("error generating N2HTLCBase %v", err)
		return
	}

	qc.State.NextHTLCBase = msg.NextHTLCBase
	qc.State.N2HTLCBase = msg.N2HTLCBase

	// save new channel to db
	err = nd.SaveQChan(qc)
	if err != nil {
		log.Printf("QChanDescHandler err %s", err.Error())
		return
	}

	// load ... the thing I just saved.  why?
	qc, err = nd.GetQchan(opArr)
	if err != nil {
		log.Printf("QChanDescHandler GetQchan err %s", err.Error())
		return
	}

	// when funding a channel, give them the first *2* elkpoints.
	theirElkPointZero, err := qc.ElkPoint(false, 0)
	if err != nil {
		log.Printf("QChanDescHandler err %s", err.Error())
		return
	}
	theirElkPointOne, err := qc.ElkPoint(false, 1)
	if err != nil {
		log.Printf("QChanDescHandler err %s", err.Error())
		return
	}

	theirElkPointTwo, err := qc.N2ElkPointForThem()
	if err != nil {
		log.Printf("QChanDescHandler err %s", err.Error())
		return
	}

	sig, _, err := nd.SignState(qc)
	if err != nil {
		log.Printf("QChanDescHandler SignState err %s", err.Error())
		return
	}

	outMsg := lnutil.NewChanAckMsg(
		msg.Peer(), op,
		theirElkPointZero, theirElkPointOne, theirElkPointTwo,
		sig)
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
		log.Printf("QChanAckHandler GetQchan err %s", err.Error())
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
		log.Printf("QChanAckHandler VerifySig err %s", err.Error())
		return
	}

	// verify worked; Save state 1 to DB
	err = nd.SaveQchanState(qc)
	if err != nil {
		log.Printf("QChanAckHandler SaveQchanState err %s", err.Error())
		return
	}

	// Make sure everything works & is saved, then clear InProg.

	// sign their com tx to send
	sig, _, err = nd.SignState(qc)
	if err != nil {
		log.Printf("QChanAckHandler SignState err %s", err.Error())
		return
	}

	// OK to fund.
	err = nd.SubWallet[qc.Coin()].ReallySend(&qc.Op.Hash)
	if err != nil {
		log.Printf("QChanAckHandler ReallySend err %s", err.Error())
		return
	}

	err = nd.SubWallet[qc.Coin()].WatchThis(qc.Op)
	if err != nil {
		log.Printf("QChanAckHandler WatchThis err %s", err.Error())
		return
	}

	// tell base wallet about watcher refund address in case that happens
	// TODO this is weird & ugly... maybe have an export keypath func?
	nullTxo := new(portxo.PorTxo)
	nullTxo.Value = 0 // redundant, but explicitly show that this is just for adr
	nullTxo.KeyGen = qc.KeyGen
	nullTxo.KeyGen.Step[2] = UseChannelWatchRefund
	nd.SubWallet[qc.Coin()].ExportUtxo(nullTxo)

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
// SigProofHandler saves the signature the recipient stores.
// In some cases you don't need this message.
func (nd *LitNode) SigProofHandler(msg lnutil.SigProofMsg, peer *RemotePeer) {

	op := msg.Outpoint
	opArr := lnutil.OutPointToBytes(op)

	qc, err := nd.GetQchan(opArr)
	if err != nil {
		log.Printf("SigProofHandler err %s", err.Error())
		return
	}

	wal, ok := nd.SubWallet[qc.Coin()]
	if !ok {
		log.Printf("Not connected to coin type %d\n", qc.Coin())
		return
	}

	err = qc.VerifySigs(msg.Signature, nil)
	if err != nil {
		log.Printf("SigProofHandler err %s", err.Error())
		return
	}

	// sig OK, save
	err = nd.SaveQchanState(qc)
	if err != nil {
		log.Printf("SigProofHandler err %s", err.Error())
		return
	}

	err = wal.WatchThis(op)

	if err != nil {
		log.Printf("SigProofHandler err %s", err.Error())
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
	return
}
