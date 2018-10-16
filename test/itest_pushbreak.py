import testlib
import test_combinators

def forward(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]
    test_combinators.run_pushbreak_test(env, lit1, lit2, lit1)

def reverse(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]
    test_combinators.run_pushbreak_test(env, lit1, lit2, lit2)
