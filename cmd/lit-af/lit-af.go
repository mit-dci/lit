package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/websocket"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/mit-dci/lit/lnutil"
)

/*
Lit-AF

The Lit Advanced Functionality interface.
This is a text mode interface to lit.  It connects over jsonrpc to the a lit
node and tells that lit node what to do.  The lit node also responds so that
lit-af can tell what's going on.

lit-gtk does most of the same things with a gtk interface, but there will be
some yet-undefined advanced functionality only available in lit-af.

May end up using termbox-go

*/

//// BalReply is the reply when the user asks about their balance.
//// This is a Non-Channel
//type BalReply struct {
//	ChanTotal         int64
//	TxoTotal          int64
//	SpendableNow      int64
//	SpendableNowWitty int64
//}

const (
	litHomeDirName  = ".lit"
	historyFilename = "lit-af.history"
)

type litAfClient struct {
	remote string
	port   uint16
	rpccon *rpc.Client
	//httpcon
	litHomeDir string
}

type Command struct {
	Format           string
	Description      string
	ShortDescription string
}

func setConfig(lc *litAfClient) {
	hostptr := flag.String("node", "127.0.0.1", "host to connect to")
	portptr := flag.Int("p", 8001, "port to connect to")
	dirptr := flag.String("dir", filepath.Join(os.Getenv("HOME"), litHomeDirName), "directory to save settings")

	flag.Parse()

	lc.remote = *hostptr
	lc.port = uint16(*portptr)
	lc.litHomeDir = *dirptr
}

// for now just testing how to connect and get messages back and forth
func main() {
	lc := new(litAfClient)
	setConfig(lc)

	//	dialString := fmt.Sprintf("%s:%d", lc.remote, lc.port)

	/*
		client, err := net.Dial("tcp", dialString)
		if err != nil {
			log.Fatal("dialing:", err)
		}
		defer client.Close()
	*/

	//	dialString := fmt.Sprintf("%s:%d", lc.remote, lc.port)
	origin := "http://127.0.0.1/"
	urlString := fmt.Sprintf("ws://%s:%d/ws", lc.remote, lc.port)
	//	url := "ws://127.0.0.1:8000/ws"
	wsConn, err := websocket.Dial(urlString, "", origin)
	if err != nil {
		log.Fatal(err)
	}
	defer wsConn.Close()

	lc.rpccon = jsonrpc.NewClient(wsConn)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       lnutil.Prompt("lit-af") + lnutil.White("# "),
		HistoryFile:  filepath.Join(lc.litHomeDir, historyFilename),
		AutoComplete: lc.NewAutoCompleter(),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()
	go lc.RequestAsync()
	go lc.AsyncInvoiceReplier()

	// main shell loop
	for {
		// setup reader with max 4K input chars
		msg, err := rl.Readline()
		if err != nil {
			break
		}
		msg = strings.TrimSpace(msg)
		if len(msg) == 0 {
			continue
		}
		rl.SaveHistory(msg)

		cmdslice := strings.Fields(msg)                         // chop input up on whitespace
		fmt.Fprintf(color.Output, "entered command: %s\n", msg) // immediate feedback

		err = lc.Shellparse(cmdslice)
		if err != nil { // only error should be user exit
			log.Fatal(err)
		}
	}

	//	err = c.Call("LitRPC.Bal", nil, &br)
	//	if err != nil {
	//		log.Fatal("rpc call error:", err)
	//	}
	//	fmt.Printf("Sent bal req, response: txototal %d\n", br.TxoTotal)
}

func (lc *litAfClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	c := make(chan error, 1)
	go func() { c <- lc.rpccon.Call(serviceMethod, args, &reply) }()
	select {
	case err := <-c:
		return err
	case <-time.After(time.Second * 10):
		return errors.New("RPC call timed out")
	}
}
