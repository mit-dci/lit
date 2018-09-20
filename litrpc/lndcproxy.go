package litrpc

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/logging"
	"golang.org/x/net/websocket"
)

// LndcRpcWebsocketProxy is a regular websocket server that translates the
// received requests into a call on the new LNDC based remote control transport
type LndcRpcWebsocketProxy struct {
	lndcRpcClient *LndcRpcClient
}

// NewLocalLndcRpcWebsocketProxy is an overload to NewLndcRpcWebsocketProxyWithLndc
// connecting to the local lit using a derived key from the locally stored
// privkey.hex using the default home dir and port
func NewLocalLndcRpcWebsocketProxy() (*LndcRpcWebsocketProxy, error) {
	client, err := NewLocalLndcRpcClient()
	if err != nil {
		return nil, err
	}
	return NewLndcRpcWebsocketProxyWithLndc(client), nil
}

// NewLocalLndcRpcWebsocketProxy is an overload to NewLndcRpcWebsocketProxyWithLndc
// connecting to the local lit using a derived key from the locally stored
// privkey.hex using the default home dir and the given port
func NewLocalLndcRpcWebsocketProxyWithPort(port uint32) (*LndcRpcWebsocketProxy, error) {
	client, err := NewLocalLndcRpcClientWithPort(port)
	if err != nil {
		return nil, err
	}
	return NewLndcRpcWebsocketProxyWithLndc(client), nil
}

// NewLocalLndcRpcWebsocketProxy is an overload to NewLndcRpcWebsocketProxyWithLndc
// connecting to the local lit using a derived key from the locally stored
// privkey.hex using the given homedir and the given port
func NewLocalLndcRpcWebsocketProxyWithHomeDirAndPort(litHomeDir string, port uint32) (*LndcRpcWebsocketProxy, error) {
	client, err := NewLocalLndcRpcClientWithHomeDirAndPort(litHomeDir, port)
	if err != nil {
		return nil, err
	}
	return NewLndcRpcWebsocketProxyWithLndc(client), nil
}

// NewLndcRpcWebsocketProxy is an overload to NewLndcRpcWebsocketProxyWithLndc
// connecting to the given lit node specified by litAdr, identifying with the
// given key.
func NewLndcRpcWebsocketProxy(litAdr string, key *koblitz.PrivateKey) (*LndcRpcWebsocketProxy, error) {
	client, err := NewLndcRpcClient(litAdr, key)
	if err != nil {
		return nil, err
	}
	return NewLndcRpcWebsocketProxyWithLndc(client), nil
}

// NewLndcRpcWebsocketProxyWithLndc creates a new proxy that listens on the
// websocket port and translates requests into remote control messages over
// lndc transport.
func NewLndcRpcWebsocketProxyWithLndc(lndcRpcClient *LndcRpcClient) *LndcRpcWebsocketProxy {
	proxy := new(LndcRpcWebsocketProxy)
	proxy.lndcRpcClient = lndcRpcClient
	return proxy
}

// Listen starts listening on the given host and port for websocket requests
// This function blocks unless an error occurs, so you should call it as a
// goroutine
func (p *LndcRpcWebsocketProxy) Listen(host string, port uint16) {

	listenString := fmt.Sprintf("%s:%d", host, port)
	http.HandleFunc("/ws",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{Handler: websocket.Handler(p.proxyServeWS)}
			s.ServeHTTP(w, req)
		})
	/*http.HandleFunc("/static/", WebUIHandler)
	http.HandleFunc("/", WebUIHandler)
	http.HandleFunc("/oneoff", serveOneoffs)*/

	logging.Infof("Listening regular Websocket RPC on %s...", listenString)
	err := http.ListenAndServe(listenString, nil)
	logging.Errorf("Error on websocket server: %s", err.Error())
}

// proxyServeWS handles incoming websocket requests
func (p *LndcRpcWebsocketProxy) proxyServeWS(ws *websocket.Conn) {

	// Close the connection when we're done
	defer ws.Close()
	for {

		// Receive a websocket frame
		var data interface{}
		err := websocket.JSON.Receive(ws, &data)
		if err != nil {
			logging.Warnf("Error receiving websocket frame: %s\n", err.Error())
			break
		}
		if data == nil {
			logging.Warnf("Received nil websocket frame")
			break
		}
		var reply interface{}
		// Parse the incoming request as generic JSON
		msg := data.(map[string]interface{})
		method := msg["method"]
		var args interface{}

		// Default to a NoArgs in case the params array is nil
		args = new(NoArgs)

		if msg["params"] != nil {

			// Parse the params into the args object
			argsArray, ok := msg["params"].([]interface{})
			if ok && len(argsArray) > 0 {
				// In JsonRpc 2.0, the params are always an array
				// but lit always only expects a single Args object
				// so we take index 0 from the array.
				args = msg["params"].([]interface{})[0]
			}
		}

		// store the id from the jsonrpc request
		id := msg["id"]

		// Use the LNDC RPC client to execute the call
		err = p.lndcRpcClient.Call(method.(string), args, &reply)
		var jsonResponse []byte
		if err != nil {
			// In case of an error send back a JSON RPC formatted error
			// response
			jsonResponse, _ = json.Marshal(errorResponse(id, err))
		} else {
			// In case of success, send back a JSON RPC formatted
			// response
			jsonResponse, _ = json.Marshal(successResponse(id, reply))
		}

		// Write the response to the stream, and continue with the next websocket
		// frame
		ws.Write(jsonResponse)
	}

}

// baseResponse adds the basic elements for each response type to the JSON Response
func baseResponse(requestId interface{}) map[string]interface{} {
	response := make(map[string]interface{})
	response["jsonrpc"] = "2.0"
	response["id"] = requestId
	return response
}

// errorResponse translates an error into a properly formatted JSON RPC response
func errorResponse(requestId interface{}, err error) map[string]interface{} {
	errorObj := make(map[string]interface{})
	errorObj["code"] = -32000
	errorObj["message"] = "Internal Server Error"
	errorObj["data"] = err.Error()

	response := baseResponse(requestId)
	response["error"] = errorObj

	return response
}

// successResponse formats a reply object into a properly formatted JSON RPC response
func successResponse(requestId interface{}, data interface{}) map[string]interface{} {
	response := baseResponse(requestId)
	response["result"] = data
	return response
}
