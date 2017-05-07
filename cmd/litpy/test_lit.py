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

    def add_rpc_connection(self, ip, port):
        self.rpc = litrpc.LitConnection(ip, port)
        self.rpc.connect()

    def __getattr__(self, name):
        return self.rpc.__getattr__(name)

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
    node0.add_rpc_connection("127.0.0.1", "8001")
    node0.new_address()
    node0.Bal()

    # Start lit node 1 and open websocket connection
    node1 = LitNode(1)
    node1.args.extend(["-rpcport", "8002"])
    node1.start_node()
    node1.add_rpc_connection("127.0.0.1", "8002")
    node1.new_address()
    node1.Bal()

    # Listen on lit node0 and connect from lit node1
    res = node0.Listen(Port="127.0.0.1:10001")["result"]
    node0.lit_address = res["Status"].split(' ')[5] + '@' + res["Status"].split(' ')[2]

    res = node1.Connect(LNAddr=node0.lit_address)
    assert not res['error']

    # Check that node0 and node1 are connected
    assert len(node0.ListConnections()['result']['Connections']) == 1
    assert len(node1.ListConnections()['result']['Connections']) == 1

    # Stop lit nodes
    node0.Stop()
    node1.Stop()

    print("Test succeeds!")

if __name__ == "__main__":
    testLit()
