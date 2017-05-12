#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Test basic lit functionality

- start bitcoind process
- start two lit processes
- connect over websocket
- create new address
- get balance
- listen on litnode0
- connect from litnode1 to litnode0
- send funds from bitcoind process to litnode0 address
- open channel between litnode0 and litnode1
- push funds from litnode0 to litnode1
- push funds back
- close channel co-operatively
- stop"""
import time

from lit_test_framework import LitTest, wait_until

BC_REGTEST = 257

class TestBasic(LitTest):
    def run_test(self):

        # Start a bitcoind node
        self.add_bcnode()
        self.bcnodes[0].start_node()

        self.log.info("Generate 500 blocks to activate segwit")
        self.bcnodes[0].generate(nblocks=500)
        network_info = self.bcnodes[0].getblockchaininfo().json()['result']
        assert network_info['bip9_softforks']['segwit']['status'] == 'active'

        # Start lit node 0 and open websocket connection
        self.add_litnode()
        self.litnodes[0].args.extend(["-reg", "127.0.0.1"])
        self.litnodes[0].start_node()
        self.litnodes[0].add_rpc_connection("127.0.0.1", "8001")

        # Start lit node 1 and open websocket connection
        self.add_litnode()
        self.litnodes[1].args.extend(["-rpcport", "8002", "-reg", "127.0.0.1"])
        self.litnodes[1].start_node()
        self.litnodes[1].add_rpc_connection("127.0.0.1", "8002")

        self.log.info("Connect lit nodes")
        res = self.litnodes[0].Listen(Port="127.0.0.1:10001")["result"]
        self.litnodes[0].lit_address = res["Adr"] + '@' + res["LisIpPorts"][0]

        res = self.litnodes[1].Connect(LNAddr=self.litnodes[0].lit_address)
        assert not res['error']
        time.sleep(1)

        # Check that litnode0 and litnode1 are connected
        assert len(self.litnodes[0].ListConnections()['result']['Connections']) == 1
        assert len(self.litnodes[1].ListConnections()['result']['Connections']) == 1
        self.log.info("lit nodes connected")

        self.log.info("Send funds from bitcoind node to litnode0")
        balance = self.litnodes[0].get_balance(BC_REGTEST)
        self.log_balances(BC_REGTEST)
        addr = self.litnodes[0].rpc.new_address()
        self.bcnodes[0].sendtoaddress(address=addr["result"]["LegacyAddresses"][0], amount=12.34)
        self.bcnodes[0].generate(nblocks=1).text

        self.log.info("Waiting to receive transaction")

        # Wait for transaction to be received (15 seconds timeout)
        wait_until(lambda: self.litnodes[0].get_balance(BC_REGTEST) - balance == 1234000000)
        balance = self.litnodes[0].get_balance(BC_REGTEST)
        self.log.info("Funds received")
        self.log_balances(BC_REGTEST)

        self.log.info("Send money from litnode0 to its own segwit address and confirm")
        self.log.info("sending 1000000000 satoshis to litnode1 address")
        self.bcnodes[0].generate(nblocks=1)
        self.litnodes[0].Send(DestAddrs=[self.litnodes[0].Address()['result']['WitAddresses'][0]], Amts=[1000000000])
        self.bcnodes[0].generate(nblocks=1)
        wait_until(lambda: self.litnodes[0].Balance()['result']["Balances"][0]["SyncHeight"] == 503)

        # We'll lose some money to fees. Hopefully not too much!
        assert balance - self.litnodes[0].get_balance(BC_REGTEST) < 50000
        balance = self.litnodes[0].get_balance(BC_REGTEST)
        self.log.info("Funds transferred to segwit address")
        self.log_balances(BC_REGTEST)

        self.log.info("Open channel from litnode0 to litnode1")
        assert self.litnodes[0].ChannelList()['result']['Channels'] == []
        assert self.litnodes[1].ChannelList()['result']['Channels'] == []

        self.litnodes[0].FundChannel(Peer=1, CoinType=257, Capacity=1000000000)
        self.log.info("litnode0 has funded channel")

        # Wait for channel to open
        wait_until(lambda: len(self.litnodes[0].ChannelList()['result']['Channels']) > 0)
        assert len(self.litnodes[1].ChannelList()['result']['Channels']) > 0
        self.log.info("Channel open")

        assert abs(balance - self.litnodes[0].get_balance(BC_REGTEST) - 1000000000) < 50000
        balance = self.litnodes[0].get_balance(BC_REGTEST)
        self.log_balances(BC_REGTEST)

        litnode0_channel = self.litnodes[0].ChannelList()['result']['Channels'][0]
        litnode1_channel = self.litnodes[1].ChannelList()['result']['Channels'][0]

        assert litnode0_channel['Capacity'] == 1000000000
        assert litnode0_channel['StateNum'] == 0
        assert not litnode0_channel['Closed']
        assert litnode0_channel['MyBalance'] == 1000000000

        assert litnode1_channel['Capacity'] == 1000000000
        assert litnode1_channel['StateNum'] == 0
        assert not litnode1_channel['Closed']
        assert litnode1_channel['MyBalance'] == 0

        self.log_channel_balance(self.litnodes[0], 0, self.litnodes[1], 0)
        self.log.info("Now push some funds from litnode0 to litnode1")

        self.litnodes[0].Push(ChanIdx=1, Amt=100000000)

        litnode0_channel = self.litnodes[0].ChannelList()['result']['Channels'][0]
        litnode1_channel = self.litnodes[1].ChannelList()['result']['Channels'][0]
        assert litnode0_channel['MyBalance'] == 900000000
        assert litnode1_channel['MyBalance'] == 100000000

        self.log_channel_balance(self.litnodes[0], 0, self.litnodes[1], 0)

        self.log.info("Push some funds back")
        self.litnodes[1].Push(ChanIdx=1, Amt=50000000)

        litnode0_channel = self.litnodes[0].ChannelList()['result']['Channels'][0]
        litnode1_channel = self.litnodes[1].ChannelList()['result']['Channels'][0]
        assert litnode0_channel['MyBalance'] == 950000000
        assert litnode1_channel['MyBalance'] == 50000000

        self.log_channel_balance(self.litnodes[0], 0, self.litnodes[1], 0)
        self.log.info("Close channel")
        self.litnodes[0].CloseChannel(ChanIdx=1)
        self.bcnodes[0].generate(nblocks=10)
        wait_until(lambda: self.litnodes[0].Balance()['result']["Balances"][0]["SyncHeight"] == 513)

        assert abs(self.litnodes[1].get_balance(BC_REGTEST) - 50000000) < 50000
        assert abs(balance + 950000000 - self.litnodes[0].get_balance(BC_REGTEST)) < 50000

        self.log_balances(BC_REGTEST)

if __name__ == "__main__":
    exit(TestBasic().main())
