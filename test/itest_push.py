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

    # Figure out if it's actually open now.
    res = lit1.rpc.ChannelList(ChanIdx=chan_id)
    cinfo = res['Channels'][0]
    assert cinfo['Height'] == env.get_height(), "Channel height doesn't match new block."

    # Now update the balances back and forth a bit to make sure it all works.
    ct0 = lit1.get_balance_info()['ChanTotal']
    lit1.rpc.Push(ChanIdx=chan_id, Amt=1000, Data=None)
    ct1 = lit1.get_balance_info()['ChanTotal']
    assert ct1 == ct0 - 1000, "channel update didn't work properly"
    lit1.rpc.Push(ChanIdx=chan_id, Amt=10000, Data=None)
    ct2 = lit1.get_balance_info()['ChanTotal']
    assert ct2 == ct1 - 10000, "channel update didn't work properly"
    lit2.rpc.Push(ChanIdx=chan_id, Amt=5000, Data=None)
    ct3 = lit1.get_balance_info()['ChanTotal']
    assert ct3 == ct2 + 5000, "channel update didn't work properly"
    lit1.rpc.Push(ChanIdx=chan_id, Amt=250, Data=None)
    ct4 = lit1.get_balance_info()['ChanTotal']
    assert ct4 == ct3 - 250, "channel update didn't work properly"

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
