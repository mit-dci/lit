package litrpc

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/mit-dci/lit/qln"
	"github.com/mit-dci/lit/uspv"
)

/*
Remote Procedure Calls
RPCs are how people tell the lit node what to do.
It ends up being the root of ~everything in the executable.


*/

// A LitRPC is the user I/O interface; it owns and initialized a SPVCon and LitNode
// and listens and responds on RPC
type LitRPC struct {
	SCon uspv.SPVCon
	Node qln.LitNode
}

func RpcListen(scon uspv.SPVCon, node qln.LitNode, port uint16) {
	rpcl := new(LitRPC)
	rpcl.SCon = scon
	rpcl.Node = node

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
}
