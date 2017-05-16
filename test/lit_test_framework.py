#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Test lit"""
import argparse
import collections
import glob
import logging
try:
    import ipdb as pdb
except:
    import pdb
import shutil
import subprocess
import sys
import tempfile
import time
import traceback

from bcnode import BCNode, LCNode
from litnode import LitNode

class LitTest():
    """A lit test case"""

    # Mainline functions. run_test() should be overridden by subclasses. Other
    # methods should not be overridden.
    def __init__(self):
        # Warn and exit if lit or coin nodes are already running
        try:
            if subprocess.check_output(["pidof", "lit"]) is not None:
                print("ERROR! There is already a lit process running on this system. Tests may fail unexpectedly due to resource contention!")
                sys.exit(1)
            if subprocess.check_output(["pidof", "bitcoind"]) is not None:
                print("ERROR! There is already a bitcoind process running on this system. Tests may fail unexpectedly due to resource contention!")
                sys.exit(1)
            if subprocess.check_output(["pidof", "litecoind"]) is not None:
                print("ERROR! There is already a litecoind process running on this system. Tests may fail unexpectedly due to resource contention!")
                sys.exit(1)
        except (OSError, subprocess.SubprocessError):
            pass
        self.litnodes = []
        self.bcnodes = []
        self.lcnodes = []
        self.tmpdir = tempfile.mkdtemp(prefix="test")
        self._getargs()
        self._start_logging()
        self.log.info("Using tmp dir %s" % self.tmpdir)

    def main(self):
        """Setup, run and cleanup test case"""
        self.rc = 0
        try:
            self.run_test()
            self.log.info("Test succeeds!")
        except:
            # Test asserted. Return 1
            self.log.error("Unexpected error: %s" % str(sys.exc_info()[0]))
            traceback.print_exc(file=sys.stdout)
            self.rc = 1
            if self.args.debugger:
                self.log.info("Attaching debugger")
                pdb.set_trace()
        finally:
            self.cleanup()

        return self.rc

    def run_test(self):
        """Test Logic. This method should be overridden by subclasses"""
        raise NotImplementedError

    def cleanup(self):
        """Cleanup test resources"""
        if self.rc == 1:
            # Dump the end of the debug logs, to aid in debugging rare
            # travis failures.
            filenames = [self.tmpdir + "/test_framework.log"]
            filenames += glob.glob(self.tmpdir + "/bcnode*/regtest/debug.log")
            filenames += glob.glob(self.tmpdir + "/lcnode*/regtest/debug.log")
            filenames += glob.glob(self.tmpdir + "/litnode*/lit.log")
            for fn in filenames:
                try:
                    with open(fn, 'r') as f:
                        print("From %s:\n" % fn)
                        print("".join(collections.deque(f, 500)))
                except OSError:
                    print("Opening file %s failed." % fn)
                    traceback.print_exc()

        for bcnode in self.bcnodes:
            bcnode.stop()
            try:
                bcnode.process.wait(2)
            except subprocess.TimeoutExpired:
                bcnode.process.kill()

        for lcnode in self.lcnodes:
            lcnode.stop()
            try:
                lcnode.process.wait(2)
            except subprocess.TimeoutExpired:
                lcnode.process.kill()

        for litnode in self.litnodes:
            litnode.Stop()
            try:
                litnode.process.wait(2)
            except subprocess.TimeoutExpired:
                litnode.process.kill()

        if self.rc == 0 and not self.args.nocleanup:
            self.log.info("Cleaning up")
            shutil.rmtree(self.tmpdir)
        else:
            self.log.warning("Not cleaning up %s" % self.tmpdir)

    # Helper methods. Can be called by test case subclasses
    def add_litnode(self):
        self.litnodes.append(LitNode(self.tmpdir))

    def add_bcnode(self):
        self.bcnodes.append(BCNode(self.tmpdir))

    def add_lcnode(self):
        self.lcnodes.append(LCNode(self.tmpdir))

    def log_balances(self, coin_type):
        log_str = "Balances:"
        for node in self.litnodes:
            log_str += " litnode%s: " % node.index
            log_str += str(node.get_balance(coin_type))
        self.log.info(log_str)

    def log_channel_balance(self, node1, node1_chan, node2, node2_chan):
        log_str = "Channel balance: " + \
                  str(node1.ChannelList()['result']['Channels'][node1_chan]['MyBalance']) + \
                  " // " + \
                  str(node2.ChannelList()['result']['Channels'][node2_chan]['MyBalance'])
        self.log.info(log_str)

    # Internal methods. Should not be called by test case subclasses
    def _getargs(self):
        """Parse arguments and pass through unrecognised args"""
        parser = argparse.ArgumentParser(description=__doc__)
        parser.add_argument("--debugger", "-d", action='store_true', help="Automatically attach a debugger on test failure.")
        parser.add_argument("--loglevel", "-l", default="INFO", help="log events at this level and higher to the console. Can be set to DEBUG, INFO, WARNING, ERROR or CRITICAL. Passing --loglevel DEBUG will output all logs to console. Note that logs at all levels are always written to the test_framework.log file in the temporary test directory.")
        parser.add_argument("--nocleanup", "-n", action='store_true', help="Don't clean up the test directory after running (even on success).")
        self.args, self.unknown_args = parser.parse_known_args()

    def _start_logging(self):
        """Add logging"""
        # Add logger and logging handlers
        self.log = logging.getLogger('TestFramework')
        self.log.setLevel(logging.DEBUG)
        self.log.propagate = False

        # Create file handler to log all messages
        fh = logging.FileHandler(self.tmpdir + '/test_framework.log')
        fh.setLevel(logging.DEBUG)
        # Create console handler to log messages to stderr. By default this logs only error messages, but can be configured with --loglevel.
        ch = logging.StreamHandler(sys.stdout)
        # User can provide log level as a number or string (eg DEBUG). loglevel was caught as a string, so try to convert it to an int
        ll = int(self.args.loglevel) if self.args.loglevel.isdigit() else self.args.loglevel.upper()
        ch.setLevel(ll)
        # Format logs the same as bitcoind's debug.log with microprecision (so log files can be concatenated and sorted)
        formatter = logging.Formatter(fmt='%(asctime)s.%(msecs)03d000 %(name)s (%(levelname)s): %(message)s', datefmt='%Y-%m-%d %H:%M:%S')
        formatter.converter = time.gmtime
        fh.setFormatter(formatter)
        ch.setFormatter(formatter)
        # add the handlers to the logger
        self.log.addHandler(fh)
        self.log.addHandler(ch)

        rpc_logger = logging.getLogger("litrpc")
        rpc_logger.setLevel(logging.DEBUG)
        rpc_logger.propagate = False
        rpc_logger.addHandler(fh)
        rpc_logger.addHandler(ch)

def wait_until(predicate, *, attempts=float('inf'), timeout=float('inf')):
    if attempts == float('inf') and timeout == float('inf'):
        timeout = 60
    attempt = 0
    elapsed = 0

    while attempt < attempts and elapsed < timeout:
        if predicate():
            return True
        attempt += 1
        elapsed += 0.25
        time.sleep(0.25)

    raise AssertionError("wait_until() timed out")
