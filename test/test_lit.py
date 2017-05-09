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

from litpy import litrpc

TMP_DIR = tempfile.mkdtemp(prefix="test")
print("Using tmp dir %s" % TMP_DIR)
LIT_BIN = "%s/../lit" % os.path.abspath(os.path.dirname(__file__))

class LitNode():
    """A class representing a Lit node"""
    def __init__(self, i):
        self.data_dir = TMP_DIR + "/litnode%s" % i
        os.makedirs(self.data_dir)

        # Write a hexkey to the hexkey file
        with open(self.data_dir + "/privkey.hex", 'w+') as f:
            f.write("1" * 63 + str(i) + "\n")

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

        self.args = ["-regtest", "-datadir=%s" % self.data_dir, "-rpcuser=regtestuser", "-rpcpassword=regtestpass", "-rpcport=18332"]
        self.msg_id = random.randint(0, 9999)
        self.rpc_url = "http://regtestuser:regtestpass@127.0.0.1:18332"

    def start_node(self):
        try:
            process = subprocess.Popen([self.__class__.bin_name] + self.args)
        except FileNotFoundError:
            print("%s not found on path. Please install %s" % (self.__class__.bin_name, self.__class__.bin_name))
            sys.exit(1)

        # Wait for process to start
        while True:
            if process.poll() is not None:
                raise Exception('%s exited with status %i during initialization' % (self.__class__.bin_name, process.returncode))
            try:
                self.getblockcount()
                break  # break out of loop on success
            except:
                time.sleep(0.25)

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

class LCNode(BCNode):
    """A class representing a litecoind node"""
    bin_name = "litecoind"
    short_name = "lc"

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
    # takes a while to start on a pi
    time.sleep(20)
    print("generate response: %s" % bcnode.generate(nblocks="101").text)
    time.sleep(5)
    print("Received response from bitcoin node: %s" % bcnode.getinfo().text)

    # Start lit node 0 and open websocket connection
    litnode0 = LitNode(0)
    litnode0.args.extend(["-reg", "127.0.0.1"])
    litnode0.start_node()
    time.sleep(5)
    litnode0.add_rpc_connection("127.0.0.1", "8001")
    print(litnode0.rpc.new_address())
    litnode0.Bal()

    # Start lit node 1 and open websocket connection
    litnode1 = LitNode(1)
    litnode1.args.extend(["-rpcport", "8002", "-reg", "127.0.0.1"])
    litnode1.start_node()
    time.sleep(5)
    litnode1.add_rpc_connection("127.0.0.1", "8002")
    litnode1.rpc.new_address()
    litnode1.Bal()

    # Listen on lit litnode0 and connect from lit litnode1
    res = litnode0.Listen(Port="127.0.0.1:10001")["result"]
    litnode0.lit_address = res["Adr"] + '@' + res["LisIpPorts"][0]

    res = litnode1.Connect(LNAddr=litnode0.lit_address)
    assert not res['error']

    time.sleep(1)
    # Check that litnode0 and litnode1 are connected
    assert len(litnode0.ListConnections()['result']['Connections']) == 1
    assert len(litnode1.ListConnections()['result']['Connections']) == 1

    # Send funds from the bitcoin node to lit node 0
    #print(litnode0.Bal())
    bal = litnode0.Bal()['result']['Balances'][0]["TxoTotal"]
    print("previous bal: " + str(bal))
    addr = litnode0.rpc.new_address()
    bcnode.sendtoaddress(address=addr["result"]["LegacyAddresses"][0], amount=12.34)

    print("waiting to receive transaction")

    # wait for transaction to be received (5 seconds timeout)
    for i in range(50):
        time.sleep(0.1)
        balNew = litnode0.Bal()['result']["Balances"][0]["TxoTotal"]
        if balNew - bal == 1234000000:
            print("Transaction received. Current balance = %s" % balNew)
            break
    else:
        print("Test failed. No transaction received")
        exit(1)

    # Stop bitcoind and lit nodes
    bcnode.stop()
    litnode0.Stop()
    litnode1.Stop()

    print("Test succeeds!")

if __name__ == "__main__":
    testLit()
