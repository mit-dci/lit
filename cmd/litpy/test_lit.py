#!/usr/bin/python3
"""Test lit"""
import os
import subprocess
import tempfile

import litrpc

TMP_DIR = tempfile.mkdtemp(prefix="test")
LIT_BIN = "%s/../../lit" % os.path.abspath(os.path.dirname(__file__))

class LitNode():
    """A class representing a Lit node"""
    def __init__(self, i):
        self.data_dir = TMP_DIR + "litnode%s" % i
        os.makedirs(self.data_dir)

        # Write a hexkey to the hexkey file
        with open(self.data_dir + "/testkey.hex", 'w+') as f:
            f.write("1" * 64 + "\n")

        self.args = ["-dir", self.data_dir]

    def start_node(self):
        subprocess.Popen([LIT_BIN] + self.args)


def testLit():
    """starts a lit process and tests basic functionality:

    - connect over websocket
    - create new address
    - get balance
    - stop"""

    node = LitNode(1)
    node.start_node()

    litConn = litrpc.LitConnection("127.0.0.1", "8001")
    litConn.connect()
    litConn.new_address()
    litConn.Bal()
    litConn.Stop()

    print("Test succeeds!")

if __name__ == "__main__":
    testLit()
