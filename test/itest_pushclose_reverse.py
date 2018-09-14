import testlib
import test_combinators

def run_test(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]
    test_combinators.run_pushclose_test(env, lit1, lit2, lit2)
