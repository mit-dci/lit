import testlib

def run_test(env):
    bc = env.bitcoind
    lit = env.lits[0]

    # First figure out where we should send the money.
    addr = lit.make_new_addr()
    print('Got lit address:', addr)

    # Get the starting balance.
    bal0 = bc.rpc.getbalance()

    # Send a bitcoin.
    tx1 = bc.rpc.sendtoaddress(addr, 1)
    env.generate_block(count=1)
    print('Sent and mined... (tx: %s)' % tx1)

    # Log it to make sure we got it.
    txo_total = lit.get_balance_info()['TxoTotal']
    print('lit balance:', txo_total)

    # Get an address for bitcoind.
    bcaddr = bc.rpc.getnewaddress("wallet.dat", "bech32")
    print('Got bitcoind address:', bcaddr)

    # Get the old balance.
    bal1 = bc.rpc.getbalance()

    # Send some bitcoin back, and mine it.
    res = lit.rpc.Send(DestAddrs=[ bcaddr ], Amts=[ 50000000 ])
    if "_error" in res:
        raise AssertionError("Problem sending bitcoin back to bitcoind.")

    env.generate_block(count=1)
    print('Sent and mined again... (tx: %s)' % res['Txids'][0])

    # Validate.
    bal2 = bc.rpc.getbalance()
    print('bitcoind balance:', bal0, '->', bal1, '->', bal2, '(', bal2 - bal1, ')')
    if float(bal2) != float(bal1) + 12.5 + 0.5: # the 12.5 is because we mined a block
        raise AssertionError("Balance in bitcoind doesn't match what we think it should be!")
