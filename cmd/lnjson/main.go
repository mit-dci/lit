package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc/jsonrpc"
)

type Args struct {
	S string
}

func main() {

	client, err := net.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	// Synchronous call
	args := &Args{"helo"}
	var reply string
	c := jsonrpc.NewClient(client)
	err = c.Call("LNRpc.Bal", args, &reply)
	if err != nil {
		log.Fatal("Bal error:", err)
	}
	fmt.Printf("Sent %s, response %s\n", args.S, reply)
}
