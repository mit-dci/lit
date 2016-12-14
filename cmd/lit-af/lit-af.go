package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc/jsonrpc"
)

/*
Lit-AF

The Lit Advanced Functionality interface.
This is a text mode interface to lit.  It connects over jsonrpc to the a lit
node and tells that lit node what to do.  The lit node also responds so that
lit-af can tell what's going on.

lit-gtk does most of the same things with a gtk interface, but there will be
some yet-undefined advanced functionality only available in lit-af.
*/

// BalReply is the reply when the user asks about their balance.
// This is a Non-Channel
type BalReply struct {
	ChanTotal         int64
	TxoTotal          int64
	SpendableNow      int64
	SpendableNowWitty int64
}

// for now just testing how to connect and get messages back and forth
func main() {

	client, err := net.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer client.Close()

	// Synchronous call
	br := new(BalReply)
	c := jsonrpc.NewClient(client)

	err = c.Call("LitRPC.Bal", nil, &br)
	if err != nil {
		log.Fatal("rpc call error:", err)
	}
	fmt.Printf("Sent bal req, response: txototal %d\n", br.TxoTotal)
}
