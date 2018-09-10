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

    # Figure out if it's actually open now.
    res = lit1.rpc.ChannelList(ChanIdx=cid)
    cinfo = res['Channels'][0]
    assert cinfo['Height'] == env.get_height(), "Channel height doesn't match new block."

    # Now update the balances back and forth a bit to make sure it all works.
    ct0 = lit1.get_balance_info()['ChanTotal']
    lit1.rpc.Push(ChanIdx=cid, Amt=1000, Data=None)
    ct1 = lit1.get_balance_info()['ChanTotal']
    assert ct1 == ct0 - 1000, "channel update didn't work properly"
    lit1.rpc.Push(ChanIdx=cid, Amt=10000, Data=None)
    ct2 = lit1.get_balance_info()['ChanTotal']
    assert ct2 == ct1 - 10000, "channel update didn't work properly"
    lit2.rpc.Push(ChanIdx=cid, Amt=5000, Data=None)
    ct3 = lit1.get_balance_info()['ChanTotal']
    assert ct3 == ct2 + 5000, "channel update didn't work properly"
    lit1.rpc.Push(ChanIdx=cid, Amt=250, Data=None)
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
