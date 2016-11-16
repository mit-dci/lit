package qln

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
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

// RECIPIENT
// PubReqHandler gets a (content-less) pubkey request.  Respond with a pubkey
// and a refund pubkey hash. (currently makes pubkey hash, need to only make 1)
// so if someone sends 10 pubkeyreqs, they'll get the same pubkey back 10 times.
// they have to provide an actual tx before the next pubkey will come out.
func (nd *LnNode) PointReqHandler(from [16]byte, pointReqBytes []byte) {
	// pub req; check that idx matches next idx of ours and create pubkey
	var peerArr [33]byte
	copy(peerArr[:], nd.RemoteCon.RemotePub.SerializeCompressed())

	peerIdx, cIdx, err := nd.NextIdxForPeer(peerArr)
	if err != nil {
		fmt.Printf("PointReqHandler err %s", err.Error())
		return
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = 0 | 1<<31
	kg.Step[2] = UseChannelFund
	kg.Step[3] = peerIdx | 1<<31
	kg.Step[4] = cIdx | 1<<31

	myChanPub := nd.GetUsePub(kg, UseChannelFund)
	myRefundPub := nd.GetUsePub(kg, UseChannelRefund)
	myHAKDbase := nd.GetUsePub(kg, UseChannelHAKDBase)
	fmt.Printf("Generated channel pubkey %x\n", myChanPub)

	msg := []byte{MSGID_POINTRESP}
	msg = append(msg, myChanPub[:]...)
	msg = append(msg, myRefundPub[:]...)
	msg = append(msg, myHAKDbase[:]...)
	_, err = nd.RemoteCon.Write(msg)
	return
}

// FUNDER
// PointRespHandler takes in a point response, and returns a channel description
func (nd LnNode) PointRespHandler(from [16]byte, pointRespBytes []byte) error {
	// not sure how to do this yet

	if nd.InProg.PeerIdx == 0 {
		return fmt.Errorf("Got point response but no channel creation in progress")
	}

	if len(pointRespBytes) != 99 {
		return fmt.Errorf("PointRespHandler err: pointRespBytes %d bytes, expect 99\n",
			len(pointRespBytes))
	}

	// should put these pubkeys somewhere huh.

	var peerArr [33]byte
	copy(peerArr[:], nd.RemoteCon.RemotePub.SerializeCompressed())

	peerIdx, err := nd.GetPeerIdx(nd.RemoteCon.RemotePub)
	if err != nil {
		return err
	}
	if nd.InProg.PeerIdx != peerIdx {
		return fmt.Errorf("making channel with peer %d but got PointResp from %d")
	}

	// make channel (not in db) just for keys / elk
	qc := new(Qchan)

	qc.PeerId = peerArr
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
	copy(qc.TheirPub[:], pointRespBytes[:33])
	copy(qc.TheirRefundPub[:], pointRespBytes[33:66])
	copy(qc.TheirHAKDBase[:], pointRespBytes[66:])

	// We have their & our pubkeys; can make DHmask
	qc.DHmask = nd.GetDHMask(qc)
	if qc.DHmask&1<<63 != 0 { // crash if high bits set
		return fmt.Errorf("GetDHMask error")
	}

	// make sure their pubkeys are real pubkeys
	_, err = btcec.ParsePubKey(qc.TheirPub[:], btcec.S256())
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
	txo, err := FundTxOut(qc.MyPub, qc.TheirPub, nd.InProg.Amt)
	if err != nil {
		return err
	}

	// call MaybeSend, freezing inputs and learning the txid of the channel
	outPoints, err := nd.BaseWallet.MaybeSend([]*wire.TxOut{txo})
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

	// should watch for this tx.  Maybe after broadcasting

	opArr := lnutil.OutPointToBytes(*nd.InProg.op)

	// create initial state for elkrem points
	qc.State = new(StatCom)
	qc.State.StateIdx = 1
	qc.State.MyAmt = nd.InProg.Amt - nd.InProg.InitSend

	// save channel to db
	err = nd.SaveQChan(qc)
	if err != nil {
		return fmt.Errorf("PointRespHandler SaveQchanState err %s", err.Error())
	}

	theirElkPointR, theirElkPointT, err := qc.MakeTheirCurElkPoints()
	if err != nil {
		return err
	}

	elk, err := qc.ElkSnd.AtIndex(0)
	if err != nil {
		return err
	}

	initPayBytes := lnutil.I64tB(nd.InProg.InitSend) // also will be an arg
	capBytes := lnutil.I64tB(nd.InProg.Amt)

	// description is outpoint (36), mypub(33), myrefund(33),
	// myHAKDbase(33), capacity (8),
	// initial payment (8), ElkPointR (33), ElkPointT (33), elk0 (32)
	// total length 249
	msg := []byte{MSGID_CHANDESC}
	msg = append(msg, opArr[:]...)
	msg = append(msg, qc.MyPub[:]...)
	msg = append(msg, qc.MyRefundPub[:]...)
	msg = append(msg, qc.MyHAKDBase[:]...)
	msg = append(msg, capBytes...)
	msg = append(msg, initPayBytes...)
	msg = append(msg, theirElkPointR[:]...)
	msg = append(msg, theirElkPointT[:]...)
	msg = append(msg, elk.CloneBytes()...)
	_, err = nd.RemoteCon.Write(msg)

	return nil
}

// RECIPIENT
// QChanDescHandler takes in a description of a channel output.  It then
// saves it to the local db, and returns a channel acknowledgement
func (nd *LnNode) QChanDescHandler(from [16]byte, descbytes []byte) {
	if len(descbytes) < 249 || len(descbytes) > 249 {
		fmt.Printf("got %d byte channel description, expect 249", len(descbytes))
		return
	}
	var peerArr, myFirstElkPointR, myFirstElkPointT, theirPub, theirRefundPub, theirHAKDbase [33]byte
	var opArr [36]byte
	copy(peerArr[:], nd.RemoteCon.RemotePub.SerializeCompressed())

	// deserialize desc
	copy(opArr[:], descbytes[:36])
	op := lnutil.OutPointFromBytes(opArr)
	copy(theirPub[:], descbytes[36:69])
	copy(theirRefundPub[:], descbytes[69:102])
	copy(theirHAKDbase[:], descbytes[102:135])
	amt := lnutil.BtI64(descbytes[135:143])
	initPay := lnutil.BtI64(descbytes[143:151])
	copy(myFirstElkPointR[:], descbytes[151:184])
	copy(myFirstElkPointT[:], descbytes[184:217])
	revElk, err := chainhash.NewHash(descbytes[217:])
	if err != nil {
		fmt.Printf("QChanDescHandler SaveFundTx err %s", err.Error())
		return
	}

	peerIdx, cIdx, err := nd.NextIdxForPeer(peerArr)
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
	qc.KeyGen.Step[3] = peerIdx | 1<<31
	qc.KeyGen.Step[4] = cIdx | 1<<31
	qc.Value = amt
	qc.Mode = portxo.TxoP2WSHComp
	qc.Op = *op

	qc.MyPub = nd.GetUsePub(qc.KeyGen, UseChannelFund)
	qc.TheirPub = theirPub
	qc.PeerId = peerArr
	qc.TheirRefundPub = theirRefundPub
	qc.TheirHAKDBase = theirHAKDbase
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
	qc.State.MyAmt = initPay
	qc.State.StateIdx = 1
	// use new ElkPoint for signing
	qc.State.ElkPointR = myFirstElkPointR
	qc.State.ElkPointT = myFirstElkPointT

	// create empty elkrem receiver to save
	qc.ElkRcv = new(elkrem.ElkremReceiver)
	err = qc.IngestElkrem(revElk)
	if err != nil { // this can't happen because it's the first elk... remove?
		fmt.Printf("QChanDescHandler err %s", err.Error())
		return
	}

	// save new channel to db
	err = nd.SaveQChan(qc)
	if err != nil {
		fmt.Printf("QChanDescHandler err %s", err.Error())
		return
	}

	//	err = nd.SaveQchanState(qc)
	//	if err != nil {
	//		fmt.Printf("QChanDescHandler SaveQchanState err %s", err.Error())
	//		return
	//	}
	// load ... the thing I just saved.  why?
	qc, err = nd.GetQchan(peerArr, opArr)
	if err != nil {
		fmt.Printf("QChanDescHandler GetQchan err %s", err.Error())
		return
	}

	theirElkPointR, theirElkPointT, err := qc.MakeTheirCurElkPoints()
	if err != nil {
		fmt.Printf("QChanDescHandler MakeTheirCurElkPoint err %s", err.Error())
		return
	}

	sig, err := nd.SignState(qc)
	if err != nil {
		fmt.Printf("QChanDescHandler SignState err %s", err.Error())
		return
	}

	elk, err := qc.ElkSnd.AtIndex(qc.State.StateIdx - 1) // which is 0
	if err != nil {
		fmt.Printf("QChanDescHandler ElkSnd err %s", err.Error())
		return
	}
	// ACK the channel address, which causes the funder to sign / broadcast
	// ACK is outpoint (36), ElkPointR (33), ElkPointT (33), elk (32) and signature (64)
	msg := []byte{MSGID_CHANACK}
	msg = append(msg, opArr[:]...)
	msg = append(msg, theirElkPointR[:]...)
	msg = append(msg, theirElkPointT[:]...)
	msg = append(msg, elk.CloneBytes()...)
	msg = append(msg, sig[:]...)
	_, err = nd.RemoteCon.Write(msg)
	return
}

// FUNDER
// QChanAckHandler takes in an acknowledgement multisig description.
// when a multisig outpoint is ackd, that causes the funder to sign and broadcast.
func (nd *LnNode) QChanAckHandler(from [16]byte, ackbytes []byte) {
	if len(ackbytes) < 198 || len(ackbytes) > 198 {
		fmt.Printf("got %d byte multiAck, expect 198", len(ackbytes))
		return
	}
	var opArr [36]byte
	var peerArr, myFirstElkPointR, myFirstElkPointT [33]byte
	var sig [64]byte

	copy(peerArr[:], nd.RemoteCon.RemotePub.SerializeCompressed())
	// deserialize chanACK
	copy(opArr[:], ackbytes[:36])
	copy(myFirstElkPointR[:], ackbytes[36:69])
	copy(myFirstElkPointT[:], ackbytes[69:102])
	// don't think this can error as length is specified
	revElk, _ := chainhash.NewHash(ackbytes[102:134])
	copy(sig[:], ackbytes[134:])

	//	op := lnutil.OutPointFromBytes(opArr)

	// load channel to save their refund address
	qc, err := nd.GetQchan(peerArr, opArr)
	if err != nil {
		fmt.Printf("QChanAckHandler GetQchan err %s", err.Error())
		return
	}

	err = qc.IngestElkrem(revElk)
	if err != nil { // this can't happen because it's the first elk... remove?
		fmt.Printf("QChanAckHandler IngestElkrem err %s", err.Error())
		return
	}
	qc.State.ElkPointR = myFirstElkPointR
	qc.State.ElkPointT = myFirstElkPointT

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

	// kindof want to clear inFlight and call ReallySend atomically, but clear
	// inFlight first.  Worst case you've got some stuck / invalid non-channel,
	// but no loss of funds.

	nd.InProg.Clear()

	// sign their com tx to send
	sig, err = nd.SignState(qc)
	if err != nil {
		fmt.Printf("QChanAckHandler SignState err %s", err.Error())
		return
	}

	// OK to fund.
	err = nd.BaseWallet.ReallySend(&qc.Op.Hash)
	if err != nil {
		fmt.Printf("QChanAckHandler ReallySend err %s", err.Error())
		return
	}

	// sig proof should be sent later once there are confirmations.
	// it'll have an spv proof of the fund tx.
	// but for now just send the sig.
	msg := []byte{MSGID_SIGPROOF}
	msg = append(msg, opArr[:]...)
	msg = append(msg, sig[:]...)
	_, err = nd.RemoteCon.Write(msg)
	return
}

// RECIPIENT
// SigProofHandler saves the signature the recipent stores.
// In some cases you don't need this message.
func (nd *LnNode) SigProofHandler(from [16]byte, sigproofbytes []byte) {
	if len(sigproofbytes) < 100 || len(sigproofbytes) > 100 {
		fmt.Printf("got %d byte Sigproof, expect ~100\n", len(sigproofbytes))
		return
	}
	var peerArr [33]byte
	var opArr [36]byte
	var sig [64]byte
	copy(peerArr[:], nd.RemoteCon.RemotePub.SerializeCompressed())
	copy(opArr[:], sigproofbytes[:36])
	copy(sig[:], sigproofbytes[36:])

	qc, err := nd.GetQchan(peerArr, opArr)
	if err != nil {
		fmt.Printf("SigProofHandler err %s", err.Error())
		return
	}

	err = qc.VerifySig(sig)
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
	op := lnutil.OutPointFromBytes(opArr)
	err = nd.BaseWallet.WatchThis(*op)
	if err != nil {
		fmt.Printf("SigProofHandler err %s", err.Error())
		return
	}
	// add to bloom filter here; later should instead receive spv proof
	//	filt, err := SCon.TS.GimmeFilter()
	//	if err != nil {
	//		fmt.Printf("QChanDescHandler RefilterLocal err %s", err.Error())
	//		return
	//	}
	//	SCon.Refilter(filt)

	// sig OK; in terms of UI here's where you can say "payment received"
	// "channel online" etc
	return
}
