#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Class representing a lit node

LitNode represents a lit node. It can be used to start/stop a lit node
and communicate with it over RPC."""
import logging
import os
import subprocess
import time

from litpy import litrpc

LIT_BIN = "%s/../lit" % os.path.abspath(os.path.dirname(__file__))

logger = logging.getLogger("TestFramework.litnode")
logger.propagate = False

class LitNode():
    """A class representing a Lit node"""
    index = 0

    def __init__(self, tmp_dir):
        self.index = LitNode.index
        LitNode.index += 1
        self.data_dir = tmp_dir + "/litnode%d" % self.index
        os.makedirs(self.data_dir)

        # Write a hexkey to the privkey file
        with open(self.data_dir + "/privkey.hex", 'w+') as f:
            f.write("1" * 63 + str(self.index) + "\n")

        self.args = ["--dir", self.data_dir]
        # disable auto-connect to testnet3 and litetest4
        self.args.extend(['--tn3', '', '--lt4', ''])

        self.rpc = None

    def start_node(self):
        logger.debug("Starting litnode %d with args %s" % (self.index, str(self.args)))
        assert os.path.isfile(LIT_BIN), "lit binary not found at %s" % LIT_BIN
        self.process = subprocess.Popen([LIT_BIN] + self.args, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        time.sleep(5)

    def stop_node(self):
        try:
            self.Stop()
        except AssertionError as e:
            if e == "lit node not running":
                logger.debug("node already stopped")
            else:
                raise

    def add_rpc_connection(self, ip, port):
        logger.debug("Opening rpc connection to litnode %d: %s:%s" % (self.index, ip, port))
        self.rpc = litrpc.LitConnection(ip, port)
        self.rpc.connect()

    def __getattr__(self, name):
        if self.rpc is not None:
            return self.rpc.__getattr__(name)
        else:
            raise AssertionError("lit node not running")

    def get_balance(self, coin_type):
        # convenience method for grabbing the node balance
        balances = self.rpc.Balance()['result']['Balances']
        for balance in balances:
            if balance['CoinType'] == coin_type:
                return balance
        raise AssertionError("No balance for coin_type %s" % coin_type)

    def get_height(self, coin_type):
        # convenience method for grabbing the sync height
        return self.get_balance(coin_type)['SyncHeight']
