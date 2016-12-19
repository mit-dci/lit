from PyQt4 import QtCore, QtGui

#Extend the library search path to our `qt_files` directory
import sys
sys.path.append("qt_files") 

#ui file import 
import test_ui

class testWindow(QtGui.QMainWindow, test_ui.Ui_MainWindow):
    def __init__(self, parent):
        #Set up calls to get QT working
        QtGui.QMainWindow.__init__(self, parent)
        self.setupUi(self)

        #There is no need for a hint button
        self.setWindowFlags(self.windowFlags() & \
                            ~QtCore.Qt.WindowContextHelpButtonHint)

def main(args):
    app = QtGui.QApplication(args)
    window = testWindow(None)

    window.show()

    sys.exit(app.exec_())

if __name__ == '__main__':
    main(sys.argv)
