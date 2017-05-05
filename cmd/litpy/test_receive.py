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

def main(args):
	if not checkPortOpen(8001):
		return

	# start regtest node
	regConn = RegtestConn()
	regConn.Start()
	regConn.mineblock(101)
	regConn.getinfo()
	
	#start lit
	print("starting lit...")
	
	node0 = LitNode(0)
	node0.args.extend(["-spv", "127.0.0.1", "-reg"])
	node0.start_node()
	
	litConn0 = litrpc.LitConnection("127.0.0.1", "8001")
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
	

	bal = litConn0.balance()
	dprint("previous bal: " + str(bal))
	addr = litConn0.new_address()
	# check what's in addr[] (2 things)
	newConn.sendTo(addr[0], 12.34)
	
	print("waiting to receive transaction")
	
	#wait for transaction to be received (max 15 seconds timeout)
	for i in range(5):
		time.sleep(3)
		balNew = litConn0.balance()
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
	