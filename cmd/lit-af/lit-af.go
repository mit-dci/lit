package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strings"
)

/*
Lit-AF

The Lit Advanced Functionality interface.
This is a text mode interface to lit.  It connects over jsonrpc to the a lit
node and tells that lit node what to do.  The lit node also responds so that
lit-af can tell what's going on.

lit-gtk does most of the same things with a gtk interface, but there will be
some yet-undefined advanced functionality only available in lit-af.

May end up using termbox-go

*/

//// BalReply is the reply when the user asks about their balance.
//// This is a Non-Channel
//type BalReply struct {
//	ChanTotal         int64
//	TxoTotal          int64
//	SpendableNow      int64
//	SpendableNowWitty int64
//}

type litAfClient struct {
	remote string
	port   uint16
	rpccon *rpc.Client
}

func setConfig(lc *litAfClient) {
	hostptr := flag.String("node", "127.0.0.1", "host to connect to")
	portptr := flag.Int("p", 9750, "port to connect to")

	//	regtestptr := flag.Bool("reg", false, "use regtest (not testnet3)")
	//	resyncprt := flag.Bool("resync", false, "force resync from given tip")

	flag.Parse()

	lc.remote = *hostptr
	lc.port = uint16(*portptr)
}

// for now just testing how to connect and get messages back and forth
func main() {

	lc := new(litAfClient)
	setConfig(lc)

	dialString := fmt.Sprintf("%s:%d", lc.remote, lc.port)

	client, err := net.Dial("tcp", dialString)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer client.Close()

	lc.rpccon = jsonrpc.NewClient(client)

	go lc.RequestAsync()

	// main shell loop
	for {
		// setup reader with max 4K input chars
		reader := bufio.NewReaderSize(os.Stdin, 4000)
		fmt.Printf("lit-af# ")              // prompt
		msg, err := reader.ReadString('\n') // input finishes on enter key
		if err != nil {
			log.Fatal(err)
		}

		cmdslice := strings.Fields(msg) // chop input up on whitespace
		if len(cmdslice) < 1 {
			continue // no input, just prompt again
		}
		fmt.Printf("entered command: %s\n", msg) // immediate feedback
		err = lc.Shellparse(cmdslice)
		if err != nil { // only error should be user exit
			log.Fatal(err)
		}
	}

	//	err = c.Call("LitRPC.Bal", nil, &br)
	//	if err != nil {
	//		log.Fatal("rpc call error:", err)
	//	}
	//	fmt.Printf("Sent bal req, response: txototal %d\n", br.TxoTotal)
}
