#!/usr/bin/env python3

#test testlib 1

import testlib

create = 1

def run_test(env):
    litcnt = len(env.lits)
    print('Found', len(env.lits), 'Lit nodes created.')
    if litcnt == create:
        print('OK')
    else:
        print('ERR')

if __name__ == '__main__':
    env = None
    try:
        env = testlib.TestEnv(create)
        run_test(env)
    finally:
        if env is not None:
            env.shutdown()
