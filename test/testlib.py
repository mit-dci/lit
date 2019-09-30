import os
import os.path as paths
import time
import signal
import subprocess
import logging
import random
import shutil
import json

import testutil
import litrpc
import requests


from bitcoinrpc.authproxy import AuthServiceProxy, JSONRPCException


# The dlcoracle binary must be accessible throught a PATH variable.

LIT_BIN = "%s/../lit" % paths.abspath(paths.dirname(__file__))

ORACLE_BIN = "%s/../dlcoracle" % paths.abspath(paths.dirname(__file__))

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

hexchars = "0123456789abcdef"

# FIXME This doesn't work as expected anymore since IDs are global.
next_id = 0
def get_new_id():
    global next_id
    id = next_id
    next_id += 1
    return id


class LitNode():
    def __init__(self, bcnode):
        self.bcnode = bcnode
        self.id = get_new_id()
        self.p2p_port = new_port()
        self.rpc_port = new_port()
        self.data_dir = new_data_dir("lit")
        self.peer_mapping = {}
        self.proc = None

        # Write a hexkey to the privkey file
        with open(paths.join(self.data_dir, "privkey.hex"), 'w+') as f:
            s = ''
            for _ in range(64):
                s += hexchars[random.randint(0, len(hexchars) - 1)]
            print('Using key (lit):', s)
            f.write(s + "\n")

        # Go and do the initial startup and sync.
        self.start()

    def start(self):
        # Sanity check.
        assert self.proc is None, "tried to start a node that is already started!"

        # See if we should print stdout
        outputredir = subprocess.DEVNULL
        ev_output_show = os.getenv("LIT_OUTPUT_SHOW", default="0")
        ev_show_id = os.getenv("LIT_ID_SHOW", default="X")
        if ev_output_show == "1" and (ev_show_id == "X" or ev_show_id == str(self.id)):
            outputredir = None

        # Now figure out the args to use and then start Lit.
        args = [
            LIT_BIN,
            #"-vv",
            "--reg", "127.0.0.1:" + str(self.bcnode.p2p_port),
            "--tn3", "", # disable autoconnect
            "--dir", self.data_dir,
            "--unauthrpc",
            "--rpcport=" + str(self.rpc_port),
            "--noautolisten"
        ]
        penv = os.environ.copy()
        lkw = 'LIT_KEYFILE_WARN'
        if lkw not in penv:
            penv[lkw] = '0'
        self.proc = subprocess.Popen(args,
            stdin=subprocess.DEVNULL,
            stdout=outputredir,
            stderr=outputredir,
            env=penv)

        # Make the RPC client for future use, too.
        testutil.wait_until_port("localhost", self.rpc_port)
        self.rpc = litrpc.LitClient("localhost", str(self.rpc_port))

        # Make it listen to P2P connections!
        lres = self.rpc.Listen(Port=self.p2p_port)
        testutil.wait_until_port("localhost", self.p2p_port)
        self.lnid = lres["Adr"] # technically we do this more times than we have to, that's okay

    def get_sync_height(self):
        for bal in self.rpc.balance():
            if bal['CoinType'] == REGTEST_COINTYPE:
                return bal['SyncHeight']
        print("return -1")
        return -1

    def connect_to_peer(self, other):
        addr = other.lnid + '@127.0.0.1:' + str(other.p2p_port)
        res = self.rpc.Connect(LNAddr=addr)
        self.update_peers()
        if 'PeerIdx' in res and self.peer_mapping[other.lnid] != res['PeerIdx']:
            raise AssertionError("new peer ID doesn't match reported ID (%s vs %s)" % (self.peer_mapping[other.lnid], res['PeerIdx']))
        other.update_peers()

    def get_peer_id(self, other):
        return self.peer_mapping[other.lnid]

    def make_new_addr(self):
        res = self.rpc.Address(NumToMake=1, CoinType=REGTEST_COINTYPE)
        return res['WitAddresses'][0]

    def update_peers(self):
        res = self.rpc.ListConnections()
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

    def open_channel(self, peer, capacity, initialsend, cointype=None):
        ct = REGTEST_COINTYPE
        if cointype is not None: # I had to do thi because of reasons.
            ct = cointype
        res = self.rpc.FundChannel(
            Peer=self.get_peer_id(peer),
            CoinType=ct,
            Capacity=capacity,
            InitialSend=initialsend,
            Data=None) # maybe use [0 for _ in range(32)] or something?
        return res['ChanIdx']

    def resync(self):
        def ck_synced():
            return self.get_sync_height() == self.bcnode.get_block_height()
        testutil.wait_until(ck_synced, attempts=40, errmsg="node failing to resync!")

    def shutdown(self):
        if self.proc is not None:
            self.proc.kill()
            self.proc.wait()
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
            "-txindex"
        ]

        self.proc = subprocess.Popen(args,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL)


        # Make the RPC client for it.
        testutil.wait_until_port("localhost", self.rpc_port)
        testutil.wait_until_port("localhost", self.p2p_port)

        self.rpc = AuthServiceProxy("http://%s:%s@127.0.0.1:%s"%("rpcuser", "rpcpass", self.rpc_port), timeout=240)

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

    def get_block_height(self):
        return self.rpc.getblockchaininfo()['blocks']

    def shutdown(self):
        if self.proc is not None:
            self.proc.kill()
            self.proc.wait()
            self.proc = None
        else:
            pass # do nothing I guess?

class OracleNode():
    def __init__(self, interval, valueToPublish):

        self.data_dir = new_data_dir("oracle")
        self.httpport = str(new_port())
        self.interval = str(interval)
        self.valueToPublish = str(valueToPublish)

        # Write a hexkey to the privkey file
        with open(paths.join(self.data_dir, "privkey.hex"), 'w+') as f:
            s = ''
            for _ in range(192):
                s += hexchars[random.randint(0, len(hexchars) - 1)]
            print('Using key (oracle):', s)
            f.write(s + "\n")

        self.start()    


    def start(self):

        # See if we should print stdout
        outputredir = subprocess.DEVNULL
        ev_output_show = os.getenv("ORACLE_OUTPUT_SHOW", default="0")
        if ev_output_show == "1":
            outputredir = None

        # Now figure out the args to use and then start Lit.
        args = [
            ORACLE_BIN,
            "--DataDir="+self.data_dir,
            "--HttpPort=" + self.httpport, 
            "--Interval=" + self.interval,
            "--ValueToPublish=" + self.valueToPublish,
        ]

        penv = os.environ.copy()

        self.proc = subprocess.Popen(args,
        stdin=subprocess.DEVNULL,
        stdout=outputredir,
        stderr=outputredir,
        env=penv)

    def shutdown(self):
        if self.proc is not None:
            self.proc.kill()
            self.proc.wait()
            self.proc = None
        else:
            pass # do nothing I guess?

        shutil.rmtree(self.data_dir)
    
    def get_pubkey(self):
        res = requests.get("http://localhost:"+self.httpport+"/api/pubkey")
        return res.text

    def get_datasources(self):
        res = requests.get("http://localhost:"+self.httpport+"/api/datasources")
        return res.text

              
    def get_rpoint(self, datasourceID, unixTime):
        res = requests.get("http://localhost:"+self.httpport+"/api/rpoint/" + str(datasourceID) + "/" + str(unixTime))
        print("get_rpoint:", "http://localhost:"+self.httpport+"/api/rpoint/" + str(datasourceID) + "/" + str(unixTime))
        print(res.text)
        return res.text

    def get_publication(self, rpoint):
        res  = requests.get("http://localhost:"+self.httpport+"/api/publication/" + rpoint)
        return res.text


class TestEnv():
    def __init__(self, litcnt):
        logger.info("starting nodes...")
        self.bitcoind = BitcoinNode()
        self.oracles = []
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

    def new_lit_node(self):
        node = LitNode(self.bitcoind)
        self.lits.append(node)
        self.generate_block(count=0) # Force it to wait for sync.
        return node

    def new_oracle(self, interval, valueToPublish):
        oracle = OracleNode(interval, valueToPublish)
        self.oracles.append(oracle)
        return oracle


    def generate_block(self, count=1):
        if count > 0:
            self.bitcoind.rpc.generate(count)
        h = self.bitcoind.get_block_height()
        def ck_lits_synced():
            for l in self.lits:
                sh = l.get_sync_height()
                if sh != h:
                    return False
            return True
        testutil.wait_until(ck_lits_synced, errmsg="lits aren't syncing to bitcoind")

    def get_height(self):
        return self.bitcoind.get_block_height()

    def shutdown(self):
        for l in self.lits:
            l.shutdown()
        self.bitcoind.shutdown()
        for o in self.oracles:
            o.shutdown()

def clean_data_dir():
    datadir = get_root_data_dir()
    shutil.rmtree(datadir)
