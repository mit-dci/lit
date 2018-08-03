#!/usr/bin/env python3

import time

import testlib

print('starting bitcoind')
bc = testlib.BitcoinNode()
print('started!')

lits = []
for r in range(5):
    print('starting lit')
    l = testlib.LitNode(bc.rpc_port)
    print('started!')
    lits.append(l)

time.sleep(30)

bc.shutdown()
for l in lits:
    l.shutdown()
