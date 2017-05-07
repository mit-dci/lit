#!/usr/bin/python3
"""Test lit"""
import json
import os
import random
import subprocess
import sys
import tempfile
import time

import requests  # `pip install requests`

import litrpc

TMP_DIR = tempfile.mkdtemp(prefix="test")
LIT_BIN = "%s/../../lit" % os.path.abspath(os.path.dirname(__file__))

class LitNode():
    """A class representing a Lit node"""
    def __init__(self, i):
        self.data_dir = TMP_DIR + "/litnode%s" % i
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

class BCNode():
    """A class representing a bitcoind node"""
    bin_name = "bitcoind"
    short_name = "bc"

    def __init__(self, i):
        self.data_dir = TMP_DIR + "/%snode%s" % (self.__class__.short_name, i)
        os.makedirs(self.data_dir)

        self.args = ["-daemon", "-regtest", "-datadir=%s" % self.data_dir, "-rpcuser=regtestuser", "-rpcpassword=regtestpass", "-rpcport=18332"]
        self.msg_id = random.randint(0, 9999)
        self.rpc_url = "http://regtestuser:regtestpass@127.0.0.1:18332"

    def start_node(self):
        try:
            subprocess.Popen([self.__class__.bin_name] + self.args)
        except FileNotFoundError:
            print("%s not found on path. Please install %s" % (self.__class__.bin_name, self.__class__.bin_name))
            sys.exit(1)

    def send_message(self, method, params):
        self.msg_id += 1
        rpcCmd = {
            "method": method,
            "params": params,
            "jsonrpc": "2.0",
            "id": str(self.msg_id)
        }
        payload = json.dumps(rpcCmd)

        return requests.post(self.rpc_url, headers={"Content-type": "application/json"}, data=payload)

    def __getattr__(self, name):
        """Dispatches any unrecognised messages to the websocket connection"""
        def dispatcher(**kwargs):
            return self.send_message(name, kwargs)
        return dispatcher

def testLit():
    """starts two lit processes and tests basic functionality:

    - connect over websocket
    - create new address
    - get balance
    - listen on litnode0
    - connect from litnode1 to litnode0
    - stop"""

    # Start a bitcoind node
    bcnode = BCNode(0)
    bcnode.start_node()
    time.sleep(3)

    # Start lit node 0 and open websocket connection
    litnode0 = LitNode(0)
    litnode0.start_node()
    litnode0.add_rpc_connection("127.0.0.1", "8001")
    litnode0.new_address()
    litnode0.Bal()

    # Start lit node 1 and open websocket connection
    litnode1 = LitNode(1)
    litnode1.args.extend(["-rpcport", "8002"])
    litnode1.start_node()
    litnode1.add_rpc_connection("127.0.0.1", "8002")
    litnode1.new_address()
    litnode1.Bal()

    # Listen on lit litnode0 and connect from lit litnode1
    res = litnode0.Listen(Port="127.0.0.1:10001")["result"]
    litnode0.lit_address = res["Status"].split(' ')[5] + '@' + res["Status"].split(' ')[2]

    res = litnode1.Connect(LNAddr=litnode0.lit_address)
    assert not res['error']

    # Check that litnode0 and litnode1 are connected
    assert len(litnode0.ListConnections()['result']['Connections']) == 1
    assert len(litnode1.ListConnections()['result']['Connections']) == 1

    # Stop bitcoind and lit nodes
    bcnode.stop()
    litnode0.Stop()
    litnode1.Stop()

    print("Test succeeds!")

if __name__ == "__main__":
    testLit()
