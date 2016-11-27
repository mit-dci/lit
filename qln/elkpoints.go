package qln

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/mit-dci/lit/lnutil"
)

/*
functions dealing with elkrem points and the elkrem structure
*/

// MakeTheirCurElkPoint makes the current state elkrem points to send out
func (q *Qchan) MakeTheirCurElkPoints() (r, t [33]byte, err error) {
	// generate revocable elkrem point
	r, err = q.ElkPoint(false, q.State.StateIdx)
	if err != nil {
		return
	}
	// generate timeout elkrem point
	t, err = q.ElkPoint(false, q.State.StateIdx)
	return
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
	// hash with suffix "POINT" just for fun
	*elk = chainhash.DoubleHashH(append(elk[:], []byte("POINT")...))

	// turn the hash into a point
	p = lnutil.PubFromHash(*elk)
	return
}

// IngestElkrem takes in an elkrem hash, performing 2 checks:
// that it produces the proper elk point, and that it fits into the elkrem tree.
// if both of these are the case it updates the channel state, removing the
// revoked point. If either of these checks fail, and definitely the second one
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
	fmt.Printf("ingested hash, receiver now has up to %d\n", q.ElkRcv.UpTo())

	// if this is state 1, then we have elkrem 0 and we can stop here.
	// there's nothing to revoke.
	if q.State.StateIdx == 1 {
		return nil
	}

	// next verify if the elkrem produces the previous elk point.
	// We don't actually use the private key operation here, because we can
	// do the same operation on our pubkey that they did, and we have faith
	// in the mysterious power of abelian group homomorphisms that the private
	// key modification will also work.

	// Make r and t points from received elk hash
	CheckR := lnutil.PubFromHash(chainhash.DoubleHashH(append(elk.CloneBytes(), 0x72))) // r
	CheckT := lnutil.PubFromHash(chainhash.DoubleHashH(append(elk.CloneBytes(), 0x74))) // t

	// see if it matches previous elk point
	if CheckR != q.State.PrevElkPointR || CheckT != q.State.PrevElkPointT {
		// didn't match, the whole channel is borked.
		return fmt.Errorf("hash %x (index %d) fits tree but creates wrong elkpoint!",
			elk[:8], q.State.PrevElkPointR, q.State.PrevElkPointT)
	}

	// it did match, so we can clear the previous HAKD pub
	var empty [33]byte
	q.State.PrevElkPointR = empty
	q.State.PrevElkPointT = empty

	return nil
}
