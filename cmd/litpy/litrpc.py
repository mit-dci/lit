#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
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
        self.msg_id = random.randint(0, 9999)

    def send_message(self, method, params):
        """Sends a websocket message to the lit node"""
        self.ws.send(json.dumps({"method": "LitRPC.%s" % method,
                                 "params": [params],
                                 "jsonrpc": "2.0",
                                 "id": str(self.msg_id)}))

        self.msg_id = self.msg_id + 1 % 10000
        return json.loads(self.ws.recv())

    def __getattr__(self, name):
        """Dispatches any unrecognised messages to the websocket connection"""
        def dispatcher(**kwargs):
            return self.send_message(name, kwargs)
        return dispatcher

    def new_address(self):
        """Add a new wallit address"""
        return self.Address(NumToMake=1)

    def balance(self):
        """Get wallit balance"""
        return self.Bal()

if __name__ == '__main__':
    """Test litrpc.py. lit instance must be running and available on 127.0.0.1:8001"""
    litConn = LitConnection("127.0.0.1", "8001")
    litConn.connect()
    print(litConn.new_address())
    print(litConn.balance())
