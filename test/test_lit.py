#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Test lit"""
import subprocess
import tempfile
import time

from bcnode import BCNode
from litnode import LitNode

TMP_DIR = tempfile.mkdtemp(prefix="test")
print("Using tmp dir %s" % TMP_DIR)

class LitTest():
    def __init__(self):
        self.litnodes = []
        self.bcnodes = []

    def main(self):
        rc = 0
        try:
            self.run_test()
            print("Test succeeds!")
        except:
            # Test asserted. Return 1
            rc = 1
            print("Test fails")
        finally:
            self.cleanup()

        return rc

    def run_test(self):
        """starts two lit processes and tests basic functionality:

        - connect over websocket
        - create new address
        - get balance
        - listen on litnode0
        - connect from litnode1 to litnode0
        - stop"""

        # Start a bitcoind node
        self.bcnodes = [BCNode(0, TMP_DIR)]
        self.bcnodes[0].start_node()
        time.sleep(15)
        print("generate response: %s" % bcnode.generate(nblocks=150).text)
        time.sleep(2)
        print("Received response from bitcoin node: %s" % self.bcnodes[0].getinfo().text)

        # Start lit node 0 and open websocket connection
        self.litnodes.append(LitNode(0, TMP_DIR))
        self.litnodes[0].args.extend(["-reg", "127.0.0.1"])
        self.litnodes[0].start_node()
        time.sleep()
        self.litnodes[0].add_rpc_connection("127.0.0.1", "8001")
        print(self.litnodes[0].rpc.new_address())
        self.litnodes[0].Bal()

        # Start lit node 1 and open websocket connection
        self.litnodes[1] = LitNode(1, TMP_DIR)
        self.litnodes[1].args.extend(["-rpcport", "8002", "-reg", "127.0.0.1"])
        self.litnodes[1].start_node()
        time.sleep(1)
        self.litnodes[1].add_rpc_connection("127.0.0.1", "8002")
        self.litnodes[1].rpc.new_address()
        self.litnodes[1].Bal()

        # Listen on litnode0 and connect from litnode1
        res = self.litnodes[0].Listen(Port="127.0.0.1:10001")["result"]
        self.litnodes[0].lit_address = res["Adr"] + '@' + res["LisIpPorts"][0]

        res = self.litnodes[1].Connect(LNAddr=self.litnodes[0].lit_address)
        assert not res['error']

        time.sleep(1)
        # Check that litnode0 and litnode1 are connected
        assert len(self.litnodes[0].ListConnections()['result']['Connections']) == 1
        assert len(self.litnodes[1].ListConnections()['result']['Connections']) == 1

        # Send funds from the bitcoin node to litnode0
        bal = self.litnodes[0].Bal()['result']['balances'][0]['TxoTotal']
        print("previous bal: " + str(bal))
        addr = self.litnodes[0].rpc.new_address()
        self.bcnodes[0].sendtoaddress(address=addr["result"]["LegacyAddresses"][0], amount=12.34)
        print("generate response: %s" % bcnodes[0].generate(nblocks=1).text)
        print("waiting to receive transaction")

        # wait for transaction to be received (5 seconds timeout)
        for i in range(50):
            time.sleep(0.1)
            balNew = self.litnodes[0].Bal()['result']["Balances"][0]["TxoTotal"]
            if balNew - bal == 1234000000:
                print("Transaction received. Current balance = %s" % balNew)
                break
        else:
            print("Test failed. No transaction received")
            exit(1)


    def cleanup(self):
        # Stop bitcoind and lit nodes
        for bcnode in self.bcnodes:
            bcnode.stop()
            try:
                bcnode.process.wait(2)
            except subprocess.TimeoutExpired:
                bcnode.process.kill()
        for litnode in self.litnodes:
            litnode.Stop()
            try:
                litnode.process.wait(2)
            except subprocess.TimeoutExpired:
                litnode.process.kill()

if __name__ == "__main__":
    exit(LitTest().main())
