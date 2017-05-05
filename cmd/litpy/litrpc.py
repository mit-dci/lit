#!/usr/bin/python3
"""Python interface to lit"""

import json
import random
import time
import subprocess
import os
import shutil
import requests
import websocket  # `pip install websocket-client`

DEBUG = False

def dprint(x):
	if DEBUG:
		print(x)

class LitConnection():
    """A class representing a connection to a lit node."""
    def __init__(self, ip, port):
        self.ip = ip
        self.port = port

    def connect(self):
        """Connect to the node. Continue trying for 10 seconds"""
        self.ws = websocket.WebSocket()
        for _ in range(50):
            try:
                self.ws.connect("ws://%s:%s/ws" % (self.ip, self.port))
            except ConnectionRefusedError:
                # lit is not ready to accept connections yet
                time.sleep(0.2)
            else:
                # No exception - we're connected!
                break
        self.msg_id = random.randint(0, 9999)

    def send_message(self, method, params):
        """Sends a websocket message to the lit node"""
        self.ws.send(json.dumps({"method": "LitRPC.%s" % method,
                                 "params": [params],
                                 "jsonrpc": "2.0",
                                 "id": str(self.msg_id)}))

        self.msg_id = self.msg_id + 1 % 10000
        return json.loads(self.ws.recv())

    def __getattr__(self, name):
        """Dispatches any unrecognised messages to the websocket connection"""
        def dispatcher(**kwargs):
            return self.send_message(name, kwargs)
        return dispatcher

    def new_address(self):
        """Add a new wallit address"""
        return self.Address(NumToMake=1)

    def balance(self):
        """Get wallit balance"""
        return self.Bal()

class RegtestConn():
	""" A class representing a regtest node"""
	rpcuser = "regtestuser"
	rpcpass = "regtestpass"
	rpcport = 18332
	serverURL = "http://" + rpcuser + ":" + rpcpass + "@127.0.0.1:" + str(rpcport)
	header = {"Content-type": "application/json"}

	def __init__(self):
		self.id = 0
	
	def start(self):
		#stop and remove current regtest server
		subprocess.call(["bitcoin-cli", "-regtest", "stop"], stdout=subprocess.DEVNULL)
		home = os.path.expanduser('~')
		if os.path.exists(home + "/.bitcoin/regtest"):
			shutil.rmtree(home + "/.bitcoin/regtest")
		
		time.sleep(1) #otherwise regtest might have error 1 when starting...?
		subprocess.call(["bitcoind","-daemon","-regtest"]) #restart regtest

		#remove current regtest user data
		if os.path.exists("/tmp/test1"):
			shutil.rmtree("/tmp/test1")
				#wait for regtest network to start up
		while True:
			time.sleep(3)
			try:
				subprocess.check_output(["bitcoin-cli","-regtest","getbalance"], stderr=subprocess.STDOUT)
				break #if there is no exception in previous line, regtest network has started up
			except:
				print("...")
				continue
	
	def mineblock(self, number, log=False):
		self.id += 1
		rpcCmd = {
			"method": "generate",
			"params": [number],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		payload = json.dumps(rpcCmd)
		
		dprint("sending: " + payload)
		response = requests.post(RegtestConn.serverURL, headers=RegtestConn.header, data=payload)
		dprint("received: " + str(response.json()))
	
	def getinfo(self):
		self.id += 1
		rpcCmd = {
			"method": "getinfo",
			"params": [],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		payload = json.dumps(rpcCmd)
		dprint("sending: " + payload)
		response = requests.post(RegtestConn.serverURL, headers=RegtestConn.header, data=payload)
		dprint("received: " + str(response.json()))
	
	def sendTo(self, addr, amt):
		self.id += 1
		rpcCmd = {
			"method": "sendtoaddress",
			"params": [addr, amt],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		payload = json.dumps(rpcCmd)
		dprint("sending: " + payload)
		response = requests.post(RegtestConn.serverURL, headers=RegtestConn.header, data=payload)
		dprint("received: " + str(response.json()))
		
	def stop(self):
		self.id += 1
		rpcCmd = {
			"method": "stop",
			"params": [],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		payload = json.dumps(rpcCmd)
		dprint("sending: " + payload)
		response = requests.post(RegtestConn.serverURL, headers=RegtestConn.header, data=payload)
		dprint("received: " + str(response.json()))



if __name__ == '__main__':
    """Test litrpc.py. lit instance must be running and available on 127.0.0.1:8001"""
    litConn = LitConnection("127.0.0.1", "8001")
    litConn.connect()
    print(litConn.new_address())
    print(litConn.balance())
