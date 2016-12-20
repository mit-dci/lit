# -*- coding: utf-8 -*-

# Form implementation generated from reading ui file 'rpcui.ui'
#
# Created: Mon Dec 19 22:04:30 2016
#      by: PyQt4 UI code generator 4.10.4
#
# WARNING! All changes made in this file will be lost!

from PyQt4 import QtCore, QtGui

try:
    _fromUtf8 = QtCore.QString.fromUtf8
except AttributeError:
    def _fromUtf8(s):
        return s

try:
    _encoding = QtGui.QApplication.UnicodeUTF8
    def _translate(context, text, disambig):
        return QtGui.QApplication.translate(context, text, disambig, _encoding)
except AttributeError:
    def _translate(context, text, disambig):
        return QtGui.QApplication.translate(context, text, disambig)

class Ui_MainWindow(object):
    def setupUi(self, MainWindow):
        MainWindow.setObjectName(_fromUtf8("MainWindow"))
        MainWindow.resize(700, 350)
        MainWindow.setMinimumSize(QtCore.QSize(700, 350))
        MainWindow.setWindowOpacity(1.0)
        self.centralwidget = QtGui.QWidget(MainWindow)
        self.centralwidget.setObjectName(_fromUtf8("centralwidget"))
        self.verticalLayout = QtGui.QVBoxLayout(self.centralwidget)
        self.verticalLayout.setObjectName(_fromUtf8("verticalLayout"))
        self.addr_frame = QtGui.QFrame(self.centralwidget)
        self.addr_frame.setFrameShape(QtGui.QFrame.StyledPanel)
        self.addr_frame.setFrameShadow(QtGui.QFrame.Raised)
        self.addr_frame.setObjectName(_fromUtf8("addr_frame"))
        self.horizontalLayout = QtGui.QHBoxLayout(self.addr_frame)
        self.horizontalLayout.setObjectName(_fromUtf8("horizontalLayout"))
        self.label_2 = QtGui.QLabel(self.addr_frame)
        self.label_2.setObjectName(_fromUtf8("label_2"))
        self.horizontalLayout.addWidget(self.label_2)
        self.addr_label = QtGui.QLabel(self.addr_frame)
        self.addr_label.setObjectName(_fromUtf8("addr_label"))
        self.horizontalLayout.addWidget(self.addr_label)
        spacerItem = QtGui.QSpacerItem(40, 20, QtGui.QSizePolicy.Expanding, QtGui.QSizePolicy.Minimum)
        self.horizontalLayout.addItem(spacerItem)
        self.verticalLayout.addWidget(self.addr_frame)
        spacerItem1 = QtGui.QSpacerItem(20, 40, QtGui.QSizePolicy.Minimum, QtGui.QSizePolicy.Expanding)
        self.verticalLayout.addItem(spacerItem1)
        self.bal_frame = QtGui.QFrame(self.centralwidget)
        self.bal_frame.setFrameShape(QtGui.QFrame.StyledPanel)
        self.bal_frame.setFrameShadow(QtGui.QFrame.Raised)
        self.bal_frame.setObjectName(_fromUtf8("bal_frame"))
        self.gridLayout = QtGui.QGridLayout(self.bal_frame)
        self.gridLayout.setObjectName(_fromUtf8("gridLayout"))
        self.bal_refresh_button = QtGui.QPushButton(self.bal_frame)
        self.bal_refresh_button.setObjectName(_fromUtf8("bal_refresh_button"))
        self.gridLayout.addWidget(self.bal_refresh_button, 1, 0, 1, 1)
        self.bal_label = QtGui.QLabel(self.bal_frame)
        self.bal_label.setObjectName(_fromUtf8("bal_label"))
        self.gridLayout.addWidget(self.bal_label, 0, 1, 1, 1)
        spacerItem2 = QtGui.QSpacerItem(40, 20, QtGui.QSizePolicy.Expanding, QtGui.QSizePolicy.Minimum)
        self.gridLayout.addItem(spacerItem2, 0, 2, 1, 1)
        self.label_3 = QtGui.QLabel(self.bal_frame)
        self.label_3.setObjectName(_fromUtf8("label_3"))
        self.gridLayout.addWidget(self.label_3, 0, 0, 1, 1)
        self.verticalLayout.addWidget(self.bal_frame)
        spacerItem3 = QtGui.QSpacerItem(20, 40, QtGui.QSizePolicy.Minimum, QtGui.QSizePolicy.Expanding)
        self.verticalLayout.addItem(spacerItem3)
        self.send_frame = QtGui.QFrame(self.centralwidget)
        self.send_frame.setFrameShape(QtGui.QFrame.StyledPanel)
        self.send_frame.setFrameShadow(QtGui.QFrame.Raised)
        self.send_frame.setObjectName(_fromUtf8("send_frame"))
        self.gridLayout_2 = QtGui.QGridLayout(self.send_frame)
        self.gridLayout_2.setObjectName(_fromUtf8("gridLayout_2"))
        self.spinBox = QtGui.QSpinBox(self.send_frame)
        self.spinBox.setAccelerated(True)
        self.spinBox.setMaximum(999999999)
        self.spinBox.setObjectName(_fromUtf8("spinBox"))
        self.gridLayout_2.addWidget(self.spinBox, 1, 1, 1, 1)
        self.lineEdit = QtGui.QLineEdit(self.send_frame)
        self.lineEdit.setText(_fromUtf8(""))
        self.lineEdit.setObjectName(_fromUtf8("lineEdit"))
        self.gridLayout_2.addWidget(self.lineEdit, 1, 0, 1, 1)
        self.label_6 = QtGui.QLabel(self.send_frame)
        self.label_6.setObjectName(_fromUtf8("label_6"))
        self.gridLayout_2.addWidget(self.label_6, 0, 1, 1, 1)
        self.label_4 = QtGui.QLabel(self.send_frame)
        self.label_4.setObjectName(_fromUtf8("label_4"))
        self.gridLayout_2.addWidget(self.label_4, 0, 0, 1, 1)
        self.send_button = QtGui.QPushButton(self.send_frame)
        self.send_button.setObjectName(_fromUtf8("send_button"))
        self.gridLayout_2.addWidget(self.send_button, 2, 1, 1, 1)
        self.verticalLayout.addWidget(self.send_frame)
        MainWindow.setCentralWidget(self.centralwidget)
        self.menubar = QtGui.QMenuBar(MainWindow)
        self.menubar.setGeometry(QtCore.QRect(0, 0, 700, 27))
        self.menubar.setObjectName(_fromUtf8("menubar"))
        MainWindow.setMenuBar(self.menubar)
        self.statusbar = QtGui.QStatusBar(MainWindow)
        self.statusbar.setObjectName(_fromUtf8("statusbar"))
        MainWindow.setStatusBar(self.statusbar)

        self.retranslateUi(MainWindow)
        QtCore.QMetaObject.connectSlotsByName(MainWindow)

    def retranslateUi(self, MainWindow):
        MainWindow.setWindowTitle(_translate("MainWindow", "MainWindow", None))
        self.label_2.setText(_translate("MainWindow", "Receive Address:", None))
        self.addr_label.setText(_translate("MainWindow", "address here", None))
        self.bal_refresh_button.setText(_translate("MainWindow", "Refresh", None))
        self.bal_label.setText(_translate("MainWindow", "balance here", None))
        self.label_3.setText(_translate("MainWindow", "Current Balance:", None))
        self.lineEdit.setPlaceholderText(_translate("MainWindow", "Address Here", None))
        self.label_6.setText(_translate("MainWindow", "Amount (Satoshi)", None))
        self.label_4.setText(_translate("MainWindow", "Send To:", None))
        self.send_button.setText(_translate("MainWindow", "Send", None))

