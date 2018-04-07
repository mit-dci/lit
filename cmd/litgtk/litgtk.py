#!/usr/bin/env python

import websocket  # `pip install websocket-client`
import socket
import json

import pygtk
pygtk.require('2.0')
import gtk

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)  # global for socket connection

def getBal():
	rpcCmd = {
		   "method": "LitRPC.Bal",
		   "params": [{
	   }]
	}
	rpcCmd.update({"jsonrpc": "2.0", "id": "93"})
	print(json.dumps(rpcCmd))
	s.sendall(json.dumps(rpcCmd))
	r = json.loads(s.recv(8000000))
	print(r)
	return r["result"]["TotalScore"]

def getAdr():
	rpcCmd = {
		   "method": "LitRPC.Address",
		   "params": [{
	   "NumToMake": 0,
	   }]
	}
	rpcCmd.update({"jsonrpc": "2.0", "id": "94"})
	print(json.dumps(rpcCmd))
	s.sendall(json.dumps(rpcCmd))
	r = json.loads(s.recv(8000000))
	print(r)
	n = len(r["result"]["PreviousAddresses"]) -1
	return r["result"]["PreviousAddresses"][n]   #[len(r["result"]["PreviousAddresses"]-1)]

def prSend(adr, amt):
	rpcCmd = {
		   "method": "LitRPC.Send",
		   "params": [{"DestAddrs": [adr,],"Amts": [amt,]}]
	}
	rpcCmd.update({"jsonrpc": "2.0", "id": "95"})
	print(json.dumps(rpcCmd))
	s.sendall(json.dumps(rpcCmd))
	r = json.loads(s.recv(8000000))
	print(r)
	if r["error"] != None:
		return "send error: " + r["error"]
	return "Sent. TXID: " + r["result"]["Txids"][0]


class lndrpcui:

	def dialog(self, widget, adrWidget, amtWidget):
		txid = prSend(adrWidget.get_text(), amtWidget.get_value_as_int())
		d = gtk.MessageDialog(
			type=gtk.MESSAGE_INFO,  buttons=gtk.BUTTONS_OK,message_format=txid)
		d.run()
		d.destroy()

	def gBal(self, widget, balWidget, rcvadrWidget):
		bal = getBal()
		balWidget.set_text("Balance: " + "{:,}".format(bal) + " (" + str(bal/100000000.0) + "BTC)")
		adr = getAdr()
		rcvadrWidget.set_text(adr)

	def __init__(self):

		window = gtk.Window(gtk.WINDOW_TOPLEVEL)
		self.window = window
		window.connect("destroy", lambda w: gtk.main_quit())
		window.set_title("lit-gtk")

		main_vbox = gtk.VBox(False, 5)
		main_vbox.set_border_width(10)
		window.add(main_vbox)

		rcvFrame = gtk.Frame("Receive Address")
		main_vbox.pack_start(rcvFrame, True, False, 0)
#~ recvHbox
		rcvhbox = gtk.HBox(False, 0)
		rcvhbox.set_border_width(5)
		rcvFrame.add(rcvhbox)
		rcvLabel = gtk.Label("receive address here")
		rcvLabel.set_selectable(True)
		rcvhbox.pack_start(rcvLabel, False, False, 5)

		balFrame = gtk.Frame("balance")
		main_vbox.pack_start(balFrame, True, False, 0)
#~ balHbox
		balhbox = gtk.HBox(False, 0)
		balhbox.set_border_width(5)
		balFrame.add(balhbox)
		balLabel = gtk.Label("balance here")
		refreshButton = gtk.Button("Refresh")
		refreshButton.connect("clicked", self.gBal, balLabel, rcvLabel)
		balhbox.pack_start(refreshButton,  False, False, 5)
		balhbox.pack_end(balLabel,  False, False, 5)

#~ adr / amt vbox
		frame = gtk.Frame("send coins (satoshis)")
		main_vbox.pack_start(frame, True, False, 0)

		vbox = gtk.VBox(False, 0)
		vbox.set_border_width(5)
		frame.add(vbox)

 #~ adr / amt hbox
		hbox = gtk.HBox(False, 0)
		vbox.pack_start(hbox, False, False, 5)

		sendButton = gtk.Button("Send")
		vbox.pack_start(sendButton, False, False, 5)
#~ adrVbox
		adrVbox = gtk.VBox(False, 0)
		hbox.pack_start(adrVbox, True, True, 5)
		adrLabel = gtk.Label("send to address")
		adrLabel.set_alignment(0, 1)

		adrVbox.pack_start(adrLabel, False, False, 0)

		adrEntry = gtk.Entry(50)
		adrEntry.set_size_request(500, -1)
		adrVbox.pack_start(adrEntry, True, True, 0)
#~ amtVbox
		amtVbox = gtk.VBox(False, 0)
		hbox.pack_start(amtVbox, False, False, 5)

		label = gtk.Label("amount")
		label.set_alignment(0, 1)
		amtVbox.pack_start(label, False, False, 0)

		adj = gtk.Adjustment(0, 1000000, 100000000.0, 1.0)
		sendamtSpinner = gtk.SpinButton(adj, 1.0, 0)
		sendamtSpinner.set_wrap(False)
		#~ sendamtSpinner.set_size_request(100, -1)
		amtVbox.pack_start(sendamtSpinner, False, False, 0)


		#~ sendButton.connect("clicked", lambda w: prSend(adrEntry, sendamtSpinner))
		sendButton.connect("clicked", self.dialog, adrEntry, sendamtSpinner)



		quitButton = gtk.Button("Quit")
		quitButton.connect("clicked", lambda w: gtk.main_quit())
		buttonBox = gtk.HBox(False, 0)
		buttonBox.pack_start(quitButton, False, False, 5)
		main_vbox.pack_start(buttonBox, False, False, 5)



		window.show_all()

def main():
	s.connect(("127.0.0.1", 8001))
	gtk.main()
	return 0

if __name__ == "__main__":
	lndrpcui()
	main()
