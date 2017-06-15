package litrpc

import "fmt"

// ------------------------- listen

type WatchArgs struct {
	ChanIdx, SendToPeer uint32
}

type WatchReply struct {
	Msg string
}

func (r *LitRPC) Watch(args WatchArgs, reply *WatchReply) error {

	// load the whole channel from disk (pretty inefficient)
	qc, err := r.Node.GetQchanByIdx(args.ChanIdx)
	if err != nil {
		return err
	}
	// see if channel is closed and error early
	if qc.CloseData.Closed {
		return fmt.Errorf("Can't push; channel %d closed", args.ChanIdx)
	}

	err = r.Node.SyncWatch(qc, args.SendToPeer)
	if err != nil {
		return err
	}

	reply.Msg = "ok"
	return nil
}
