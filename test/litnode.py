#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Class representing a lit node

LitNode represents a lit node. It can be used to start/stop a lit node
and communicate with it over RPC."""
import os
import subprocess

from litpy import litrpc

LIT_BIN = "%s/../lit" % os.path.abspath(os.path.dirname(__file__))

class LitNode():
    """A class representing a Lit node"""
    def __init__(self, i, tmp_dir):
        self.data_dir = tmp_dir + "/litnode%s" % i
        os.makedirs(self.data_dir)

        # Write a hexkey to the privkey file
        with open(self.data_dir + "/privkey.hex", 'w+') as f:
            f.write("1" * 63 + str(i) + "\n")

        self.args = ["-dir", self.data_dir]
        #disable auto-connect to testnet3 and litetest4
        self.args.extend(['-tn3', '', '-lt4', ''])
		
    def start_node(self):
        self.process = subprocess.Popen([LIT_BIN] + self.args, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    def add_rpc_connection(self, ip, port):
        self.rpc = litrpc.LitConnection(ip, port)
        self.rpc.connect()

    def __getattr__(self, name):
        return self.rpc.__getattr__(name)

    def get_balance(self, coin_type):
        # convenience method for grabbing the node balance
        balances = self.rpc.Balance()['result']['Balances']
        for balance in balances:
            if balance['CoinType'] == coin_type:
                return balance['TxoTotal']
        raise AssertionError("No balance for coin_type %s" % coin_type)
