#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Test lit"""
import argparse
import logging
import subprocess
import sys
import tempfile
import time
import traceback

from bcnode import BCNode
from litnode import LitNode

class LitTest():
    """A lit test case"""

    # Mainline functions. run_test() should be overridden by subclasses. Other
    # methods should not be overridden.
    def __init__(self):
        self.litnodes = []
        self.bcnodes = []
        self.tmpdir = tempfile.mkdtemp(prefix="test")
        self._getargs()
        self._start_logging()
        self.log.info("Using tmp dir %s" % self.tmpdir)

    def main(self):
        """Setup, run and cleanup test case"""
        rc = 0
        try:
            self.run_test()
            self.log.info("Test succeeds!")
        except:
            # Test asserted. Return 1
            self.log.error("Unexpected error: %s" % str(sys.exc_info()[0]))
            traceback.print_exc(file=sys.stdout)
            rc = 1
        finally:
            self.cleanup()

        return rc

    def run_test(self):
        """Test Logic. This method should be overridden by subclasses"""
        raise NotImplementedError

    def cleanup(self):
        """Cleanup test resources"""
        for bcnode in self.bcnodes:
            bcnode.stop()
            try:
                bcnode.process.wait(2)
            except subprocess.TimeoutExpired:
                bcnode.process.kill()
        for litnode in self.litnodes:
            litnode.Stop()
            try:
                litnode.process.wait(2)
            except subprocess.TimeoutExpired:
                litnode.process.kill()

    # Helper methods. Can be called by test case subclasses
    def add_litnode(self):
        self.litnodes.append(LitNode(self.tmpdir))

    def add_bcnode(self):
        self.bcnodes.append(BCNode(self.tmpdir))

    def log_balances(self, coin_type):
        log_str = "Balances:"
        for node in self.litnodes:
            log_str += " litnode%s: " % node.index
            log_str += str(node.get_balance(coin_type))
        self.log.info(log_str)

    # Internal methods. Should not be called by test case subclasses
    def _getargs(self):
        """Parse arguments and pass through unrecognised args"""
        parser = argparse.ArgumentParser(description=__doc__)
        parser.add_argument("--loglevel", "-l", default="INFO", help="log events at this level and higher to the console. Can be set to DEBUG, INFO, WARNING, ERROR or CRITICAL. Passing --loglevel DEBUG will output all logs to console. Note that logs at all levels are always written to the test_framework.log file in the temporary test directory.")
        args, unknown_args = parser.parse_known_args()
        self.loglevel = args.loglevel

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
        ll = int(self.loglevel) if self.loglevel.isdigit() else self.loglevel.upper()
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
        elapsed += 0.05
        time.sleep(0.05)

    raise AssertionError("wait_until() timed out")
