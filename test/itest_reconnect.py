
import testlib

def run_test(env):
    print('Starting nodes')
    lit1 = env.new_lit_node()
    lit2 = env.new_lit_node()
    print('Connecting nodes...')
    lit1.connect_to_peer(lit2)
    print('OK, shutting down node 1...')
    lit1.shutdown()
    print('Restarting...')
    lit1.start()
    print('Connecting to node 2.')
    lit1.connect_to_peer(lit2)
    print('Connected! OK')

def run_test_unordered(env):
    print('Starting nodes')
    lit1 = env.new_lit_node()
    lit2 = env.new_lit_node()
    lit3 = env.new_lit_node()
    print('Connecting nodes... (2 then 3)')
    lit1.connect_to_peer(lit2)
    lit1.connect_to_peer(lit3)
    print('OK, shutting down node 1...')
    lit1.shutdown()
    print('Restarting...')
    lit1.start()
    lit1.resync()
    print('Connecting nodes again... (3 then 2)')
    lit1.connect_to_peer(lit3)
    lit1.connect_to_peer(lit2)
    print('Connected! OK')
