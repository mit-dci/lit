#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Utils for lit testing"""
import time

def assert_equal(thing1, thing2, *args):
    if thing1 != thing2 or any(thing1 != arg for arg in args):
        raise AssertionError("not(%s)" % " == ".join(str(arg) for arg in (thing1, thing2) + args))

def wait_until(predicate, *, attempts=float('inf'), timeout=float('inf')):
    if attempts == float('inf') and timeout == float('inf'):
        timeout = 60
    attempt = 0
    elapsed = 0
    print("Attempt: ", attempt, "elapsed", elapsed)

    while attempt < attempts and elapsed < timeout:
        print("Attempts: ", attempt, "elapsed", elapsed)
        if predicate():
            return True
        attempt += 1
        elapsed += 0.25
        time.sleep(0.25)

    raise AssertionError("wait_until() timed out")
