#!/usr/bin/env python3

import testlib
import testlib_combinatoric_tests

fee = 20
initialsend = 200000
capacity = 1000000

pushsend = 250000

def run_test(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]
    testlib_combinatoric_tests.run_pushclose_test(env, lit1, lit2, lit2)

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
