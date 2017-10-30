package main

import (
	"fmt"

	"github.com/aarzilli/nucular"
	"github.com/aarzilli/nucular/label"
	nstyle "github.com/aarzilli/nucular/style"
)

var tx nucular.TextEditor

func buttonFunc(w *nucular.Window) {

	w.Row(50).Static(100, 100)
	if w.Button(label.T("Addr"), false) {
		fmt.Printf("Addr button\n")
	}
	if w.Button(label.T("Send"), false) {
		fmt.Printf("Send button\n")
	}
	// note that it doesn't work (can't change text field) when the tx.Flags
	// and other tx.stuff is written to here, in the function.  Why? Who knows!
	// no docs anywhere so... yeah yoloui.

	w.Row(25).Dynamic(1)
	w.Label("Address:", "LC")

	_ = tx.Edit(w)
	//	od.FieldEditor.Edit(w)
}

func main() {

	tx.Flags = nucular.EditSimple
	tx.Active = true
	tx.Initialized = true
	tx.Maxlen = 16
	tx.Text([]rune("hi"))

	wnd := nucular.NewMasterWindow(0, "litnuc test", buttonFunc)
	wnd.SetStyle(nstyle.FromTheme(nstyle.DarkTheme, 1))
	wnd.Main()

}
