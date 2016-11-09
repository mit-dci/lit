package qln

import (
	"fmt"

	"github.com/mit-dci/lit/uspv"
)

/* CloseChannel --- cooperative close
This is the simplified close which sends to the same outputs as a break tx,
just with no timeouts.

Users might want a more advanced close function which allows multiple outputs.
They can exchange txouts and sigs.  That could be "fancyClose", but this is
just close, so only a signature is sent by the initiator, and the receiver
doesn't reply, as the channel is closed.

*/

// CloseReqHandler takes in a close request from a remote host, signs and
// responds with a close response.  Obviously later there will be some judgment
// over what to do, but for now it just signs whatever it's requested to.
func (nd *LnNode) CloseReqHandler(from [16]byte, reqbytes []byte) {
	if len(reqbytes) < 100 {
		fmt.Printf("got %d byte closereq, expect 100ish\n", len(reqbytes))
		return
	}

	// figure out who we're talking to
	var peerArr [33]byte
	copy(peerArr[:], nd.RemoteCon.RemotePub.SerializeCompressed())

	// deserialize outpoint
	var opArr [36]byte
	copy(opArr[:], reqbytes[:36])

	// find their sig
	theirSig := reqbytes[36:]

	// get channel
	qc, err := nd.GetQchan(peerArr, opArr)
	if err != nil {
		fmt.Printf("CloseReqHandler GetQchan err %s", err.Error())
		return
	}
	// verify their sig?  should do that before signing our side just to be safe

	// build close tx
	tx, err := qc.SimpleCloseTx()
	if err != nil {
		fmt.Printf("CloseReqHandler SimpleCloseTx err %s", err.Error())
		return
	}

	// sign close
	mySig, err := nd.SignSimpleClose(qc, tx)
	if err != nil {
		fmt.Printf("CloseReqHandler SignSimpleClose err %s", err.Error())
		return
	}
	pre, swap, err := FundTxScript(qc.MyPub, qc.TheirPub)
	if err != nil {
		fmt.Printf("CloseReqHandler FundTxScript err %s", err.Error())
		return
	}

	// swap if needed
	if swap {
		tx.TxIn[0].Witness = SpendMultiSigWitStack(pre, theirSig, mySig)
	} else {
		tx.TxIn[0].Witness = SpendMultiSigWitStack(pre, mySig, theirSig)
	}
	fmt.Printf(uspv.TxToString(tx))
	// broadcast
	err = nd.BaseWallet.PushTx(tx)
	if err != nil {
		fmt.Printf("CloseReqHandler NewOutgoingTx err %s", err.Error())
		return
	}

	return
}
