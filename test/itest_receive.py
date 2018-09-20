import testlib

def run_test(env):
    bc = env.bitcoind
    lit = env.lits[0]

    # First figure out where we should send the money.
    addr = lit.make_new_addr()
    print('Got address:', addr)

    # Send a bitcoin.
    bc.rpc.sendtoaddress(addr, 1)
    env.generate_block()
    print('Sent and mined...')

    # Now verify that we got what we thought we would.
    txo_total = lit.get_balance_info()['TxoTotal']
    print('lit balance:', txo_total)
    if txo_total != 100000000:
        raise AssertionError("Didn't get the amount we thought we would!")
