#!/usr/bin/env python3

import os
import sys
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
    if len(parts) != 2 and len(parts) != 2:
        return None
    name = parts[0]
    nodes = parts[1]
    func = 'run_test'
    if len(parts) == 3:
        func = parts[2]
    return {
        'test_name': name,
        'func_name': func,
        'node_cnt': int(nodes), # Raises if fails.
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
            mod = mods[modname]
        else:
            fname = 'itest_%s.py' % tname
            modname = 'testmod_' + tname
            mod = __load_mod_from_file(fname, modname)
            mods[tname] = mod
        tests.append({
            'name': tname,
            'test_func': getattr(mod, t['func_name']),
            'node_cnt': t['node_cnt']
        })
    return tests

# Returns the list of tests in a file.
def parse_test_file(path):
    base = os.path.splitext(path)[0]
    founddecls = []
    with open(path, 'r') as f:
        for l in f.readlines():
            if l.startswith(TEST_DECL_PREFIX):
                d = __parse_test_def(l)
                if d is not None:
                    founddecls.append(d)
                else:
                    raise ValueError('invalid test declaration: ' + l)
    return founddecls

# Given a dir path, returns a list of dicts describing which tests to run and how.
def load_tests_in_dir(dirpath):
    files = os.listdir(dirpath)
    tests = []
    for f in files:
        if f.startswith('itest_') and f.endswith('.py'):
            tmodname = test_name_from_filename(f)
            fpath = os.path.join(dirpath, f)
            ftests = parse_test_file(fpath)
            if len(ftests) == 0:
                continue
            mod = __load_mod_from_file(f, tmodname)
            for t in ftests:
                tests.append({
                    'name': t['test_name'],
                    'test_func': getattr(mod, t['func_name']),
                    'node_cnt': t['node_cnt']
                })
    return tests

def run_test_list(tests):
    ok = 0
    fail = 0

    # First, just run the tests.
    for t in tests:
        name = t['name']

        print('==============================')
        print('Running test:', name)

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
            t['test_func'](env)
            env.shutdown()
            print('------------------------------')
            print('Success:', name)
            ok += 1
        except BaseException as e:
            env.shutdown()
            print('------------------------------')
            print('Failure:', name)
            print('\nError:', e)
            if type(e) is KeyboardInterrupt:
                break
            fail += 1
            # TODO Report failures and why.

    # Collect results.
    res = {
        'ok': ok,
        'fail': fail,
        'ignored': len(tests) - ok - fail
    }
    return res

if __name__ == '__main__':
    tests = load_tests_from_file('tests.txt')
    run_test_list(tests)
