#!/usr/bin/env python3

import testlib

fee = 20
initialsend = 200000
capacity = 1000000

pushsend = 250000

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

    # Send the money through the channel.
    ct0lit1 = lit1.get_balance_info()['ChanTotal']
    ct0lit2 = lit2.get_balance_info()['ChanTotal']
    lit1.rpc.Push(ChanIdx=chan_id, Amt=pushsend, Data=None)
    ct1lit1 = lit1.get_balance_info()['ChanTotal']
    ct1lit2 = lit2.get_balance_info()['ChanTotal']
    assert ct1lit1 == ct0lit1 - pushsend, "channel balances don't match up"
    assert ct1lit2 == ct0lit2 + pushsend, "channel balances don't match up"

    # Close it, but Bob be the initiator.
    print('Closing channel... (with Bob)')
    tt0lit2 = lit2.get_balance_info()['TxoTotal']
    res = lit2.rpc.CloseChannel(ChanIdx=chan_id)
    print('Status:', res['Status'])
    print('Mining new block(s) to confirm closure...')
    env.generate_block(count=20)
    tt1lit2 = lit2.get_balance_info()['TxoTotal']

    # Now report the difference in channel balance.
    print('Bob:', tt0lit2, '->', tt1lit2, '( expected:', initialsend + pushsend - 20000, ')')
    assert tt1lit2 == tt0lit2 + initialsend + pushsend - 20000, "final balance doesn't match"

    # Idk where the 20000 gets removed from, fees probably but I'm not sure exactly where.

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
