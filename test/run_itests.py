#!/usr/bin/env python3

import testlib
import sys

noOfNodes = 2

from itest_connect import run_test as connect
from itest_receive import run_test as receive
from itest_send import run_test as send
from itest_send2 import run_test as send2
from itest_setgetfee import run_test as setgetfee
from itest_fund import run_test as fund
# from itest_close import run_test as close
# from itest_close_reverse import run_test as close_reverse
# from itest_break import run_test as break
# from itest_break_reverse import run_test as break_reverse
# from itest_push import run_test as push
# from itest_pushclose import run_test as pushclose
# from itest_pushclose_reverse import run_test as pushclose_reverse
# from itest_pushbreak import run_test as pushbreak
# from itest_pushbreak_reverse import run_test as pushbreak_reverse


if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(noOfNodes)
        try:
            connect(env)
        except Exception as e:
            print(e)
        try:
            receive(env)
        except Exception as e:
            print(e)
        try:
            send(env)
        except Exception as e:
            print(e)
        try:
            send2(env)
        except Exception as e:
            print(e)
        try:
            setgetfee(env)
        except Exception as e:
            print(e)
        try:
            fund(env)
        except Exception as e:
            print(e)
    finally:
        if env is not None:
            #env.shutdown()
            print("LENGTH of open lits", len(env.lits))
            #sys.exit(0)
