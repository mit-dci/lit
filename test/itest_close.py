#!/usr/bin/env python3

import testlib

fee = 20
initialsend = 200000
capacity = 1000000

def run_test(env):
    bc = env.bitcoind
    lit1 = env.lits[0]
    lit2 = env.lits[1]

    # Connect the nodes.
    lit1.connect_to_peer(lit2)

    # First figure out where we should send the money.
    addr1 = lit1.make_new_addr()
    print('Got lit1 address:', addr1)

    # Send a bitcoin.
    bc.rpc.sendtoaddress(addr1, 1)
    env.generate_block()

    # Log it to make sure we got it.
    bal1 = lit1.get_balance_info()['TxoTotal']
    print('initial lit1 balance:', bal1)

    # Set the fee so we know what's going on.
    lit1.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)
    lit2.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)

    # Now actually do the funding.
    # TODO Abstract this call onto the LitNode object.
    res = lit1.rpc.FundChannel(
        Peer=lit1.get_peer_id(lit2),
        CoinType=testlib.REGTEST_COINTYPE,
        Capacity=capacity,
        InitialSend=initialsend,
        Data=None) # maybe use [0 for _ in range(32)] or something?

    chan_id = res['ChanIdx']
    print('Created channel ( ID', chan_id, ')')

    # Now we confirm the block.
    env.generate_block()

    # Now close the channel.
    print('Now closing...')
    res = lit1.rpc.CloseChannel(ChanIdx=chan_id)
    print('Status:', res['Status'])
    env.generate_block()

    # Check balances.
    bals = lit1.get_balance_info()
    fbal = bals['TxoTotal']
    print('final balance:', fbal)
    expected = bal1 - initialsend - 3560
    print('expected:', expected)
    print('diff:', expected - fbal)

    assert bals['ChanTotal'] == 0, "channel balance isn't zero!"

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
