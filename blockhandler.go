package main

import (
	"fmt"

	"github.com/btcsuite/btcd/wire"
)

// get blocks that come in on the channel and hand them off to the processBlock
// function.  makes a new thread for each block coming in.
func BlockHandler(BlockChannel chan wire.MsgBlock) {
	fmt.Printf("started blockHandler")
	for {

		b := <-BlockChannel
		go processBlock(b)
	}
}

// process a full block of transactions
// here's where to write to a database or external file or something
func processBlock(b wire.MsgBlock) {
	fmt.Printf("Got block %s\n", b.Header.BlockHash().String())
	fmt.Printf("has %d txs\n", len(b.Transactions))
	for i, tx := range b.Transactions {
		s := fmt.Sprintf("\ttx %d %s\n", i, tx.TxHash().String())
		for j, input := range tx.TxIn {
			s += fmt.Sprintf("\t\tin %d %s\n", j, input.PreviousOutPoint.String())
		}
		for k, output := range tx.TxOut {
			s += fmt.Sprintf("\t\tout %d %x %d\n", k, output.PkScript, output.Value)
		}
		fmt.Printf(s)
	}
}
