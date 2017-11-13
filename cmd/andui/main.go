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

	"github.com/andlabs/ui"
)

const (
	litHomeDirName  = ".lit"
	historyFilename = "lit-af.history"
)

type litUiClient struct {
	remote     string
	port       uint16
	rpccon     *rpc.Client
	litHomeDir string
}

func setConfig(lc *litUiClient) {
	hostptr := flag.String("node", "127.0.0.1", "host to connect to")
	portptr := flag.Int("p", 8001, "port to connect to")
	dirptr := flag.String("dir", filepath.Join(os.Getenv("HOME"), litHomeDirName),
		"directory to save settings")

	flag.Parse()

	lc.remote = *hostptr
	lc.port = uint16(*portptr)
	lc.litHomeDir = *dirptr
}

var lu *litUiClient

func main() {

	lu = new(litUiClient)
	setConfig(lu)

	origin := "http://127.0.0.1/"
	urlString := fmt.Sprintf("ws://%s:%d/ws", lu.remote, lu.port)
	wsConn, err := websocket.Dial(urlString, "", origin)
	if err != nil {
		log.Fatal(err)
	}
	defer wsConn.Close()

	lu.rpccon = jsonrpc.NewClient(wsConn)

	firstAdr, err := lu.Address()
	if err != nil {
		panic(err)
	}

	firstBal, err := lu.GetBalance()
	if err != nil {
		panic(err)
	}

	err = ui.Main(func() {

		adrBar := ui.NewEntry()
		adrBar.SetReadOnly(true)

		adrBar.SetText(firstAdr)

		adrButton := ui.NewButton("new address")
		adrButton.OnClicked(func(*ui.Button) {
			adr, err := lu.NewAddress()
			if err != nil {
				panic(err)
			}

			adrBar.SetText(adr)
		})

		sendLabel := ui.NewLabel("send to:")

		sendAmtLabel := ui.NewLabel("amt:")
		sendAdrBox := ui.NewEntry()
		sendAmtBox := ui.NewSpinbox(0, 100000000000)
		statusTextBox := ui.NewLabel("")

		balGroup := ui.NewGroup("baLanCes")
		balGroup.SetTitle("Balances")
		balBox := ui.NewVerticalBox()
		balGroup.SetChild(balBox)

		for _, bal := range firstBal.Balances {
			btxt := fmt.Sprintf("Coin: %d Height %d Bal:%d",
				bal.CoinType, bal.SyncHeight, bal.TxoTotal)
			balBox.Append(ui.NewLabel(btxt), false)
		}

		sendBtn := ui.NewButton("send")

		sendHbox := ui.NewHorizontalBox()
		sendHbox.Append(sendLabel, false)
		sendHbox.Append(sendAdrBox, true)
		sendHbox.Append(sendAmtLabel, false)
		sendHbox.Append(sendAmtBox, false)
		sendHbox.Append(sendBtn, false)

		sendBtn.OnClicked(func(*ui.Button) {
			amtString := fmt.Sprintf("%d", sendAmtBox.Value())
			reponse, err := lu.Send(sendAdrBox.Text(), amtString)
			if err != nil {
				// you need to make a window for MsgBoxError, but
				// it doesn't seem to DO anything.  If the window you give
				// is nil, however, you get a nil pointer dereference crash
				dummyWindow := ui.NewWindow("", 100, 100, false)
				ui.MsgBoxError(dummyWindow, "Send error", err.Error())
			} else {
				statusTextBox.SetText(reponse)
			}
		})

		sendAdrBox.SetText("")
		//		recvHbox.Append(adrBar, true)
		//		recvHbox.Append(button, false)

		recvHbox := ui.NewHorizontalBox()
		recvHbox.Append(adrBar, true)
		recvHbox.Append(ui.NewHorizontalSeparator(), false)
		recvHbox.Append(adrButton, false)

		box := ui.NewVerticalBox()
		box.Append(ui.NewLabel("My address:"), false)
		box.Append(recvHbox, false)
		box.Append(sendHbox, false)
		box.Append(statusTextBox, false)
		box.Append(balGroup, false)

		//		vtab := ui.NewTab()
		//		cbx := ui.NewCombobox()
		//		cbx.Append("a")
		//		cbx.Append("b")

		//		grp := ui.NewGroup("grp")
		//		grp.SetTitle("is group")
		//		grp.SetChild(cbx)

		//		box.Append(vtab, false)

		window := ui.NewWindow("lit ui", 650, 300, false)
		window.SetChild(box)

		window.OnClosing(func(*ui.Window) bool {
			ui.Quit()
			return true
		})
		window.Show()
	})

	if err != nil {
		panic(err)
	}
}

/*
Lit-UI

Try using andlabs/ui for this

Try to get most of the lit-af functionality working here.

*/

/*

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
*/
