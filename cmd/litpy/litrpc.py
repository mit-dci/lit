#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Python interface to lit"""

import json
import logging
import random
import time
from urllib.request import urlopen
import urllib.request
import http.client
import ssl, socket

logger = logging.getLogger("litrpc")

class LitConnection():
    """A class representing a connection to a lit node."""
    def __init__(self, ip, port):
        self.ip = ip
        self.port = port

    def connect(self):
        """Connect to the node. Continue trying for 10 seconds"""
        logger.debug("Opening RPC connection to litnode %s:%s" % (self.ip, self.port))
        HOST, PORT = self.ip, int(self.port)

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(10)
        self.ws = ssl.wrap_socket(sock)
        for _ in range(50):
            try:
                self.ws.connect((HOST, PORT))
            except Exception as e:
                print (e)
                # lit is not ready to accept connections yet
                time.sleep(0.25)
            else:
                # No exception - we're connected!
                break
        self.msg_id = random.randint(0, 9999)


    def send_message(self, method, params):
        """Sends a socket message to the lit node"""
        logger.debug("Sending rpc message to %s:%s %s(%s)" % (self.ip, self.port, method, str(params)))
        self.ws.send(json.dumps({"method": "LitRPC.%s" % method,
                                 "params": [params],
                                 "jsonrpc": "2.0",
                                 "id": str(self.msg_id)}).encode('utf-8'))

        self.msg_id = self.msg_id + 1 % 10000

        resp = json.loads(self.ws.recv().decode('utf-8'))
        logger.debug("Recieved rpc response from %s:%s method: %s Response: %s." % (self.ip, self.port, method, str(resp)))
        return resp

    def __getattr__(self, name):
        """Dispatches any unrecognised messages to the https connection"""
        def dispatcher(**kwargs):
            return self.send_message(name, kwargs)
        return dispatcher

    def new_address(self):
        """Add a new wallit address"""
        return self.Address(NumToMake=1)

    def balance(self):
        """Get wallit balance"""
        return self.Bal()
