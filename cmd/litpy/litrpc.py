#!/usr/bin/python3

import websocket
import json
import sys
import requests

class RegtestConn:
	rpcuser = "regtestuser"
	rpcpass = "regtestpass"
	rpcport = 18332
	serverURL = "http://" + rpcuser + ":" + rpcpass + "@127.0.0.1:" + str(rpcport)
	header = {"Content-type": "application/json"}
	
	def __init__(self):
		self.id = 0
	
	def mineblock(self, number):
		self.id += 1
		rpcCmd = {
			"method": "generate",
			"params": [number],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		payload = json.dumps(rpcCmd)
		print("sending: " + payload)
		response = requests.post(RegtestConn.serverURL, headers=RegtestConn.header, data=payload)
		print("received: " + str(response.json()))
	
	def getinfo(self):
		self.id += 1
		rpcCmd = {
			"method": "getinfo",
			"params": [],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		payload = json.dumps(rpcCmd)
		print("sending: " + payload)
		response = requests.post(RegtestConn.serverURL, headers=RegtestConn.header, data=payload)
		print("received: " + str(response.json()))
	
	def sendTo(self, addr, amt):
		self.id += 1
		rpcCmd = {
			"method": "sendtoaddress",
			"params": [addr, amt],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		payload = json.dumps(rpcCmd)
		print("sending: " + payload)
		response = requests.post(RegtestConn.serverURL, headers=RegtestConn.header, data=payload)
		print("received: " + str(response.json()))

class LitConn:
	def __init__(self):
		self.id = 0
		self.ws = websocket.WebSocket()
		self.ws.connect("ws://127.0.0.1:8001/ws")
	
	def getAddress(self):
		self.id += 1
		rpcCmd = {
			"method": "LitRPC.Address",
			"params": [{"NumToMake": 0}],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		self.ws.send(json.dumps(rpcCmd))
		result = json.loads(self.ws.recv())
		print(result)
	
	def getBal(self):
		self.id += 1
		rpcCmd = {
			"method": "LitRPC.Bal",
			"params": [],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		self.ws.send(json.dumps(rpcCmd))
		result = json.loads(self.ws.recv())
		print(result)

def main(args):
	newConn = RegtestConn()
	newConn.mineblock(5)
	#newConn.getinfo()
	
	litConn = LitConn()
	litConn.getAddress()
	litConn.getBal()

if __name__ == '__main__':
    main(sys.argv)
