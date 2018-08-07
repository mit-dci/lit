#!/usr/bin/env python3

import os
import sys

import testlib

def run_test(env):
    bc = env.bitcoind
    lit = env.lits[0]
    lit2 = env.lits[1]

    # First figure out where we should send the money.
    addr = lit.make_new_addr()
    print('Got lit1 address:', addr)

    # Send a bitcoin.
    bc.rpc.sendtoaddress(addr, 1)
    env.generate_block()
    print('Sent and mined...')

    # Log it to make sure we got it.
    txo_total = lit.get_balance_info()['TxoTotal']
    print('initial lit1 balance:', txo_total)

    # Get an address for bitcoind.
    addr2 = lit2.make_new_addr()
    print('Got lit2 address:', addr2)

    # Send some bitcoin back, and mine it.
    lit.rpc.Send(DestAddrs=[ addr2 ], Amts=[ 50000000 ])
    env.generate_block()
    print('Sent and mined again...')

    # Validate.
    fbal1 = lit.get_balance_info()['TxoTotal']
    print('lit2 balance:', fbal1)
    fbal2 = lit2.get_balance_info()['TxoTotal']
    print('lit2 balance:', fbal2)
    if fbal2 != 50000000:
        raise AssertionError("Didn't get the amount we expected in the final lit node.")

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
