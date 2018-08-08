#!/usr/bin/env python3

import testlib
import testlib_combinatoric_tests

def run_test(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]
    testlib_combinatoric_tests.run_pushclose_test(env, lit1, lit2, lit1)

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
