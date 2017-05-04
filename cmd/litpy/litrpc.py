#!/usr/bin/python3
"""Python interface to lit"""

import json
import random
import websocket  # `pip install websocket-client`

def litconnect():
    """Connect to a lit node"""
    ws = websocket.WebSocket()
    ws.connect("ws://127.0.0.1:8001/ws")
    return ws

def litNewAddr(wsconn):
    """Add a new wallit address"""
    rpcCmd = {
        "method": "LitRPC.Address",
        "params": [{"NumToMake": 0}]
    }

    rid = random.randint(0, 9999)
    rpcCmd.update({"jsonrpc": "2.0", "id": str(rid)})

    wsconn.send(json.dumps(rpcCmd))
    resp = json.loads(wsconn.recv())
    return resp["result"]["WitAddresses"][0]

def litSend(wsconn, adr, amt):
    """Send amt to adr"""
    rpcCmd = {
        "method": "LitRPC.Send",
        "params": [
            {"DestAddrs": adr, "Amts": amt},
        ]
    }

    rid = random.randint(0, 9999)
    rpcCmd.update({"jsonrpc": "2.0", "id": str(rid)})
    wsconn.send(json.dumps(rpcCmd))
    resp = json.loads(wsconn.recv())
    return resp

def litBalance(wsconn):
    """Get wallit balance"""
    rpcCmd = {
        "method": "LitRPC.Bal",
        "params": []
    }
    rpcCmd.update({"jsonrpc": "2.0", "id": "92"})

    wsconn.send(json.dumps(rpcCmd))
    return json.loads(wsconn.recv())

if __name__ == '__main__':
    ws = litconnect()
    resp = litNewAddr(ws)
    print(resp)
