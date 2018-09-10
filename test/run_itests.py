#!/usr/bin/env python3

import testlib
import sys

noOfNodes = 2

from itest_connect import run_test as connect
from itest_receive import run_test as receive

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(noOfNodes)
        try:
            connect(env)
        except e:
            print(e)
        try:
            receive(env)
        except e:
            print(e)
    finally:
        if env is not None:
            #env.shutdown()
            print("LENGTH of open lits", len(env.lits))
            #sys.exit(0)
