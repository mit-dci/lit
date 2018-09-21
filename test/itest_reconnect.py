
import testlib

def run_test(env):
    print('Starting nodes')
    lit1 = env.new_lit_node()
    lit2 = env.new_lit_node()
    print('Connecting nodes...')
    lit1.connect_to_peer(lit2)
    l2p1 = lit1.get_peer_id(lit2)
    print('OK, shutting down node 1...')
    lit1.shutdown()
    print('Restarting...')
    lit1.start()
    print('Connecting to node 2...')
    lit1.connect_to_peer(lit2)
    l2p2 = lit1.get_peer_id(lit2)
    print('Checking IDs match...')
    print('Node 2: %s -> %s' % (l2p2, l2p2))
    assert l2p1 == l2p2, 'peer IDs on node 1 don\'t match across restarts!'
    print('OK')

def run_test_unordered(env):
    print('Starting nodes')
    lit1 = env.new_lit_node()
    lit2 = env.new_lit_node()
    lit3 = env.new_lit_node()
    print('Connecting nodes... (2 then 3)')
    lit1.connect_to_peer(lit2)
    lit1.connect_to_peer(lit3)
    l2p1 = lit1.get_peer_id(lit2)
    l3p1 = lit1.get_peer_id(lit3)
    print('OK, shutting down node 1...')
    lit1.shutdown()
    print('Restarting...')
    lit1.start()
    lit1.resync()
    print('Connecting nodes again... (3 then 2)')
    lit1.connect_to_peer(lit3)
    lit1.connect_to_peer(lit2)
    l2p2 = lit1.get_peer_id(lit2)
    l3p2 = lit1.get_peer_id(lit3)
    print('Checking IDs match...')
    print('Node 2: %s -> %s' % (l2p1, l2p2))
    print('Node 3: %s -> %s' % (l3p1, l3p2))
    assert l2p1 == l2p2, 'peer IDs on node 1 don\'t match across restarts for node 2!'
    assert l3p1 == l3p2, 'peer IDs on node 1 don\'t match across restarts for node 3!'
    print('OK')
