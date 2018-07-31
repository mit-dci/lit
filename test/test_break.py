#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Test basic lit functionality

- start coin node
- start two lit nodes
- connect over websocket
- create new address
- get balance
- listen on lit node 0
- connect from lit node 1 to lit node 0
- send funds from coin node to lit node 0 address
- open channel between lit node 0 and lit node 1
- push funds from lit node 0 to lit node 1
- push funds back
- break channel
- stop"""
from lit_test_framework import wait_until
from test_basic import TestBasic

class TestBreak(TestBasic):
    def run_test(self):
        self._ready_coinnode()
        self._ready_litnodes()
        self._ready_litnode_for_channel()
        self._open_channel()
        self._push_funds_through_channel()
        self._break_channel()

    def _break_channel(self):
        self.log.info("Break channel")
        self.litnodes[0].BreakChannel(ChanIdx=1)
        self.confirm_transactions(self.coinnodes[0], self.litnodes[0], 1)

        # Make sure balances are as expected
        self.log.info("Make sure balances match")
        wait_until(lambda: abs(self.litnodes[1].get_balance(self.coins[0]['code'])['TxoTotal'] - 50200000) < self.coins[0]["feerate"] * 2000)
        litnode1_balance = self.litnodes[1].get_balance(self.coins[0]['code'])
        assert litnode1_balance['TxoTotal'] == litnode1_balance['MatureWitty']
        litnode0_balance = self.litnodes[0].get_balance(self.coins[0]['code'])
        assert abs(self.balance + 949800000 - litnode0_balance['TxoTotal']) < self.coins[0]["feerate"] * 2000

        self.log.info("Verify that channel breaker cannot spend funds immediately")
        assert abs(litnode0_balance['TxoTotal'] - litnode0_balance['MatureWitty'] - 949800000) < self.coins[0]["feerate"] * 2000

        self.log_balances(self.coins[0]['code'])

        self.log.info("Advance chain 5 blocks. Verify that channel breaker can now spend funds")
        self.coinnodes[0].generate(5)
        self.chain_height += 5
        wait_until(lambda: self.litnodes[0].get_balance(self.coins[0]['code'])["SyncHeight"] == self.chain_height)
        litnode0_balance = self.litnodes[0].get_balance(self.coins[0]['code'])
        assert litnode0_balance['TxoTotal'] == litnode0_balance['MatureWitty']

        self.log_balances(self.coins[0]['code'])

if __name__ == "__main__":
    exit(TestBreak().main())
