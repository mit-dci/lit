package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"
)

func FundChannel(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("need args: fund capacity initialSend")
	}
	if LNode.RemoteCon == nil || LNode.RemoteCon.RemotePub == nil {
		return fmt.Errorf("Not connected to anyone")
	}

	if LNode.InProg != nil && LNode.InProg.PeerIdx != 0 {
		return fmt.Errorf("channel with peer %d not done yet", LNode.InProg.PeerIdx)
	}

	// this stuff is all the same as in cclose, should put into a function...
	cCap, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return err
	}
	iSend, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return err
	}
	if iSend < 0 || cCap < 0 {
		return fmt.Errorf("Can't have negative send or capacity")
	}
	if cCap < 1000000 { // limit for now
		return fmt.Errorf("Min channel capacity 1M sat")
	}
	if iSend > cCap {
		return fmt.Errorf("Cant send %d in %d capacity channel",
			iSend, cCap)
	}

	// see if we have enough money.  Doesn't freeze here though, just
	// checks for ability to fund.  Freeze happens when we receive the response.
	// Could fail if we run out of money before calling MaybeSend()
	_, _, err = SCon.TS.PickUtxos(cCap, true)
	if err != nil {
		return err
	}

	var peerArr [33]byte
	copy(peerArr[:], LNode.RemoteCon.RemotePub.SerializeCompressed())

	peerIdx, cIdx, err := LNode.NextIdxForPeer(peerArr)
	if err != nil {
		return err
	}

	LNode.InProg = new(qln.InFlightFund)

	LNode.InProg.ChanIdx = cIdx
	LNode.InProg.PeerIdx = peerIdx
	LNode.InProg.Amt = cCap
	LNode.InProg.InitSend = iSend

	msg := []byte{qln.MSGID_POINTREQ}
	_, err = LNode.RemoteCon.Write(msg)
	return err
}

// Resume is a shell command which resumes a message exchange for channels that
// are in a non-final state.  If the channel is in a final state it will send
// a REV (which it already sent, and should be ignored)
func Resume(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("need args: fix peerIdx chanIdx")
	}
	if LNode.RemoteCon == nil || LNode.RemoteCon.RemotePub == nil {
		return fmt.Errorf("Not connected to anyone, can't fix\n")
	}
	// this stuff is all the same as in cclose, should put into a function...
	peerIdx64, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return err
	}
	cIdx64, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return err
	}
	peerIdx := uint32(peerIdx64)
	cIdx := uint32(cIdx64)

	// find the peer index of who we're connected to
	currentPeerIdx, err := LNode.GetPeerIdx(LNode.RemoteCon.RemotePub)
	if err != nil {
		return err
	}
	if uint32(peerIdx) != currentPeerIdx {
		return fmt.Errorf("Want to close with peer %d but connected to %d",
			peerIdx, currentPeerIdx)
	}
	fmt.Printf("fix channel (%d,%d)\n", peerIdx, cIdx)

	qc, err := LNode.GetQchanByIdx(peerIdx, cIdx)
	if err != nil {
		return err
	}

	return LNode.SendNextMsg(qc)
}

// Push is the shell command which calls PushChannel
func Push(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("need args: push peerIdx chanIdx amt (times)")
	}
	if LNode.RemoteCon == nil || LNode.RemoteCon.RemotePub == nil {
		return fmt.Errorf("Not connected to anyone, can't push\n")
	}
	// this stuff is all the same as in cclose, should put into a function...
	peerIdx64, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return err
	}
	cIdx64, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return err
	}
	amt, err := strconv.ParseInt(args[2], 10, 32)
	if err != nil {
		return err
	}
	times := int64(1)
	if len(args) > 3 {
		times, err = strconv.ParseInt(args[3], 10, 32)
		if err != nil {
			return err
		}
	}

	if amt > 100000000 || amt < 1 {
		return fmt.Errorf("push %d, max push is 1 coin / 100000000", amt)
	}
	peerIdx := uint32(peerIdx64)
	cIdx := uint32(cIdx64)

	// find the peer index of who we're connected to
	currentPeerIdx, err := LNode.GetPeerIdx(LNode.RemoteCon.RemotePub)
	if err != nil {
		return err
	}
	if uint32(peerIdx) != currentPeerIdx {
		return fmt.Errorf("Want to push to peer %d but connected to %d",
			peerIdx, currentPeerIdx)
	}
	fmt.Printf("push %d to (%d,%d) %d times\n", amt, peerIdx, cIdx, times)

	qc, err := LNode.GetQchanByIdx(peerIdx, cIdx)
	if err != nil {
		return err
	}
	if qc.CloseData.Closed {
		return fmt.Errorf("channel %d, %d is closed.", peerIdx, cIdx64)
	}
	for times > 0 {
		err = LNode.ReloadQchan(qc)
		if err != nil {
			return err
		}

		err = LNode.PushChannel(qc, uint32(amt))
		if err != nil {
			return err
		}
		// such a hack.. obviously need indicator of when state update complete
		time.Sleep(time.Millisecond * 25)
		times--
	}
	return nil
}

// Watch is the command to set a channel as watched.
// Watched channels get exported to the watchtower.
func Watch(args []string) error {
	if len(args) > 1 {
		// send updated state to connected watch tower
		if LNode.WatchCon == nil {
			return fmt.Errorf("can't connect; no watch tower connected")
		}
		peerIdx64, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			return err
		}
		cIdx64, err := strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			return err
		}
		qc, err := LNode.GetQchanByIdx(uint32(peerIdx64), uint32(cIdx64))
		if err != nil {
			return err
		}
		if qc.CloseData.Closed {
			return fmt.Errorf("channel %d, %d is closed.", peerIdx64, peerIdx64)
		}
		err = LNode.SyncWatch(qc)
		if err != nil {
			return err
		}
		return nil
	}
	// show contents of the local stash
	s, err := LNode.ShowJusticeDB()
	if err != nil {
		return err
	}
	fmt.Printf(s)
	// show contents of watchtower db
	s, err = LNode.Tower.Status()
	if err != nil {
		return err
	}
	fmt.Printf(s)
	return nil
}

// CloseChannel is a cooperative closing of a channel to a specified address.
func CloseChannel(args []string) error {
	if LNode.RemoteCon == nil || LNode.RemoteCon.RemotePub == nil {
		return fmt.Errorf("Not connected to anyone\n")
	}
	// need args, fail
	if len(args) < 2 {
		return fmt.Errorf("need args: cclose peerIdx chanIdx")
	}

	peerIdx64, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return err
	}
	cIdx64, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return err
	}
	peerIdx := uint32(peerIdx64)
	cIdx := uint32(cIdx64)

	// find the peer index of who we're connected to
	currentPeerIdx, err := LNode.GetPeerIdx(LNode.RemoteCon.RemotePub)
	if err != nil {
		return err
	}
	if peerIdx != currentPeerIdx {
		return fmt.Errorf("Want to close with peer %d but connected to %d	",
			peerIdx, currentPeerIdx)
	}

	qc, err := LNode.GetQchanByIdx(peerIdx, cIdx)
	if err != nil {
		return err
	}

	if qc.CloseData.Closed {
		return fmt.Errorf("can't close (%d,%d): already closed",
			qc.KeyGen.Step[3]&0x7fffffff, qc.KeyGen.Step[4]&0x7fffffff)
	}

	tx, err := qc.SimpleCloseTx()
	if err != nil {
		return err
	}

	sig, err := LNode.SignSimpleClose(qc, tx)
	if err != nil {
		return err
	}

	// Save something to db... TODO
	// Should save something, just so the UI marks it as closed, and
	// we don't accept payments on this channel anymore.

	opArr := lnutil.OutPointToBytes(qc.Op)
	// close request is just the op, sig
	msg := []byte{qln.MSGID_CLOSEREQ}
	msg = append(msg, opArr[:]...)
	msg = append(msg, sig...)

	_, err = LNode.RemoteCon.Write(msg)
	return nil
}

// BreakChannel closes the channel without the other party's involvement.
// The user causing the channel Break has to wait for the OP_CSV timeout
// before funds can be recovered.  Break output addresses are already in the
// DB so you can't specify anything other than which channel to break.
func BreakChannel(args []string) error {
	// need args, fail
	if len(args) < 2 {
		return fmt.Errorf("need args: break peerIdx chanIdx")
	}

	peerIdx, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return err
	}
	cIdx, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return err
	}

	qc, err := LNode.GetQchanByIdx(uint32(peerIdx), uint32(cIdx))
	if err != nil {
		return err
	}

	if qc.CloseData.Closed {
		return fmt.Errorf("Can't break (%d,%d), already closed\n",
			qc.KeyGen.Step[3]&0x7fffffff, qc.KeyGen.Step[4]&0x7fffffff)
	}

	fmt.Printf("breaking (%d,%d)\n",
		qc.KeyGen.Step[3]&0x7fffffff, qc.KeyGen.Step[4]&0x7fffffff)
	z, err := qc.ElkSnd.AtIndex(0)
	if err != nil {
		return err
	}
	fmt.Printf("elk send 0: %s\n", z.String())
	z, err = qc.ElkRcv.AtIndex(0)
	if err != nil {
		return err
	}
	fmt.Printf("elk recv 0: %s\n", z.String())
	// set delta to 0...
	qc.State.Delta = 0
	tx, err := LNode.SignBreakTx(qc)
	if err != nil {
		return err
	}

	// broadcast
	return LNode.BaseWallet.PushTx(tx)
}
