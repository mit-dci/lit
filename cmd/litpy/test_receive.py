#!/usr/bin/python3

import sys
import random
import subprocess
import os
import signal
import shutil
import threading
from string import ascii_lowercase, digits
import random
import time
from litrpc import *

"""
class litThread(threading.Thread):
	def __init__(self, threadID, name):
		threading.Thread.__init__(self)
		self.threadID = threadID
		self.name = name
		
	def run(self):
		print ("Starting " + self.name)
		#subprocess.call(["./../../lit","-spv","127.0.0.1","-tip","1080000","-reg", "-dir", "/dev/shm/test1"])
		# The os.setsid() is passed in the argument preexec_fn so
		# it's run after the fork() and before  exec() to run the shell.
		pro = subprocess.Popen(["./../../lit","-spv","127.0.0.1","-tip","1080000","-reg", "-dir", "/dev/shm/test1"], shell=True)
		return pro
		
	def kill(self):
		os.killpg(os.getpgid(self.pro.pid), signal.SIGTERM)  # Send the signal to all the process groups
		print ("Exiting " + self.name)
"""

def main(args):
	#stop and remove current regtest server
	subprocess.call(["bitcoin-cli", "-regtest", "stop"])
	home = os.path.expanduser('~')
	if os.path.exists(home + "/.bitcoin/regtest"):
		shutil.rmtree(home + "/.bitcoin/regtest")
		
	time.sleep(1) #otherwise regtest might have error 1 when starting...?
	subprocess.call(["bitcoind","-daemon","-regtest"]) #restart regtest

	#remove current regtest user data
	if os.path.exists("/tmp/test1"):
		shutil.rmtree("/tmp/test1")
	
	#make new regtest user
	os.makedirs("/tmp/test1")
	f = open("/tmp/test1/testkey.hex","w+")
	randomString = ''.join(random.choice("abcdef"+digits) for i in range(64))
	f.write(randomString + "\n")
	f.close()
	
	#wait for regtest network to start up
	while True:
		time.sleep(3)
		try:
			subprocess.check_output(["bitcoin-cli","-regtest","getbalance"], stderr=subprocess.STDOUT)
			break #if there is no exception in previous line, regtest network has started up
		except:
			print("...")
			continue
	
	#start lit
	print("starting lit...")
	litProcess = subprocess.Popen(["./../../lit","-spv","127.0.0.1","-reg", "-dir", "/tmp/test1"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
	while True:
		time.sleep(1)
		try:
			litConn = LitConn()
			break
		except:
			print("...")
			continue
	
	newConn = RegtestConn()
	newConn.mineblock(101)
	newConn.getinfo()

	bal = litConn.getBal()
	addr = litConn.getLegacyAddress()
	newConn.sendTo(addr, 12.34)
	
	print("waiting to receive transaction")
	
	#wait for transaction to be received (max 15 seconds timeout)
	for i in range(5):
		time.sleep(3)
		balNew = litConn.getBal()
		if balNew != bal and int(balNew) - int(bal) == 1234000000:
			print("PASS!")
			break
		print("...")
	else:
		print("FAIL")
	
	#addr = litConn.getWitAddress()
	#resp0 = litConn.litSend([addr], [1000000])
	#print(resp0)
	
	litProcess.terminate()

	#print("error!!!")
	#print(sys.exc_info())
	
	
if __name__ == '__main__':
	main(sys.argv)
	