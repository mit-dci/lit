import os
import os.path as paths
import time
import signal
import subprocess

import litrpc

LIT_BIN = "%s/../lit" % paths.abspath(paths.dirname(__file__))

REGTEST_COINTYPE = 257

next_unused_port = 11000
def new_port():
    global next_unused_port
    port = next_unused_port
    next_unused_port += 1
    return port

datadirnums = {}
def new_data_dir(name):
    global datadirnums
    id = 0
    if name in datadirnums:
        id = datadirnums[name]
        datadirnums[name] += 1
    else:
        datadirnums[name] = 1 # set the next unused to "1"
    p = "%s/_data/%s%s" % (paths.abspath(paths.dirname(__file__)), name, id)
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

        # Write a hexkey to the privkey file
        with open(paths.join(self.data_dir, "privkey.hex"), 'w+') as f:
            f.write("1" * 63 + str(self.id) + "\n") # won't work if >=16 lits

        # Now figure out the args to use and then start Lit.
        args = [
            LIT_BIN,
            "-v",
            "--dir", self.data_dir,
            "--rpcport=" + str(self.rpc_port),
            "--tn3", "", # disable autoconnect
            "--reg", "localhost:" + str(bcnode.p2p_port),
            "--autoReconnect",
            "--autoListenPort=" + str(self.p2p_port)
        ]
        self.proc = subprocess.Popen(args,
            stdin=subprocess.PIPE,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL)
        time.sleep(1)

        # Make the RPC client for future use, too.
        self.rpc = litrpc.LitClient("localhost", str(self.rpc_port))

        # Make it listen to P2P connections!
        self.rpc.Listen(Port=":" + str(self.p2p_port))

    def shutdown(self):
        if self.proc is not None:
            self.proc.terminate()
            self.proc = None
        else:
            pass # do nothing I guess?

class BitcoinNode():
    def __init__(self):
        self.p2p_port = new_port()
        self.rpc_port = new_port()
        self.data_dir = new_data_dir("bitcoind")
        args = [
            "bitcoind",
            "-v"
            "-regtest",
            "-datadir=" + self.data_dir,
            "-rpcport=" + str(self.rpc_port),
            "-port=" + str(self.p2p_port)
        ]
        self.proc = subprocess.Popen(args,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL)
        time.sleep(2.5)

    def shutdown(self):
        if self.proc is not None:
            self.proc.terminate()
            self.proc = None
        else:
            pass # do nothing I guess?
