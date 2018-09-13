#!/usr/bin/env python3

import testlib
import test_combinators

fee = 20
initialsend = 200000
capacity = 1000000

def run_test(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]
    try:
        test_combinators.run_break_test(env, lit1, lit2, lit1)
    except Exception as e:
        print(e)

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
