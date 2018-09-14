#!/usr/bin/env python3

import testlib
import test_combinators

def run_test(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]
    test_combinators.run_pushclose_test(env, lit1, lit2, lit1)

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
