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
    """starts two lit processes and tests basic functionality:

    - connect over websocket
    - create new address
    - get balance
    - listen on node0
    - connect from node1 to node0
    - stop"""

    # Start lit node 0 and open websocket connection
    node0 = LitNode(0)
    node0.start_node()

    litConn0 = litrpc.LitConnection("127.0.0.1", "8001")
    litConn0.connect()
    litConn0.new_address()
    litConn0.Bal()

    # Start lit node 1 and open websocket connection
    node1 = LitNode(1)
    node1.args.extend(["-rpcport", "8002"])
    node1.start_node()

    litConn1 = litrpc.LitConnection("127.0.0.1", "8002")
    litConn1.connect()
    litConn1.new_address()
    litConn1.Bal()

    # Listen on lit node0 and connect from lit node1
    res = litConn0.Listen(Port="127.0.0.1:10001")["result"]
    node0.lit_address = res["Status"].split(' ')[5] + '@' + res["Status"].split(' ')[2]

    res = litConn1.Connect(LNAddr=node0.lit_address)
    assert not res['error']

    # Check that node0 and node1 are connected
    assert len(litConn0.ListConnections()['result']['Connections']) == 1
    assert len(litConn1.ListConnections()['result']['Connections']) == 1

    # Stop lit nodes
    litConn0.Stop()
    litConn1.Stop()

    print("Test succeeds!")

if __name__ == "__main__":
    testLit()
