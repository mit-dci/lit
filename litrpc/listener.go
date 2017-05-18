package litrpc

import (
	"fmt"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"

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
	jsonrpc.ServeConn(ws) // this is a blocking call, it returns upon user disconnect
}

func RPCListen(rpcl *LitRPC, port uint16) {
	listenString := fmt.Sprintf("127.0.0.1:%d", port)

	rpc.Register(rpcl)

	http.Handle("/ws", websocket.Handler(serveWS))
	go http.ListenAndServe(listenString, nil)
}
