package qln

/*
functions relating to the "justice transaction" (aka penalty transaction)


because we're using the sipa/schnorr delinearization, we don't need to vary the PKH
anymore.  We can hand over 1 point per commit & figure everything out from that.
*/

// BuildWatchTxidSig builds the partial txid and signature pair which can
// be exported to the watchtower.
func (q *Qchan) BuildWatchTxidSig(prevAmt int64) {

	//	q.BuildStateTx()

	return
}
