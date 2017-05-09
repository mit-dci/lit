#!/usr/bin/python3
"""Test lit"""
import tempfile
import time

from bcnode import BCNode
from litnode import LitNode

TMP_DIR = tempfile.mkdtemp(prefix="test")
print("Using tmp dir %s" % TMP_DIR)

def testLit():
    """starts two lit processes and tests basic functionality:

    - connect over websocket
    - create new address
    - get balance
    - listen on litnode0
    - connect from litnode1 to litnode0
    - stop"""

    # Start a bitcoind node
    bcnode = BCNode(0, TMP_DIR)
    bcnode.start_node()
    # takes a while to start on a pi
    time.sleep(15)
    print("generate response: %s" % bcnode.generate(nblocks=150).text)
    time.sleep(2)
    print("Received response from bitcoin node: %s" % bcnode.getinfo().text)

    # Start lit node 0 and open websocket connection
    litnode0 = LitNode(0, TMP_DIR)
    litnode0.args.extend(["-reg", "127.0.0.1"])
    litnode0.start_node()
    time.sleep(1)
    litnode0.add_rpc_connection("127.0.0.1", "8001")
    print(litnode0.rpc.new_address())
    litnode0.Bal()

    # Start lit node 1 and open websocket connection
    litnode1 = LitNode(1, TMP_DIR)
    litnode1.args.extend(["-rpcport", "8002", "-reg", "127.0.0.1"])
    litnode1.start_node()
    time.sleep(1)
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
    bal = litnode0.Balance()['result']['Balances'][0]["TxoTotal"]
    print("previous bal: " + str(bal))
    addr = litnode0.rpc.new_address()
    bcnode.sendtoaddress(address=addr["result"]["LegacyAddresses"][0], amount=12.34)
    print("generate response: %s" % bcnode.generate(nblocks=1).text)
    print("waiting to receive transaction")

    # wait for transaction to be received (5 seconds timeout)
    for i in range(50):
        time.sleep(0.1)
        balNew = litnode0.Balance()['result']["Balances"][0]["TxoTotal"]
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
