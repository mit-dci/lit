
import time
import testlib

def run_test(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]

    # First just connect them so they know about each other.
    print('Connecting to peer...')
    lit1.connect_to_peer(lit2)
    time.sleep(0.1)
    id = lit1.get_peer_id(lit2)
    print('Peer ID:', id)

    # Now shutdown.
    print('Shutting down and restarting...')
    lit1.shutdown()

    # Now restart and hope they're connected.
    lit1.start()
    time.sleep(0.1)

    # Recheck.
    print('Now re-checking')
    lit1.update_peers()
    try:
        idnew = lit1.get_peer_id(lit2)
        assert id == idnew, 'peer ID'
    except:
        raise AssertionError('peer not found!')
