package litrpc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/mit-dci/lit/btcutil/btcec"
	"golang.org/x/net/websocket"
)

type LndcRpcWebsocketProxy struct {
	lndcRpcClient *LndcRpcClient
}

func NewLocalLndcRpcWebsocketProxy() (*LndcRpcWebsocketProxy, error) {
	client, err := NewLocalLndcRpcClient()
	if err != nil {
		return nil, err
	}
	return newLndcRpcWebsocketProxyWithLndc(client), nil
}

func NewLocalLndcRpcWebsocketProxyWithPort(port uint32) (*LndcRpcWebsocketProxy, error) {
	client, err := NewLocalLndcRpcClientWithPort(port)
	if err != nil {
		return nil, err
	}
	return newLndcRpcWebsocketProxyWithLndc(client), nil
}

func NewLocalLndcRpcWebsocketProxyWithHomeDirAndPort(litHomeDir string, port uint32) (*LndcRpcWebsocketProxy, error) {
	client, err := NewLocalLndcRpcClientWithHomeDirAndPort(litHomeDir, port)
	if err != nil {
		return nil, err
	}
	return newLndcRpcWebsocketProxyWithLndc(client), nil
}

func NewLndcRpcWebsocketProxy(litAdr string, key *btcec.PrivateKey) (*LndcRpcWebsocketProxy, error) {
	client, err := NewLndcRpcClient(litAdr, key)
	if err != nil {
		return nil, err
	}
	return newLndcRpcWebsocketProxyWithLndc(client), nil
}

func newLndcRpcWebsocketProxyWithLndc(lndcRpcClient *LndcRpcClient) *LndcRpcWebsocketProxy {
	proxy := new(LndcRpcWebsocketProxy)
	proxy.lndcRpcClient = lndcRpcClient
	return proxy
}

func (p *LndcRpcWebsocketProxy) Listen(host string, port uint16) {

	listenString := fmt.Sprintf("%s:%d", host, port)
	http.Handle("/ws", websocket.Handler(p.proxyServeWS))
	/*http.HandleFunc("/static/", WebUIHandler)
	http.HandleFunc("/", WebUIHandler)
	http.HandleFunc("/oneoff", serveOneoffs)*/
	log.Fatal(http.ListenAndServe(listenString, nil))
}

func (p *LndcRpcWebsocketProxy) proxyServeWS(ws *websocket.Conn) {

	defer ws.Close()
	for {
		var data interface{}
		err := websocket.JSON.Receive(ws, &data)
		if err != nil {
			log.Printf("Error receiving websocket frame: %s\n", err.Error())
			break
		}
		if data == nil {
			log.Println("Received nil websocket frame")
			break
		}
		var reply interface{}
		msg := data.(map[string]interface{})
		method := msg["method"]
		var args interface{}
		args = new(NoArgs)

		if msg["params"] != nil {
			argsArray, ok := msg["params"].([]interface{})
			if ok && len(argsArray) > 0 {
				args = msg["params"].([]interface{})[0]
			}
		}
		id := msg["id"]
		err = p.lndcRpcClient.Call(method.(string), args, &reply)
		var jsonResponse []byte
		if err != nil {
			jsonResponse, _ = json.Marshal(errorResponse(id, err))
		} else {
			jsonResponse, _ = json.Marshal(successResponse(id, reply))
		}

		ws.Write(jsonResponse)
	}

}

func baseResponse(requestId interface{}) map[string]interface{} {
	response := make(map[string]interface{})
	response["jsonrpc"] = "2.0"
	response["id"] = requestId
	return response
}

func errorResponse(requestId interface{}, err error) map[string]interface{} {
	errorObj := make(map[string]interface{})
	errorObj["code"] = -32000
	errorObj["message"] = "Internal Server Error"
	errorObj["data"] = err.Error()

	response := baseResponse(requestId)
	response["error"] = errorObj

	return response
}

func successResponse(requestId interface{}, data interface{}) map[string]interface{} {
	response := baseResponse(requestId)
	response["result"] = data
	return response
}
