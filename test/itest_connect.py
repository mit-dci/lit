#!/usr/bin/env python3

import testlib

def run_test(env):
    alice = env.lits[0]
    bob = env.lits[1]
    print('Connecting Alice', alice.lnid, 'to Bob', bob.lnid)
    alice.connect_to_peer(bob)
    print('Connected')
    alice.rpc.Say(Peer=alice.get_peer_id(bob), Message="hello!")
    print('Alice said hello to Bob.')
    bob.rpc.Say(Peer=bob.get_peer_id(alice), Message="world!")
    print('Bob said hello to Alice.')

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(2)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
