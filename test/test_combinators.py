
import testlib

fee = 20

initialsend = 200000
capacity = 1000000

pushsend = 250000

def run_pushclose_test(env, initiator, target, closer):
    bc = env.bitcoind
    initiator = env.lits[0]
    target = env.lits[1]

    # Connect the nodes.
    initiator.connect_to_peer(target)

    # First figure out where we should send the money.
    addr1 = initiator.make_new_addr()
    print('Got initiator address:', addr1)

    # Send a bitcoin.
    bc.rpc.sendtoaddress(addr1, 1)
    env.generate_block()

    # Log it to make sure we got it.
    bal1 = initiator.get_balance_info()['TxoTotal']
    print('initial initiator balance:', bal1)

    # Set the fee so we know what's going on.
    initiator.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)
    target.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)
    print('fees set to', fee, '(per byte)')

    # Now actually do the funding.
    cid = initiator.open_channel(target, capacity, initialsend)
    print('Created channel:', cid)

    # Now we confirm the block.
    env.generate_block()

    # Figure out if it's actually open now.
    res = initiator.rpc.ChannelList(ChanIdx=cid)
    cinfo = res['Channels'][0]
    assert cinfo['Height'] == env.get_height(), "Channel height doesn't match new block."

    # Send the money through the channel.
    ct0initiator = initiator.get_balance_info()['ChanTotal']
    ct0target = target.get_balance_info()['ChanTotal']
    initiator.rpc.Push(ChanIdx=cid, Amt=pushsend, Data=None)
    ct1initiator = initiator.get_balance_info()['ChanTotal']
    ct1target = target.get_balance_info()['ChanTotal']
    assert ct1initiator == ct0initiator - pushsend, "channel balances don't match up"
    assert ct1target == ct0target + pushsend, "channel balances don't match up"

    # Close it, but Alice be the initiator.
    print('Closing channel... (with Alice)')
    tt0 = target.get_balance_info()['TxoTotal']
    res = closer.rpc.CloseChannel(ChanIdx=cid)
    print('Status:', res['Status'])
    print('Mining new block(s) to confirm closure...')
    env.generate_block(count=20)
    tt1 = target.get_balance_info()['TxoTotal']

    # Now report the difference in channel balance.
    # 200 is the fee amount, wish we could have some sort of constatns in the tests for that
    print('Target:', tt0, '->', tt1, '( expected:', initialsend + pushsend - 200, ')')
    assert tt1 -tt0 == initialsend + pushsend - 200, "final balance doesn't match"

def run_pushbreak_test(env, initiator, target, breaker):
    bc = env.bitcoind

    # Connect the nodes.
    initiator.connect_to_peer(target)

    # First figure out where we should send the money.
    addr1 = initiator.make_new_addr()
    print('Got initiator address:', addr1)

    # Send a bitcoin.
    bc.rpc.sendtoaddress(addr1, 1)
    env.generate_block()

    # Log it to make sure we got it.
    bal1 = initiator.get_balance_info()['TxoTotal']
    print('initial initiator balance:', bal1)

    # Set the fee so we know what's going on.
    initiator.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)
    target.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)
    print('fees set to', fee, '(per byte)')

    # Now actually do the funding.
    cid = initiator.open_channel(target, capacity, initialsend)
    print('Created channel:', cid)

    # Now we confirm the block.
    env.generate_block()

    # Figure out if it's actually open now.
    res = initiator.rpc.ChannelList(ChanIdx=cid)
    cinfo = res['Channels'][0]
    assert cinfo['Height'] == env.get_height(), "Channel height doesn't match new block."

    # Send the money through the channel.
    ct0initiator = initiator.get_balance_info()['ChanTotal']
    ct0target = target.get_balance_info()['ChanTotal']
    initiator.rpc.Push(ChanIdx=cid, Amt=pushsend, Data=None)
    ct1initiator = initiator.get_balance_info()['ChanTotal']
    ct1target = target.get_balance_info()['ChanTotal']
    assert ct1initiator == ct0initiator - pushsend, "channel balances don't match up"
    assert ct1target == ct0target + pushsend, "channel balances don't match up"

    # Close it, but Bob be the initiator.
    print('Breaking channel... (with Bob)')
    tt0 = target.get_balance_info()['TxoTotal']
    res = breaker.rpc.BreakChannel(ChanIdx=cid)
    print('Status:', str(res))
    print('Mining new block(s) to confirm closure...')
    env.generate_block(count=20)
    tt1 = target.get_balance_info()['TxoTotal']

    # Now report the difference in channel balance.
    print('Target:', tt0, '->', tt1, '( expected:', initialsend + pushsend - 20000, ')')
    assert tt1 == tt0 + initialsend + pushsend - 20000, "final balance doesn't match"

    # Idk where the 20000 gets removed from, fees probably but I'm not sure exactly where.

def run_close_test(env, initiator, target, closer):
    bc = env.bitcoind

    # Connect the nodes.
    initiator.connect_to_peer(target)

    # First figure out where we should send the money.
    addr1 = initiator.make_new_addr()
    print('Got initiator address:', addr1)

    # Send a bitcoin.
    bc.rpc.sendtoaddress(addr1, 1)
    env.generate_block()

    # Log it to make sure we got it.
    bal1 = initiator.get_balance_info()['TxoTotal']
    print('initial initiator balance:', bal1)

    # Set the fee so we know what's going on.
    initiator.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)
    target.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)

    # Now actually do the funding.
    cid = initiator.open_channel(target, capacity, initialsend)
    print('Created channel:', cid)

    # Now we confirm the block.
    env.generate_block()

    # Now close the channel.
    print('Now closing...')
    res = closer.rpc.CloseChannel(ChanIdx=cid)
    print('Status:', res['Status'])
    env.generate_block()

    # Check balances.
    bals = initiator.get_balance_info()
    fbal = bals['TxoTotal']
    print('final balance:', fbal)
    expected = bal1 - initialsend - 3560
    print('expected:', expected)
    print('diff:', expected - fbal)

    assert bals['ChanTotal'] == 0, "channel balance isn't zero!"

def run_break_test(env, initiator, target, breaker):
    bc = env.bitcoind

    # Connect the nodes.
    initiator.connect_to_peer(target)

    # First figure out where we should send the money.
    addr1 = initiator.make_new_addr()
    print('Got initiator address:', addr1)

    # Send a bitcoin.
    bc.rpc.sendtoaddress(addr1, 1)
    env.generate_block()

    # Log it to make sure we got it.
    bal1 = initiator.get_balance_info()['TxoTotal']
    print('initial initiator balance:', bal1)

    # Set the fee so we know what's going on.
    initiator.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)
    target.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)

    # Now actually do the funding.
    cid = initiator.open_channel(target, capacity, initialsend)
    print('Created channel:', cid)

    # Now we confirm the block.
    env.generate_block()

    # Now we confirm the block.
    env.generate_block(count=5)

    # Now close the channel.
    print('Now breaking channel...')
    res = breaker.rpc.BreakChannel(ChanIdx=cid)
    print('Status:', str(res))

    # Now we figure out the balances at 2 points in time.
    print(str(initiator.get_balance_info()))
    print('Fast-forwarding time...')
    env.generate_block(count=5) # Just to escape the locktime to make sure we get our money.
    bi2 = initiator.get_balance_info()
    print(str(bi2))

    print(str(initiator.rpc.ChannelList(ChanIdx=cid)['Channels']))
    assert bi2['ChanTotal'] == 0, "channel balance isn't zero!"
    # TODO Make sure the channel actually gets broken.
