#!/usr/bin/python3

import websocket
import sys
import random
import subprocess
import os
import shutil
from string import digits
import time
from litrpc import *
from test_lit import LitNode

def main(args):
	# if not checkPortOpen(8001):
	# 	return

	# start regtest node
	regConn = RegtestConn()
	regConn.start()
	regConn.mineblock(101)
	regConn.getinfo()
	
	#start lit
	print("starting lit...")
	
	node0 = LitNode(0)
	node0.args.extend(["-spv", "127.0.0.1", "-reg"])
	node0.start_node()
	
	litConn0 = LitConnection("127.0.0.1", "8001")
	litConn0.connect()
	litConn0.new_address()

	#litProcess = subprocess.Popen(["./../../lit","-spv","127.0.0.1","-reg", "-dir", "/tmp/test1"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
	#while True:
	#	time.sleep(1)
	#	try:
	#		litConn = LitConn()
	#		break
	#	except:
	#		print("...")
	#		continue
	

	bal = litConn0.balance()["result"]["TxoTotal"]
	dprint("previous bal: " + str(bal))
	addr = litConn0.new_address()
	# check what's in addr[] (2 things)
	regConn.sendTo(addr["result"]["LegacyAddresses"][0], 12.34)
	
	print("waiting to receive transaction")
	
	#wait for transaction to be received (max 15 seconds timeout)
	for i in range(6):
		time.sleep(5)
		balNew = litConn0.balance()["result"]["TxoTotal"]
		dprint("current bal: " + str(balNew))
		if balNew - bal == 1234000000:
			print("PASS")
			break
		print("...")
	else:
		print("FAIL")
	
	regConn.stop()
	litConn0.Stop()
	
if __name__ == '__main__':
	main(sys.argv)
