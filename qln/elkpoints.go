package qln

import (
	"fmt"
	log "github.com/sirupsen/logrus"

	"github.com/mit-dci/lit/btcutil/chaincfg/chainhash"
	"github.com/mit-dci/lit/lnutil"
)

/*
functions dealing with elkrem points and the elkrem structure
*/

// CurElkPointForThem makes the current state elkrem point to send out
func (q *Qchan) N2ElkPointForThem() (p [33]byte, err error) {
	// generate revocable elkrem point
	return q.ElkPoint(false, q.State.StateIdx+2)
}

// ElkPoint generates an elkrem Point.  "My" elkrem point is the point
// I receive from the counter party, and can create after the state has
// been revoked.  "Their" elkrem point (mine=false) is generated from my elkrem
// sender at any index.
// Elkrem points pubkeys where the scalar is a hash coming from the elkrem tree.
// With delinearized aggregation, only one point is needed.  I'm pretty sure.
func (q *Qchan) ElkPoint(mine bool, idx uint64) (p [33]byte, err error) {
	// sanity check
	if q == nil || q.ElkSnd == nil { // no sender
		err = fmt.Errorf("can't access elkrem sender")
		return
	}
	if mine && q.ElkRcv == nil { // no receiver
		err = fmt.Errorf("can't access elkrem receiver")
		return
	}

	elk := new(chainhash.Hash)

	if mine { // make mine based on receiver
		elk, err = q.ElkRcv.AtIndex(idx)
	} else { // make theirs based on sender
		elk, err = q.ElkSnd.AtIndex(idx)
	}
	// elkrem problem, error out here
	if err != nil {
		return
	}
	p = lnutil.ElkPointFromHash(elk)

	return
}

func (q *Qchan) AdvanceElkrem(elk *chainhash.Hash, n2Elk [33]byte) error {
	// check that the revocation works
	err := q.IngestElkrem(elk)
	if err != nil {
		return err
	}
	// if ingest was successful, update the stored elk points
	// the already stored "next" point becomes the current point
	q.State.ElkPoint = q.State.NextElkPoint
	// n+2 drops to n+1
	q.State.NextElkPoint = q.State.N2ElkPoint
	// and the newly received point is n+2
	q.State.N2ElkPoint = n2Elk
	return nil
}

// IngestElkrem takes in an elkrem hash, performing 2 checks:
// that it produces the proper elk point, and that it fits into the elkrem tree.
// If either of these checks fail, and definitely the second one
// fails, I'm pretty sure the channel is not recoverable and needs to be closed.
func (q *Qchan) IngestElkrem(elk *chainhash.Hash) error {
	if elk == nil {
		return fmt.Errorf("IngestElkrem: nil hash")
	}

	// first verify the elkrem insertion (this only performs checks 1/2 the time, so
	// 1/2 the time it'll work even if the elkrem is invalid, oh well)
	err := q.ElkRcv.AddNext(elk)
	if err != nil {
		return err
	}
	log.Debugf("ingested hash, receiver now has up to %d\n", q.ElkRcv.UpTo())

	// if this is state 0, then we have elkrem 0 and we can stop here.
	// there's nothing to revoke.
	if q.State.StateIdx == 0 {
		return nil
	}

	// next verify if the elkrem produces the previous elk point.
	// We don't actually use the private key operation here, because we can
	// do the same operation on our pubkey that they did, and we have faith
	// in the mysterious power of abelian group homomorphisms that the private
	// key modification will also work.

	// Make point from received elk hash
	point := lnutil.ElkPointFromHash(elk)

	// see if it matches previous elk point
	if point != q.State.ElkPoint {
		log.Debugf("elk1: %x\nelk2: %x\nelk3: %x\nngst: %x\n",
			q.State.ElkPoint, q.State.NextElkPoint, q.State.N2ElkPoint, point)
		// didn't match, the whole channel is borked.
		return fmt.Errorf("hash %x (index %d) fits tree but creates wrong elkpoint!",
			elk[:8], q.State.StateIdx)
	}

	return nil
}
