#!/usr/bin/env python3

import testlib
import sys

noOfNodes = 2

from itest_testlib import run_test as test_lib
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

tests = [
    {'func':test_lib, 'name': 'Test test_lib', 'nodes': 1}, 
    {'func':connect, 'name': 'Test connect', 'nodes': 2},
    {'func':receive, 'name': 'Test receive', 'nodes': 2}, 
    {'func':send, 'name': 'Test send', 'nodes': 2}, 
    {'func':send2, 'name': 'Test send2', 'nodes': 2}, 
    {'func':setgetfee, 'name': 'Test setgetfee', 'nodes': 2}, 
    {'func':fund, 'name': 'Test fund', 'nodes': 2},
    {'func':close, 'name': 'Test close', 'nodes': 2},
    {'func':close_reverse, 'name': 'Test close_reverse', 'nodes': 2},
    {'func':breaktest, 'name': 'Test breaktest', 'nodes': 2},
    {'func':break_reverse, 'name': 'Test break_reverse', 'nodes': 2},
    {'func':push, 'name': 'Test push', 'nodes': 2},
    {'func':pushclose, 'name': 'Test pushclose', 'nodes': 2},
    {'func':pushclose_reverse, 'name': 'Test pushclose_reverse', 'nodes': 2},
    {'func':pushbreak, 'name': 'Test pushbreak', 'nodes': 2},
    {'func':pushbreak_reverse, 'name': 'Test pushbreak_reverse', 'nodes': 2}
]



for test in tests:
    error = False
    try:
        print("==============================")
        print("Running test " + test['name'])
        print("==============================")
        env = testlib.TestEnv(test['nodes'])
        test['func'](env)
        print("-------")
        print("SUCCESS")
        print("-------")
    except Exception as e:
        print("-------")
        print("FAILURE:")
        print(e)
        print("-------")
        error = True
    
    env.shutdown()
    
    if error:
        sys.exit(1)

