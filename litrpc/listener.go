package litrpc

import (
	"fmt"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"golang.org/x/net/websocket"

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

func serveWS(ws *websocket.Conn) {
	jsonrpc.ServeConn(ws)
}

func RpcListen(node *qln.LitNode, port uint16) {
	rpcl := new(LitRPC)
	rpcl.Node = node
	rpcl.OffButton = make(chan bool, 1)

	rpc.Register(rpcl)

	listenString := fmt.Sprintf("0.0.0.0:%d", port)

	http.Handle("/ws", websocket.Handler(serveWS))
	go http.ListenAndServe(listenString, nil)

	// ugly; add real synchronization here
	<-rpcl.OffButton
	fmt.Printf("Got stop request\n")
	time.Sleep(time.Second)
	return
}
