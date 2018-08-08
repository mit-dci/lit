#!/usr/bin/env python3

import testlib
import testutil

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
    env.generate_block(count=5)

    # Now close the channel.
    print('Now breaking channel...')
    lit1.rpc.BreakChannel(ChanIdx=chan_id)

    print('// TODO Make BreakChannel return a status.')

    # Now we figure out the balances at 2 points in time.
    print(str(lit1.get_balance_info()))
    print('Fast-forwarding time...')
    env.generate_block(count=5) # Just to escape the locktime to make sure we get our money.
    bi2 = lit1.get_balance_info()
    print(str(bi2))

    print(str(lit1.rpc.ChannelList(ChanIdx=chan_id)['Channels']))
    assert bi2['ChanTotal'] == 0, "channel balance isn't zero!"
    # TODO Make sure the channel actually gets broken.

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
