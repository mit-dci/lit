import testlib

okmsg = 'Hello, world!'

def succeed(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]
    lit1.connect_to_peer(lit2)
    l2idx = lit1.get_peer_id(lit2)

    res = lit1.rpc.PingPeer(PeerIdx=l2idx, Msg=okmsg)
    assert res['Resp'] == okmsg, 'response message not same as call message'
    assert res['Err'] == '', 'response Err field non-null'

def fail(env):
    lit1 = env.lits[0]
    lit2 = env.lits[1]
    lit1.connect_to_peer(lit2)
    l2idx = lit1.get_peer_id(lit2)

    msg = 'something' * 256
    res = lit1.rpc.PingPeer(PeerIdx=l2idx, Msg=msg)
    assert res['Err'] != '', 'response Err field non-null'
