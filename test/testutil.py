#!/usr/bin/env python3
# Copyright (c) 2018 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Utils for lit testing"""
import time

def assert_equal(thing1, thing2, *args):
    if thing1 != thing2 or any(thing1 != arg for arg in args):
        raise AssertionError("not(%s)" % " == ".join(str(arg) for arg in (thing1, thing2) + args))

def wait_until(predicate, *, attempts=120, dt=0.25, errmsg=None): # up to 30 seconds
    attempt = 0

    while attempt < attempts:
        if predicate():
            return True
        attempt += 1
        time.sleep(dt)

    if errmsg is not None:
        raise AssertionError(errmsg)
    else:
        raise AssertionError("wait_until() timed out")
