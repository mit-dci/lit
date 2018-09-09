#!/usr/bin/env python3

import testlib

create = 2

def run_test(env):
    length = len(env.lits)
    print("LENGTH IS", length)
    alice = env.lits[0]
    bob = env.lits[1]
    print('Connecting Alice', alice.lnid, 'to Bob', bob.lnid)
    alice.connect_to_peer(bob)
    print('Connected')
    alice.rpc.Say(Peer=alice.get_peer_id(bob), Message="hello!")
    print('Alice said hello to Bob.')
    bob.rpc.Say(Peer=bob.get_peer_id(alice), Message="world!")
    print('Bob said hello to Alice.')
    env.shutdown()
    # First figure out where we should send the money.
    addr = alice.make_new_addr()
    print('Got address:', addr)

    # Send a bitcoin.
    bc.rpc.sendtoaddress(addr, 1)
    env.generate_block()
    print('Sent and mined...')

    # Now verify that we got what we thought we would.
    txo_total = alice.get_balance_info()['TxoTotal']
    print('lit balance:', txo_total)
    if txo_total != 100000000:
        raise AssertionError("Didn't get the amount we thought we would!")

    # First figure out where we should send the money.
    addr = alice.make_new_addr()
    print('Got lit address:', addr)

    # Get the starting balance.
    bal0 = bc.rpc.getbalance()

    # Send a bitcoin.
    tx1 = bc.rpc.sendtoaddress(addr, 1)
    env.generate_block(count=1)
    print('Sent and mined... (tx: %s)' % tx1)

    # Log it to make sure we got it.
    txo_total = alice.get_balance_info()['TxoTotal']
    print('lit balance:', txo_total)

    # Get an address for bitcoind.
    bcaddr = bc.rpc.getnewaddress("wallet.dat", "bech32")
    print('Got bitcoind address:', bcaddr)

    # Get the old balance.
    bal1 = bc.rpc.getbalance()

    # Send some bitcoin back, and mine it.
    res = alice.rpc.Send(DestAddrs=[ bcaddr ], Amts=[ 50000000 ])
    if "_error" in res:
        raise AssertionError("Problem sending bitcoin back to bitcoind.")

    env.generate_block(count=1)
    print('Sent and mined again... (tx: %s)' % res['Txids'][0])

    # Validate.
    bal2 = bc.rpc.getbalance()
    print('bitcoind balance:', bal0, '->', bal1, '->', bal2, '(', bal2 - bal1, ')')
    if bal2 != bal1 + 12.5 + 0.5: # the 12.5 is because we mined a block
        raise AssertionError("Balance in bitcoind doesn't match what we think it should be!")


if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(create)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
