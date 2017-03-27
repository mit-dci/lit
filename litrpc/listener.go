package litrpc

import (
	"fmt"
	"log"
	"net"
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

//var GlobalServer *rpc.Server

type LitRPC struct {
	Node      *qln.LitNode
	OffButton chan bool
}

/*
type HttpConn struct {
	in  io.Reader
	out io.Writer
}

func (c *HttpConn) Read(p []byte) (n int, err error)  { return c.in.Read(p) }
func (c *HttpConn) Write(d []byte) (n int, err error) { return c.out.Write(d) }
func (c *HttpConn) Close() error                      { return nil }
*/
/*
func JSONRPCoverHTTPHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/lit" { // all RPC calls have path /lit
		serverCodec :=
			jsonrpc.NewServerCodec(&HttpConn{in: r.Body, out: w})
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(200)
		// this is super ugly; find a better way to do this.
		// maybe try a different struct with a ServeHTTP method.
		err := GlobalServer.ServeRequest(serverCodec)
		if err != nil {
			log.Printf("JSON-RPC request serving error: %v", err)
			http.Error(w, "JSON-RPC request error", 500)
			return
		}
	}
} */

func serveWS(ws *websocket.Conn) {
	jsonrpc.ServeConn(ws)
}

func RpcListen(node *qln.LitNode, port uint16) {
	rpcl := new(LitRPC)
	rpcl.Node = node
	rpcl.OffButton = make(chan bool, 1)

	server := rpc.NewServer()
	server.Register(rpcl)

	//	server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	http.Handle("/ws", websocket.Handler(serveWS))

	portString := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", portString)
	if err != nil {
		fmt.Printf(err.Error())
		return
	}
	defer listener.Close()

	//	go http.Serve(listener, http.HandlerFunc(JSONRPCoverHTTPHandler))

	go func() {
		for {
			http.ListenAndServe("localhost:8000", nil)
		}
	}()

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
