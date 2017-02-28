package litrpc

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/mit-dci/lit/qln"
)

/*
Remote Procedure Calls
RPCs are how people tell the lit node what to do.
It ends up being the root of ~everything in the executable.


*/

// A LitRPC is the user I/O interface; it owns and initialized a SPVCon and LitNode
// and listens and responds on RPC
type LitRPC struct {
	Node      *qln.LitNode
	OffButton chan bool
}

func RpcListen(node *qln.LitNode, port uint16) {
	rpcl := new(LitRPC)
	rpcl.Node = node
	rpcl.OffButton = make(chan bool, 1)

	server := rpc.NewServer()
	server.Register(rpcl)
	server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	portString := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", portString)
	if err != nil {
		fmt.Printf(err.Error())
		return
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("listener error: " + err.Error())
			} else {
				log.Printf("new connection from %s\n", conn.RemoteAddr().String())
				go server.ServeCodec(jsonrpc.NewServerCodec(conn))
			}
		}
	}()

	// ugly; add real synchronization here
	<-rpcl.OffButton
	fmt.Printf("Got stop request\n")
	time.Sleep(time.Second)
	return
}
