#!/usr/bin/env python3

import testlib
import sys

noOfNodes = 2

from itest_connect import run_test as connect()

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(noOfNodes)
        connect(env)
    finally:
        if env is not None:
            #env.shutdown()
            print("LENGTH of open lits", len(env.lits))
            #sys.exit(0)
