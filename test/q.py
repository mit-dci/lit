#!/usr/bin/env python3

import time

import testlib

print("starting env")
env = testlib.TestEnv(10)
print("started!")

time.sleep(5)

print("shutting down")
env.shutdown()
print("finished")
