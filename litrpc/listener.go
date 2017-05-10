package litrpc

import (
	"encoding/json"
	"fmt"
	"log"
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
	users     map[*websocket.Conn]bool
}

func serveWS(rpcl *LitRPC) func(*websocket.Conn) {
	return func(ws *websocket.Conn) {
		rpcl.users[ws] = true
		jsonrpc.ServeConn(ws) // this is a blocking call, it returns upon user disconnect
		delete(rpcl.users, ws)
	}
}

type serverResponse struct {
	Id     *json.RawMessage `json:"id"`
	Result interface{}      `json:"result"`
	Error  interface{}      `json:"error"`
}

func chatHandler(rpcl *LitRPC) {
	for {
		select {
		case msg := <-rpcl.Node.UserChat:
			log.Println(msg)
			for ws := range rpcl.users {
				resp := serverResponse{Id: nil}
				resp.Result = msg
				json.NewEncoder(ws).Encode(resp)
			}
		}
	}
}

func RPCListen(rpcl *LitRPC, port uint16) {
	rpcl.users = make(map[*websocket.Conn]bool)

	listenString := fmt.Sprintf("127.0.0.1:%d", port)

	rpc.Register(rpcl)

	http.Handle("/ws", websocket.Handler(serveWS(rpcl)))
	go chatHandler(rpcl)
	go http.ListenAndServe(listenString, nil)
}
