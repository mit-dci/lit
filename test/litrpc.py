#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
#
# Stolen from https://github.com/mit-dci/lit/blob/c565392054d2c2c1cf5d2917e40ab69188163808/cmd/litpy/litrpc.py
"""Python interface to lit"""

import json
import logging
import random
import time

import requests # pip3 install requests

logger = logging.getLogger("testframework")

max_id = 10000

class LitClient():
    """A class representing a connection to a lit node."""
    def __init__(self, ip, port):
        self.ip = ip
        self.port = port
        self.msg_id = random.randint(0, max_id)

    def send_message(self, method, params):
        """Sends a POST message to the lit node"""
        logger.debug("Sending lit rpc message to %s:%s %s(%s)".format(self.ip, self.port, method, str(params)))

        jsonreq = {
            'method': "LitRPC.%s" % method,
            'params': [params],
            'jsonrpc': '2.0',
            'id': self.msg_id
        }
        self.msg_id = (self.msg_id + 1) % max_id

        req = requests.post('http://{}:{}/oneoff'.format(self.ip, self.port), json=jsonreq)
        if req.status_code != 200:
            raise AssertionError('RPC error: HTTP response code is ' + str(req.status_code))

        if req.json()['error'] is None:
            resp = req.json()['result']
            logger.debug("Received rpc response from %s:%s method: %s Response: %s." % (self.ip, self.port, method, str(resp)))
            return resp
        else:
            raise AssertionError('RPC call failed: ' + req.json()['error'])

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
        return self.Balance()['Balances']
