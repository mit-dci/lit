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
    print('fees set to', fee, '(per byte)')

    # Now actually do the funding.
    cid = lit1.open_channel(lit2, capacity, initialsend)
    print('Created channel:', cid)

    # Now we confirm the block.
    env.generate_block()
    print('Mined new block to confirm channel')

    # Figure out if it's actually open now.
    res = lit1.rpc.ChannelList(ChanIdx=cid)
    cinfo = res['Channels'][0]
    assert cinfo['Height'] == env.get_height(), "Channel height doesn't match new block."

    # Make sure balances make sense
    bals2 = lit1.get_balance_info()
    print('new lit1 balance:', bals2['TxoTotal'], 'in txos,', bals2['ChanTotal'], 'in chans')
    bal2sum = bals2['TxoTotal'] + bals2['ChanTotal']
    print('  = sum ', bal2sum)
    print(' -> diff', bal1 - bal2sum)
    print(' -> fee ', bal1 - bal2sum - initialsend)

    assert bals2['ChanTotal'] > 0, "channel balance isn't nonzero!"

    test_combinators.run_close_test(env, lit1, lit2, lit1)

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
