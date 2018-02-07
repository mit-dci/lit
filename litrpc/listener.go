package litrpc

import (
	"crypto/tls"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"

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

func RPCListen(rpcl *LitRPC, port uint16) {

	rpc.Register(rpcl)

	cert, err := tls.LoadX509KeyPair("certs/server.pem", "certs/server.key")
	if err != nil {
		log.Fatalf("Failed to load keys: %s", err)
	}
	conf := tls.Config{Certificates: []tls.Certificate{cert}}
	listenString := "localhost:" + strconv.Itoa(int(port))
	go initListener(listenString, conf)
}

func initListener(listenString string, conf tls.Config) {
	listener, err := tls.Listen("tcp", listenString, &conf) // net.Listener
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Listening for connections..")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %s", err)
			return
		}
		log.Printf("accepted connection from client %s", conn.RemoteAddr())
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 512)
	for {
		log.Print("server: conn: waiting")
		_, err := conn.Read(buf)
		if err != nil {
			log.Println("read error: %s", err)
			break
		}
		jsonrpc.ServeConn(conn)
	}
	log.Println("Client Disconnected")
	return
}
