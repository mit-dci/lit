#!/usr/bin/env python

from PyQt4 import QtCore, QtGui

#Extend the library search path to our `qt_files` directory
import sys
sys.path.append("qt_files") 

import socket
import json

#ui file import 
import rpcui_ui

#Handles rpc communications and conjugate response handler functions
class rpcCom():
    def __init__(self, addr, port):

        #Open up the socket connection
        self.conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.conn.connect((addr, port))

    def getBal(self):
        r = self.rpcCommand('Bal', {})

        return r['result']['TxoTotal']

    def getAdr(self):
        r = self.rpcCommand('Address', {'NumToMake' : 0})

	return r['result']['Addresses'][-1]

    def prSend(self, adr, amt):
        r = self.rpcCommand('Send', {"DestAddrs": [adr], "Amts": [amt]})

	return "Sent. TXID: " + r['result']['Txids'][0]

    #Makes an rpc call on a given, arbitrary `command` and its `params` in json (dict) format
    # The `params` can be a dict or a string of dict-form as it is internally handled accordingly
    #
    #Ex: self.rpcCommand('Address', '{"NumToMake": 0}') and self.rpcCommand('Address', {"NumToMake": 0})
    # are both equivalent and correct calls
    def rpcCommand(self, command, params):
        #Tentatively assign `json_params` for now, assuming it is a dict
        json_params = params
        
        #If `params` is a string, load it as a json dict into `json_params`. Otherwise, it should
        # be a dict for which can be directly plugged into the `rpcCmd`
        if type(params) == str:
            #Upon error, re-raise the error with a more meaningful message
            try:
                json_params = json.loads(params)
            except:
                raise RuntimeError("Poorly formatted input parameters: %s" % params)
        
        #Build the `rpcCmd`
        rpcCmd = {
            'method' : "LitRPC" + "." + command,
            'params' : [json_params]
        }

        rpcCmd.update({'jsonrpc' : "2.0", 'id' : "96"})

        self.conn.sendall(json.dumps(rpcCmd))
        r = json.loads(self.conn.recv(8000000))

	if r['error'] != None:
            raise RuntimeError(r['error'])

        return r

class mainWindow(QtGui.QMainWindow, rpcui_ui.Ui_MainWindow):
    def __init__(self, parent):
        #Set up calls to get QT working
        QtGui.QMainWindow.__init__(self, parent)
        self.setupUi(self)

        #There is no need for a hint button
        self.setWindowFlags(self.windowFlags() & ~QtCore.Qt.WindowContextHelpButtonHint)

        #Set up the RPC communication object
        self.rpcCom = rpcCom("127.0.0.1", 9750)

        #Setup the connections to their triggers
        self.setup_connections()

    #Sets the text value for the balance label. Make this its own function to 
    # be used as a callback for the "Refresh" button
    def set_bal_label(self):
        bal = self.rpcCom.getBal()
        self.bal_label.setText(str(bal))

    #Give the `statusbar` some text, usual error messages
    def set_statusbar(self, text):
        #Display the `text` for 10 seconds
        self.statusbar.showMessage(text, 10 * 1000)

    #The trigger handler for the send button being clicked
    def send_button_clicked(self):
        #TODO: Implement address validity verification
        to_addr = str(self.send_addr_line_edit.text())
        amt = self.send_amt_spin_box.value()
        
        try:
            if amt == 0:
                raise RuntimeError("Invalid input send amount")

            self.rpcCom.prSend(to_addr, amt)
        except Exception as error:
            self.set_statusbar("Error: " + str(error))

    #The trigger handler for the rpc line edit field for when the return key is pressed
    def rpc_line_edit_return_pressed(self):
        rpc_cmd = str(self.rpc_line_edit.text())
        rpc_split = rpc_cmd.split(" ", 1) #Split once around the first space

        try:
            #Assume that this is only a command w/o params
            if len(rpc_split) == 1:
                rpc_split.append('{}') #Append some empty params
            elif len(rpc_split) != 2:
                raise RuntimeError("Invalid input! Should be: Command {json-args}")

            rpc_response = self.rpcCom.rpcCommand(*rpc_split)

            #If the program made it this far, `rpc_split` and `rpc_response` should be valid. 
            # Append the command, params, and respective response
            fmt_cmd = rpc_split[0] + " " + rpc_split[1]
            self.rpc_console_text_browser.append(fmt_cmd + " : " + str(rpc_response['result']))
        except Exception as error:
            self.set_statusbar("Error: " + str(error))
        finally:
            #Clear the line edit after appending the result
            self.rpc_line_edit.clear()

    def setup_connections(self):
        #Populate the address label
        addr = self.rpcCom.getAdr()
        self.addr_label.setText(addr);

        #Populate the balance label
        self.set_bal_label()

        #Connect the trigger for the "Refresh" button
        self.bal_refresh_button.clicked.connect(self.set_bal_label)

        #Connect the trigger for the "Send" button
        self.send_button.clicked.connect(self.send_button_clicked)

        #Connect the trigger for the rpc line edit return pressed
        self.rpc_line_edit.returnPressed.connect(self.rpc_line_edit_return_pressed)

        #Connect the trigger for the "Clear" button for the rpc console
        # Simply clears the text browser
        self.clear_button.clicked.connect(lambda x: self.rpc_console_text_browser.clear())

def main(args):
    app = QtGui.QApplication(args)
    window = mainWindow(None)

    window.show()

    sys.exit(app.exec_())

if __name__ == '__main__':
    main(sys.argv)
