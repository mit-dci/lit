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
from itest_close import run_test as close
from itest_close_reverse import run_test as close_reverse
from itest_break import run_test as breaktest
from itest_break_reverse import run_test as break_reverse
from itest_push import run_test as push
from itest_pushclose import run_test as pushclose
from itest_pushclose_reverse import run_test as pushclose_reverse
from itest_pushbreak import run_test as pushbreak
from itest_pushbreak_reverse import run_test as pushbreak_reverse


if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(noOfNodes)
        print("Running Connect Integration Test")
        try:
            connect(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Receive Integration Test")
        try:
            receive(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Send Integration Test")
        try:
            send(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Send2 Integration Test")
        try:
            send2(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running SetGetFee Integration Test")
        try:
            setgetfee(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Fund Integration Test")
        try:
            fund(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Close Integration Test")
        try:
            close(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Close_reverse Integration Test")
        try:
            close_reverse(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Break_test Integration Test")
        try:
            breaktest(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Break_reverse Integration Test")
        try:
            break_reverse(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Push Integration Test")
        try:
            push(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Pushclose Integration Test")
        try:
            pushclose(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Pushclose_reverse Integration Test")
        try:
            pushclose_reverse(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Pushbreak Integration Test")
        try:
            pushbreak(env)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Running Pushbreak_reverse Integration Test")
        try:
            pushbreak_reverse(env)
        except Exception as e:
            print(e)
            sys.exit(1)

    finally:
        if env is not None:
            #env.shutdown()
            sys.exit(0)
