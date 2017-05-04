#!/usr/bin/python3
"""Test lit"""
import os
import subprocess
import tempfile

import litrpc

def testLit():
    """starts a lit process and tests basic functionality:

    - connect over websocket
    - create new address
    - get balance
    - stop"""

    DATA_DIR = tempfile.mkdtemp(prefix="test") + "/.lit"
    os.makedirs(DATA_DIR)
    LIT_BIN = "%s/../../lit" % os.path.abspath(os.path.dirname(__file__))

    # Write a hexkey to the hexkey file
    with open(DATA_DIR + "/testkey.hex", 'w+') as f:
        f.write("1" * 64 + "\n")

    subprocess.Popen([LIT_BIN, "-dir", DATA_DIR])

    litConn = litrpc.LitConnection("127.0.0.1", "8001")
    litConn.connect()
    litConn.new_address()
    litConn.balance()
    litConn.stop()

    print("Test succeeds!")

if __name__ == "__main__":
    testLit()
