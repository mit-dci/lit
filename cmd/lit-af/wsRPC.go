package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/rpc"
	"sync"

	"golang.org/x/net/websocket"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/lnutil"
)

type wsCodec struct {
	dec     *json.Decoder // for reading JSON values
	enc     *json.Encoder // for writing JSON values
	c       *websocket.Conn
	req     clientRequest
	resp    clientResponse
	mutex   sync.Mutex      // protects pending
	pending map[uint]string // map request id to method name
}

type clientRequest struct {
	Method string         `json:"method"`
	Params [1]interface{} `json:"params"`
	Id     uint           `json:"id"`
}

func (c *wsCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	c.mutex.Lock()
	c.pending[uint(r.Seq)] = r.ServiceMethod
	c.mutex.Unlock()
	c.req.Method = r.ServiceMethod
	c.req.Params[0] = param
	c.req.Id = uint(r.Seq)
	return c.enc.Encode(&c.req)
}

type clientResponse struct {
	Id     uint             `json:"id"`
	Result *json.RawMessage `json:"result"`
	Error  interface{}      `json:"error"`
}

func (r *clientResponse) reset() {
	r.Id = 0
	r.Result = nil
	r.Error = nil
}

func (c *wsCodec) ReadResponseHeader(r *rpc.Response) error {
	c.resp.reset()
	if err := c.dec.Decode(&c.resp); err != nil {
		return err
	}

	c.mutex.Lock()
	r.ServiceMethod = c.pending[c.resp.Id]
	delete(c.pending, c.resp.Id)
	c.mutex.Unlock()

	r.Error = ""
	r.Seq = uint64(c.resp.Id)
	if c.resp.Error != nil || c.resp.Result == nil {
		x, ok := c.resp.Error.(string)
		if !ok {
			return fmt.Errorf("invalid error %v", c.resp.Error)
		}
		if x == "" {
			x = "unspecified error"
		}
		r.Error = x
	}
	return nil
}

func (c *wsCodec) ReadResponseBody(body interface{}) error {
	// if this was a chat message then the body will be nil
	if body == nil {

		// saftey chack to make sure c.resp.Result is not nil
		if c.resp.Result == nil {
			return errors.New(c.resp.Error.(string))
		}
		chat := new(lnutil.ChatMsg)

		err := json.Unmarshal(*c.resp.Result, chat)
		if err != nil {
			return err
		}
		fmt.Fprintf(color.Output,
			"\nmsg from %s: %s\n", lnutil.White(chat.PeerIdx), lnutil.Green(chat.Text))

		return nil
	}

	return json.Unmarshal(*c.resp.Result, body)
}

func (c *wsCodec) Close() error {
	return c.c.Close()
}

func (lc *litAfClient) CreateRPCCon(wsConn *websocket.Conn) {
	wsCodec := &wsCodec{
		dec:     json.NewDecoder(wsConn),
		enc:     json.NewEncoder(wsConn),
		c:       wsConn,
		pending: make(map[uint]string),
	}
	lc.rpccon = rpc.NewClientWithCodec(wsCodec)
}
