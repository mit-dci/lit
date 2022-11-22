#!/usr/bin/env python3

import os
import sys
import time
import importlib.util as imu

import testlib

# Loads a python module from a path, giving it a particular name.
def __load_mod_from_file(path, name):
    spec = imu.spec_from_file_location(name, path)
    mod = imu.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod

# Parses a line like this:
#
#   foobar 3 run_test_foo
#
# (The last argument is optional.)
def __parse_test_def(line):
    parts = line.split(' ')
    if len(parts) != 2 and len(parts) != 3:
        return None
    name = parts[0]
    nodes = parts[1]
    func = 'run_test'
    if len(parts) == 3:
        func = parts[2]
    return {
        'test_name': name.strip(),
        'func_name': func.strip(),
        'node_cnt': int(nodes.strip()), # Raises if fails.
    }

def test_name_from_filename(name):
    if name.startswith('itest_') and name.endswith('.py'):
        return name[6:-3] # Should just cut off the ends.
    else:
        raise ValueError('')

def parse_tests_file(path):
    tdefs = []
    with open(path, 'r') as f:
        for l in f.readlines():
            if len(l) == 0 or l.startswith('#'):
                continue
            t = __parse_test_def(l)
            if t is not None:
                tdefs.append(t)
    return tdefs

def load_tests_from_file(path):
    tests = []
    pdir = os.path.dirname(path)
    mods = {}
    for t in parse_tests_file(path):
        tname = t['test_name']
        mod = None
        if tname in mods:
            mod = mods[tname]
        else:
            fname = 'itest_%s.py' % tname
            modname = 'testmod_' + tname
            mod = __load_mod_from_file(fname, modname)
            mods[tname] = mod
        pretty = tname
        tfn = getattr(mod, t['func_name'])
        if tfn.__name__ != 'run_test':
            pretty += ':' + tfn.__name__
        tests.append({
            'name': tname,
            'pretty_name': pretty,
            'test_func': tfn,
            'node_cnt': t['node_cnt'],
        })
    return tests

def run_test_list(tests):
    ok = 0
    failed = []

    # First, just run the tests.
    for t in tests:
        pname = t['pretty_name']

        print('==============================')
        tfunc = t['test_func']
        print('Running test:', pname)

        # Do this before the bottom frame so we have a clue how long startup
        # took and where the fail was.
        env = None
        try:
            testlib.clean_data_dir() # IMPORTANT!
            print('------------------------------')
            env = testlib.TestEnv(t['node_cnt'])
        except Exception as e:
            print('Error initing env, this is a test framework bug:', e)
            break
        print('==============================')

        # This is where the test actually runs.
        try:
            tfunc(env)
            env.shutdown()
            print('------------------------------')
            print('Success:', pname)
            ok += 1
            time.sleep(0.1) # Wait for things to exit, just to be sure.
        except BaseException as e:
            env.shutdown()
            print('------------------------------')
            print('Failure:', pname)
            print('\nError:', e)
            failed.append(t)
            if type(e) is KeyboardInterrupt:
                break
            # TODO Report failures and why.

    print('==============================')

    # Collect results.
    res = {
        'ok': ok,
        'fail': len(failed),
        'ignored': len(tests) - ok - len(failed),
        'failed': failed
    }
    return res

if __name__ == '__main__':
    os.makedirs('_data', exist_ok=True)
    tests = load_tests_from_file('tests.txt')

    # If given arguments, run these instead.  Doesn't do them in given order, sadly.
    if len(sys.argv) > 1:
        to_run = []
        for t in tests:
            if t['name'] in sys.argv[1:]:
                to_run.append(t)
        tests = to_run

    res = run_test_list(tests)
    print('Success:', res['ok'])
    print('Failure:', res['fail'])
    print('Ignored:', res['ignored'])
    if res['fail'] > 0:
        print('\nFailures:')
        for f in res['failed']:
            print(' - %s' % f['pretty_name'])
        sys.exit(1)
    else:
        sys.exit(0)
