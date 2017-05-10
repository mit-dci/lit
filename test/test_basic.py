#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Test basic lit functionality

- start bitcoind process
- start two lit processes
- connect over websocket
- create new address
- get balance
- listen on litnode0
- connect from litnode1 to litnode0
- send funds from bitcoind process to litnode0 address
- stop"""
import time

from bcnode import BCNode
from litnode import LitNode
from lit_test_framework import LitTest

class TestBasic(LitTest):
    def run_test(self):

        # Start a bitcoind node
        self.bcnodes = [BCNode(0, self.tmpdir)]
        self.bcnodes[0].start_node()
        print("generate response: %s" % self.bcnodes[0].generate(nblocks=150).text)

        # Start lit node 0 and open websocket connection
        self.litnodes.append(LitNode(0, self.tmpdir))
        self.litnodes[0].args.extend(["-reg", "127.0.0.1"])
        self.litnodes[0].start_node()
        time.sleep(2)
        self.litnodes[0].add_rpc_connection("127.0.0.1", "8001")
        print(self.litnodes[0].rpc.new_address())
        self.litnodes[0].Balance()

        # Start lit node 1 and open websocket connection
        self.litnodes.append(LitNode(1, self.tmpdir))
        self.litnodes[1].args.extend(["-rpcport", "8002", "-reg", "127.0.0.1"])
        self.litnodes[1].start_node()
        time.sleep(1)
        self.litnodes[1].add_rpc_connection("127.0.0.1", "8002")
        self.litnodes[1].rpc.new_address()
        self.litnodes[1].Balance()

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
        print(self.litnodes[0].Balance()['result'])
        bal = self.litnodes[0].Balance()['result']['Balances'][0]['TxoTotal']
        print("previous bal: " + str(bal))
        addr = self.litnodes[0].rpc.new_address()
        self.bcnodes[0].sendtoaddress(address=addr["result"]["LegacyAddresses"][0], amount=12.34)
        print("generate response: %s" % self.bcnodes[0].generate(nblocks=1).text)
        print("waiting to receive transaction")

        # wait for transaction to be received (5 seconds timeout)
        for i in range(50):
            time.sleep(0.1)
            balNew = self.litnodes[0].Balance()['result']["Balances"][0]["TxoTotal"]
            if balNew - bal == 1234000000:
                print("Transaction received. Current balance = %s" % balNew)
                break
        else:
            print("Test failed. No transaction received")
            raise AssertionError

if __name__ == "__main__":
    exit(TestBasic().main())
