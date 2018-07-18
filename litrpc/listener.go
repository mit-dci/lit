package litrpc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"

	"golang.org/x/net/websocket"

	"github.com/mit-dci/lit/qln"
	"github.com/mit-dci/lit/webui"
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

func WebUIHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		r.URL.Path = "/index.html"
	}

	bytes, err := webui.Asset(r.URL.Path[1:])
	if err != nil {
		log.Printf("Error serving request [%s]: [%s]\n", r.URL.Path, err.Error())
		w.WriteHeader(404)
		return
	}

	if strings.HasSuffix(r.URL.Path, ".html") {
		w.Header().Add("Content-Type", "text/html")
	} else if strings.HasSuffix(r.URL.Path, ".ico") {
		w.Header().Add("Content-Type", "image/x-icon")
	} else if strings.HasSuffix(r.URL.Path, ".js") {
		w.Header().Add("Content-Type", "application/javascript")
	} else if strings.HasSuffix(r.URL.Path, ".json") {
		w.Header().Add("Content-Type", "application/json")
	} else if strings.HasSuffix(r.URL.Path, ".css") {
		w.Header().Add("Content-Type", "text/css")
	}

	w.WriteHeader(200)
	w.Write(bytes)
}

func RPCListen(rpcl *LitRPC, host string, port uint16) {

	rpc.Register(rpcl)

	listenString := fmt.Sprintf("%s:%d", host, port)

	http.Handle("/ws", websocket.Handler(serveWS))
	http.HandleFunc("/static/", WebUIHandler)
	http.HandleFunc("/", WebUIHandler)

	log.Fatal(http.ListenAndServe(listenString, nil))
}
