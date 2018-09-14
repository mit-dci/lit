#!/usr/bin/env python3

import testlib

fee = 20

def run_test(env):
    bc = env.bitcoind
    lit = env.lits[0]

    # Report the initial fee.
    got = lit.rpc.GetFee(CoinType=testlib.REGTEST_COINTYPE)['CurrentFee']
    print('Starting fee is', got, '(per byte)')

    # Set the fee.
    print('Setting fee to', fee)
    lit.rpc.SetFee(Fee=fee, CoinType=testlib.REGTEST_COINTYPE)
    got = lit.rpc.GetFee(CoinType=testlib.REGTEST_COINTYPE)['CurrentFee']
    print('Checked fee, got', fee)

    assert got == fee, "Set fee and returned fee don't match."
    print('OK')

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(1)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
