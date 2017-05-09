#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Classes representing bitcoind and litecoind nodes

BCNode and LCNode represent a bitcoind and litecoind node respectively.
They can be used to start/stop a bitcoin/litecoin node and communicate
with it over RPC."""
import json
import os
import random
import subprocess
import sys
import time

import requests  # `pip install requests`

class BCNode():
    """A class representing a bitcoind node"""
    bin_name = "bitcoind"
    short_name = "bc"
    min_version = 140000

    def __init__(self, i, tmd_dir):
        self.data_dir = tmd_dir + "/%snode%s" % (self.__class__.short_name, i)
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
                resp = self.getinfo()
                # Check that we're running at least the minimum version
                assert resp.json()['result']['version'] > self.__class__.min_version
                break  # break out of loop on success
            except requests.exceptions.ConnectionError as e:
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
    min_version = 130200
