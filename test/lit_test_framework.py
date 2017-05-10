#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Test lit"""
import subprocess
import sys
import tempfile
import traceback

class LitTest():
    def __init__(self):
        self.litnodes = []
        self.bcnodes = []
        self.tmpdir = tempfile.mkdtemp(prefix="test")
        print("Using tmp dir %s" % self.tmpdir)

    def main(self):
        rc = 0
        try:
            self.run_test()
            print("Test succeeds!")
        except:
            # Test asserted. Return 1
            print("Unexpected error:", sys.exc_info()[0])
            traceback.print_exc(file=sys.stdout)
            rc = 1
        finally:
            self.cleanup()

        return rc

    def run_test(self):
        """Test Logic. This method should be overridden by subclasses"""
        raise NotImplementedError

    def cleanup(self):
        # Stop bitcoind and lit nodes
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

if __name__ == "__main__":
    exit(LitTest().main())
