#!/usr/bin/python3
"""Python interface to lit"""

import json
import random
import time
import websocket  # `pip install websocket-client`

class LitConnection():
    """A class representing a connection to a lit node."""
    def __init__(self, ip, port):
        self.ip = ip
        self.port = port

    def connect(self):
        """Connect to the node. Continue trying for 10 seconds"""
        self.ws = websocket.WebSocket()
        for _ in range(50):
            try:
                self.ws.connect("ws://%s:%s/ws" % (self.ip, self.port))
            except ConnectionRefusedError:
                # lit is not ready to accept connections yet
                time.sleep(0.2)
            else:
                # No exception - we're connected!
                break

    def newAddr(self):
        """Add a new wallit address"""
        rpcCmd = {
            "method": "LitRPC.Address",
            "params": [{"NumToMake": 0}]
        }

        rid = random.randint(0, 9999)
        rpcCmd.update({"jsonrpc": "2.0", "id": str(rid)})

        self.ws.send(json.dumps(rpcCmd))
        resp = json.loads(self.ws.recv())
        return resp["result"]["WitAddresses"][0]

    def balance(self):
        """Get wallit balance"""
        rpcCmd = {
            "method": "LitRPC.Bal",
            "params": []
        }
        rpcCmd.update({"jsonrpc": "2.0", "id": "92"})

        self.ws.send(json.dumps(rpcCmd))
        return json.loads(self.ws.recv())

    def send(self, adr, amt):
        """Send amt to adr"""
        rpcCmd = {
            "method": "LitRPC.Send",
            "params": [
                {"DestAddrs": adr, "Amts": amt},
            ]
        }

        rid = random.randint(0, 9999)
        rpcCmd.update({"jsonrpc": "2.0", "id": str(rid)})
        self.ws.send(json.dumps(rpcCmd))
        resp = json.loads(self.ws.recv())
        return resp

if __name__ == '__main__':
    litConn = LitConnection("127.0.0.1", "8001")
    litConn.connect()
    print(litConn.newAddr())
    print(litConn.balance())
