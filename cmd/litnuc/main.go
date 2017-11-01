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
	"github.com/aarzilli/nucular/rect"
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
const popupFlags = nucular.WindowMovable |
	nucular.WindowTitle |
	nucular.WindowDynamic |
	nucular.WindowNoScrollbar

var sendAdrBar, sendAmtBar, myAdrBar nucular.TextEditor
var adr string
var lu *litNucClient

// seriously?? I can't pass the popup mesage content as an arg, only the title?
// Who wrote this "popupOpen()" func??
var popupMsg string

func buttonFunc(w *nucular.Window) {

	w.Row(50).Static(100, 100)
	if w.Button(label.T("New Address"), false) {
		var err error
		adr, err = lu.NewAddress()
		if err != nil {
			adr = err.Error()
		} else {
			fmt.Printf("Got new address %s\n", adr)
		}
		myAdrBar.Buffer = []rune(adr)
	}

	if w.Button(label.T("Send"), false) {
		responseText, err :=
			lu.Send(string(sendAdrBar.Buffer), string(sendAmtBar.Buffer))
		if err != nil {
			log.Printf("send error %s\n", err.Error())
			popupMsg = err.Error()
			w.Master().PopupOpen(
				"Send Error", popupFlags,
				rect.Rect{20, 100, 520, 180}, true, infoPopup)

		} else {
			popupMsg = responseText
			w.Master().PopupOpen(
				"Sent", popupFlags,
				rect.Rect{20, 100, 520, 180}, true, infoPopup)
		}
	}
	w.Row(25).Dynamic(1)

	myAdrBar.Edit(w)

	//	w.Label(adr, "LC")
	// note that it doesn't work (can't change text field) when the tx.Flags
	// and other tx.stuff is written to here, in the function.  Why? Who knows!
	// no docs anywhere so... yeah yoloui.

	w.Row(25).Static(350, 150)
	w.Label("Send to address", "LC")
	w.Label("amount", "LC")
	w.Row(25).Static(350, 150)
	_ = sendAdrBar.Edit(w)
	sendAmtBar.Filter = nucular.FilterDecimal
	_ = sendAmtBar.Edit(w)

}

func infoPopup(w *nucular.Window) {
	w.Row(50).Dynamic(1)
	w.Label(popupMsg, "LC")
	w.Row(25).Dynamic(2)
	if w.Button(label.T("OK"), false) {
		w.Close()
	}
	if w.Button(label.T("Sure"), false) {
		w.Close()
	}
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

	sendAdrBar.Flags = nucular.EditSelectable | nucular.EditClipboard
	sendAdrBar.Maxlen = 64

	sendAmtBar.Flags = nucular.EditSelectable | nucular.EditClipboard
	sendAmtBar.Maxlen = 12

	myAdrBar.Flags = nucular.EditSelectable | nucular.EditClipboard |
		nucular.EditReadOnly | nucular.EditNoCursor
	adr, err = lu.Address()
	if err != nil {
		adr = err.Error()
	}
	myAdrBar.Buffer = []rune(adr)

	wnd := nucular.NewMasterWindow(0, "litnuc test", buttonFunc)
	wnd.SetStyle(nstyle.FromTheme(nstyle.DarkTheme, 1))
	wnd.Main()

}
