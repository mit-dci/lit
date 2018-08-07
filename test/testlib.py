import os
import os.path as paths
import time
import signal
import subprocess
import logging

import testutil
import btcrpc
import litrpc

LIT_BIN = "%s/../lit" % paths.abspath(paths.dirname(__file__))

REGTEST_COINTYPE = 257

logger = logging.getLogger("testframework")

next_unused_port = 11000
def new_port():
    global next_unused_port
    port = next_unused_port
    next_unused_port += 1
    return port

def get_root_data_dir():
    if 'LIT_ITEST_ROOT' in os.environ:
        return os.environ['LIT_ITEST_ROOT']
    else:
        return "%s/_data" % paths.abspath(paths.dirname(__file__))

datadirnums = {}
def new_data_dir(name):
    global datadirnums
    id = 0
    if name in datadirnums:
        id = datadirnums[name]
        datadirnums[name] += 1
    else:
        datadirnums[name] = 1 # set the next unused to "1"
    p = paths.join(get_root_data_dir(), name + str(id))
    os.makedirs(p, exist_ok=True)
    return p

MSG = "01234567890abcdef"
msg_i = 0
def next_id():
    global msg_i
    m = MSG[msg_i]
    msg_i += 1
    return m

class LitNode():
    def __init__(self, bcnode):
        self.id = next_id()
        self.p2p_port = new_port()
        self.rpc_port = new_port()
        self.data_dir = new_data_dir("lit")
        self.peer_mapping = {}

        # Write a hexkey to the privkey file
        with open(paths.join(self.data_dir, "privkey.hex"), 'w+') as f:
            f.write("1" * 63 + str(self.id) + "\n") # won't work if >=16 lits

        # See if we should print stdout
        outputredir = subprocess.DEVNULL
        if os.getenv("LIT_OUTPUT_SHOW", default="0") == "1":
            outputredir = None

        # Now figure out the args to use and then start Lit.
        args = [
            LIT_BIN,
            "-v",
            "--reg", "127.0.0.1:" + str(bcnode.p2p_port),
            "--tn3", "", # disable autoconnect
            "--dir", self.data_dir,
            "--rpcport=" + str(self.rpc_port),
            "--autoReconnect",
            "--autoListenPort=" + str(self.p2p_port)
        ]
        self.proc = subprocess.Popen(args,
            stdin=subprocess.DEVNULL,
            stdout=outputredir,
            stderr=outputredir)

        # Make the RPC client for future use, too.
        testutil.wait_until_port("localhost", self.rpc_port)
        self.rpc = litrpc.LitClient("localhost", str(self.rpc_port))

        # Make it listen to P2P connections!
        lres = self.rpc.Listen(Port=":" + str(self.p2p_port))
        testutil.wait_until_port("localhost", self.p2p_port)
        self.lnid = lres["Adr"]

    def get_sync_height(self):
        for bal in self.rpc.balance():
            if bal['CoinType'] == REGTEST_COINTYPE:
                return bal['SyncHeight']
        return -1

    def connect_to_peer(self, other):
        addr = other.lnid + '@127.0.0.1:' + str(other.p2p_port)
        res = self.rpc.Connect(LNAddr=addr)
        if "_error" not in res:
            self.update_peers()
            if self.peer_mapping[other.lnid] != res['PeerIdx']:
                raise AssertError("new peer ID doesn't match reported ID")
            other.update_peers()
        else:
            raise AssertError("couldn't connect to " + lnaddr)

    def get_peer_id(self, other):
        return self.peer_mapping[other.lnid]

    def make_new_addr(self):
        res = self.rpc.Address(NumToMake=1, CoinType=REGTEST_COINTYPE)
        if "_error" not in res:
            return res['WitAddresses'][0]
        else:
            raise AssertError("unable to create new address")

    def update_peers(self):
        res = self.rpc.ListConnections()
        if "_error" in res:
            raise AssertError("couldn't ask for peers")
        pm = {}
        for p in res['Connections']:
            pm[p['LitAdr']] = p['PeerNumber']
        self.peer_mapping = pm

    def get_balance_info(self, cointype=None):
        ct = REGTEST_COINTYPE
        if cointype is not None: # I had to do this because of reasons.
            ct = cointype
        for b in self.rpc.balance():
            if b['CoinType'] == ct:
                return b
        return None

    def shutdown(self):
        if self.proc is not None:
            self.proc.kill()
            self.proc = None
        else:
            pass # do nothing I guess?

class BitcoinNode():
    def __init__(self):
        self.p2p_port = new_port()
        self.rpc_port = new_port()
        self.data_dir = new_data_dir("bitcoind")

        # Actually start the bitcoind
        args = [
            "bitcoind",
            "-regtest",
            "-server",
            "-printtoconsole",
            "-datadir=" + self.data_dir,
            "-port=" + str(self.p2p_port),
            "-rpcuser=rpcuser",
            "-rpcpassword=rpcpass",
            "-rpcport=" + str(self.rpc_port),
        ]
        self.proc = subprocess.Popen(args,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL)

        # Make the RPC client for it.
        testutil.wait_until_port("localhost", self.rpc_port)
        testutil.wait_until_port("localhost", self.p2p_port)
        self.rpc = btcrpc.BtcClient("localhost", self.rpc_port, "rpcuser", "rpcpass")

        # Make sure that we're actually ready to accept RPC calls.
        def ck_ready():
            bci = self.rpc.getblockchaininfo() # just need "some call" that'll fail if we're not ready
            if 'code' in bci and bci['code'] <= 0:
                return False
            else:
                return True
        testutil.wait_until(ck_ready, errmsg="took too long to load wallet")

        # Activate SegWit (apparently this is how you do it)
        self.rpc.generate(500)
        def ck_segwit():
            bci = self.rpc.getblockchaininfo()
            try:
                return bci["bip9_softforks"]["segwit"]["status"] == "active"
            except:
                return False
        testutil.wait_until(ck_segwit, errmsg="couldn't activate segwit")

    def shutdown(self):
        if self.proc is not None:
            self.proc.kill()
            self.proc = None
        else:
            pass # do nothing I guess?

class TestEnv():
    def __init__(self, litcnt):
        logger.info("starting nodes...")
        self.bitcoind = BitcoinNode()
        self.lits = []
        for i in range(litcnt):
            node = LitNode(self.bitcoind)
            self.lits.append(node)
        logger.info("started nodes!  syncing...")

        time.sleep(0.1)

        # Sync the nodes
        try:
            self.generate_block(count=0)
        except Exception as e:
            logger.warning("probem syncing nodes, exiting (" + str(e) + ")")
            self.shutdown()
        logger.info("nodes synced!")

    def get_height(self):
        return self.bitcoind.rpc.getblockchaininfo()['blocks']

    def generate_block(self, count=1):
        if count > 0:
            self.bitcoind.rpc.generate(count)
        h = self.get_height()
        def ck_lits_synced():
            for l in self.lits:
                sh = l.get_sync_height()
                if sh != h:
                    return False
            return True
        testutil.wait_until(ck_lits_synced, errmsg="lits aren't syncing to bitcoind")

    def shutdown(self):
        for l in self.lits:
            l.shutdown()
        self.bitcoind.shutdown()
