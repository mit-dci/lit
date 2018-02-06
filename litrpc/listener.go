package litrpc

import (
	"bytes"
	//"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/mit-dci/lit/qln"
	"golang.org/x/net/websocket"
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
	body, err := ioutil.ReadAll(ws.Request().Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		return
	}

	log.Printf(string(body))
	ws.Request().Body = ioutil.NopCloser(bytes.NewBuffer(body))

	jsonrpc.ServeConn(ws)
}

func RPCListen(rpcl *LitRPC, port uint16) {

	rpc.Register(rpcl)

	listenString := fmt.Sprintf("localhost:%d", port)

	// cert, err := tls.LoadX509KeyPair("certs/server.pem", "certs/server.key")
	// if err != nil {
	// 	log.Fatalf("Failed to load keys: %s", err)
	// }
	// conf := tls.Config{Certificates: []tls.Certificate{cert}}
	// listenString = fmt.Sprintf("localhost:%d", port)
	// listener, err := tls.Listen("tcp", listenString, &conf)
	// if err != nil {
	// 	log.Fatalf("server error: %s", err)
	// }
	// log.Print("Listening for connections..")
	// conn, err := listener.Accept()
	// if err != nil {
	// 	log.Printf("accept: %s", err)
	// 	return
	// }
	// log.Printf("accepted connection from %s", conn.RemoteAddr())
	// defer conn.Close()

	http.Handle("/ws", websocket.Handler(serveWS))
	log.Fatal(http.ListenAndServeTLS(listenString, "certs/server.pem", "certs/server.key", nil))
}
