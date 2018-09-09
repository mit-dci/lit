#!/usr/bin/env python3

import testlib

create = 1

def run_test(env):
    litcnt = len(env.lits)

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(create)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
