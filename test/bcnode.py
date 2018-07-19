#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Classes representing bitcoind and litecoind nodes

BCNode and LCNode represent a bitcoind and litecoind node respectively.
They can be used to start/stop a bitcoin/litecoin node and communicate
with it over RPC."""
import json
import logging
import os
import random
import subprocess
import time

import requests  # `pip install requests`

logger = logging.getLogger("TestFramework.bcnode")

class BCNode():
    """A class representing a bitcoind node"""
    bin_name = "bitcoind"
    short_name = "bc"
    min_version = 140000

    index = 0

    def __init__(self, tmd_dir):
        self.index = self.__class__.index
        self.__class__.index += 1

        self.data_dir = tmd_dir + "/%snode%d" % (self.__class__.short_name, self.index)
        os.makedirs(self.data_dir)

        self.args = ["-regtest", "-datadir=%s" % self.data_dir, "-rpcuser=regtestuser", "-rpcpassword=regtestpass", "-rpcport=18332", "-logtimemicros"]
        self.msg_id = random.randint(0, 9999)
        self.rpc_url = "http://regtestuser:regtestpass@127.0.0.1:18332"

    def start_node(self):
        logger.debug("Starting %s%d" % (self.__class__.bin_name, self.index))
        try:
            self.process = subprocess.Popen([self.__class__.bin_name] + self.args, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        except FileNotFoundError:
            raise Exception("%s not found on path. Please install %s" % (self.__class__.bin_name, self.__class__.bin_name))

        # Wait for process to start
        while True:
            if self.process.poll() is not None:
                raise Exception('%s exited with status %i during initialization' % (self.__class__.bin_name, self.process.returncode))
            try:
                resp = self.getnetworkinfo()
                if resp.json()['error'] and resp.json()['error']['code'] == -28:
                    # RPC is still in warmup. Sleep some more.
                    continue
                # Check that we're running at least the minimum version
                assert resp.json()['result']['version'] >= self.__class__.min_version
                logger.debug("bcnode %d started" % self.index)
                break  # break out of loop on success
            except requests.exceptions.ConnectionError as e:
                time.sleep(0.25)

    def send_message(self, method, pos_args, named_args):
        if pos_args and named_args:
            raise AssertionError("RPCs must not use a mix of positional and named arguments")
        elif named_args:
            params = named_args
        else:
            params = list(pos_args)
        logger.debug("Sending message %s, params: %s" % (method, str(params)))
        self.msg_id += 1
        rpcCmd = {
            "method": method,
            "params": params,
            "jsonrpc": "2.0",
            "id": str(self.msg_id)
        }
        payload = json.dumps(rpcCmd)

        resp = requests.post(self.rpc_url, headers={"Content-type": "application/json"}, data=payload)

        logger.debug("Response received for %s, %s" % (method, resp.text))

        return resp

    def __getattr__(self, name):
        """Dispatches any unrecognised messages to the websocket connection"""
        def dispatcher(*args, **kwargs):
            return self.send_message(name, args, kwargs)
        return dispatcher

class LCNode(BCNode):
    """A class representing a litecoind node"""
    bin_name = "litecoind"
    short_name = "lc"
    min_version = 130200
