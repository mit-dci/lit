#!/usr/bin/env python

from PyQt4 import QtCore, QtGui

#Extend the library search path to our `qt_files` directory
import sys
sys.path.append("qt_files") 

import socket
import json

#ui file import 
import rpcui_ui

class mainWindow(QtGui.QMainWindow, rpcui_ui.Ui_MainWindow):
    def __init__(self, parent):
        #Set up calls to get QT working
        QtGui.QMainWindow.__init__(self, parent)
        self.setupUi(self)

        #There is no need for a hint button
        self.setWindowFlags(self.windowFlags() & ~QtCore.Qt.WindowContextHelpButtonHint)

        #Open up the socket connection
        self.conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.conn.connect(("127.0.0.1", 1234))

        #Start populating the initial fields
        self.getBal() #ADDED

    def getBal(self):
        rpcCmd = {
            "method": "LNRpc.Bal",
            "params": [{
            }]
        }
        rpcCmd.update({"jsonrpc": "2.0", "id": "99"})
        print json.dumps(rpcCmd)
        self.conn.sendall(json.dumps(rpcCmd))
        r = json.loads(self.conn.recv(8000000))
        print r
        return r["result"]["TotalScore"]

def main(args):
    app = QtGui.QApplication(args)
    window = mainWindow(None)

    window.show()

    sys.exit(app.exec_())

if __name__ == '__main__':
    main(sys.argv)
