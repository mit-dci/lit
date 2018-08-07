#!/usr/bin/env python3

import os
import sys
import time

import testlib

def run_test(env):
    bc = env.bitcoind
    lit = env.lits[0]

    # First figure out where we should send the money.
    addr = lit.make_new_addr()
    print('Got lit address:', addr)

    # Send a bitcoin.
    bc.rpc.sendtoaddress(addr, 1)
    env.generate_block(count=5)
    print('Sent and mined...')

    # Log it to make sure we got it.
    txo_total = lit.get_balance_info()['TxoTotal']
    print('lit balance:', txo_total)

    # Get an address for bitcoind.
    bcaddr = bc.rpc.getnewaddress()
    print('Got bitcoind address:', bcaddr)

    # Get the old balance.
    bal1 = bc.rpc.getbalance()

    # Send some bitcoin back, and mine it.
    lit.rpc.Send(DestAddrs=[ bcaddr ], Amts=[ 50000000 ])

    print(str(bc.rpc.getwalletinfo()))

    env.generate_block(count=5)
    print('Sent and mined again...')

    # Validate.
    bal2 = bc.rpc.getbalance()
    print('bitcoind balance:', bal1, '->', bal2, '(', bal2 - bal1, ')')
    if bal2 != bal1 + 0.5:
        raise AssertionError("Balance in bitcoind doesn't match what we think it should be!")

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(1)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
