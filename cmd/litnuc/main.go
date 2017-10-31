package main

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"path/filepath"

	"golang.org/x/net/websocket"

	"github.com/aarzilli/nucular"
	"github.com/aarzilli/nucular/label"
	nstyle "github.com/aarzilli/nucular/style"
)

/*
Lit-nuc

Lit nucular UI RPC client

Not too sure about "nucular" but it does seem to compile and work, which
is more than I could get from any other UI package I tried.

Try to get most of the lit-af functionality working here.

*/

const (
	litHomeDirName  = ".lit"
	historyFilename = "lit-af.history"
)

type litNucClient struct {
	remote     string
	port       uint16
	rpccon     *rpc.Client
	litHomeDir string
}

func setConfig(lc *litNucClient) {
	hostptr := flag.String("node", "127.0.0.1", "host to connect to")
	portptr := flag.Int("p", 8001, "port to connect to")
	dirptr := flag.String("dir", filepath.Join(os.Getenv("HOME"), litHomeDirName),
		"directory to save settings")

	flag.Parse()

	lc.remote = *hostptr
	lc.port = uint16(*portptr)
	lc.litHomeDir = *dirptr
}

// globals!

var tx nucular.TextEditor
var adrBar nucular.TextEditor
var adr string
var lu *litNucClient

func buttonFunc(w *nucular.Window) {

	w.Row(50).Static(100, 100)
	if w.Button(label.T("New Address"), false) {
		var err error
		adr, err = lu.Address()
		if err != nil {
			adr = err.Error()
		} else {
			fmt.Printf("Got new address %s\n", adr)
		}
		adrBar.Buffer = []rune(adr)
	}
	if w.Button(label.T("Send"), false) {
		fmt.Printf("send to addr %s\n", string(tx.Buffer))

	}
	w.Row(25).Dynamic(1)

	adrBar.Edit(w)

	//	w.Label(adr, "LC")
	// note that it doesn't work (can't change text field) when the tx.Flags
	// and other tx.stuff is written to here, in the function.  Why? Who knows!
	// no docs anywhere so... yeah yoloui.

	w.Row(25).Dynamic(1)
	w.Label("Address:", "LC")

	_ = tx.Edit(w)

}

func main() {
	lu = new(litNucClient)
	setConfig(lu)

	origin := "http://127.0.0.1/"
	urlString := fmt.Sprintf("ws://%s:%d/ws", lu.remote, lu.port)
	wsConn, err := websocket.Dial(urlString, "", origin)
	if err != nil {
		log.Fatal(err)
	}
	defer wsConn.Close()

	lu.rpccon = jsonrpc.NewClient(wsConn)

	tx.Flags = nucular.EditSelectable | nucular.EditClipboard
	tx.Maxlen = 64
	//	tx.Buffer = []rune("put address here")

	adrBar.Flags = nucular.EditSelectable | nucular.EditClipboard |
		nucular.EditReadOnly | nucular.EditNoCursor
	adr, err = lu.Address()
	if err != nil {
		adr = err.Error()
	}
	adrBar.Buffer = []rune(adr)

	wnd := nucular.NewMasterWindow(0, "litnuc test", buttonFunc)
	wnd.SetStyle(nstyle.FromTheme(nstyle.DarkTheme, 1))
	wnd.Main()

}
