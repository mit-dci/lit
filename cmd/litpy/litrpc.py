#!/usr/bin/python3

import websocket
import json
import sys
import requests
import random
import subprocess

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

	def litNewAddr(self):
		self.id += 1
		rpcCmd = {
		   "method": "LitRPC.Address",
		   "params": [{"NumToMake": 0}],
		   "jsonrpc": "2.0",
		   "id": str(self.id)
		}
			
		self.ws.send(json.dumps(rpcCmd))
		resp = json.loads(self.ws.recv())
		return resp["result"]["WitAddresses"][0]

	def litSend(self, adr, amt):
		self.id += 1
		rpcCmd = {
		   "method": "LitRPC.Send",
		   "params": [{
		   		"DestAddrs": adr,
		   		"Amts": amt}	   
		   		],
		   	"jsonrpc": "2.0",
		   	"id": str(self.id)
		}
		self.ws.send(json.dumps(rpcCmd))
		resp = json.loads(self.ws.recv())
		return resp
	
	def getWitAddress(self):
		self.id += 1
		rpcCmd = {
			"method": "LitRPC.Address",
			"params": [{"NumToMake": 0}],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		self.ws.send(json.dumps(rpcCmd))
		resp = json.loads(self.ws.recv())
		return resp["result"]["WitAddresses"][0]

	def getLegacyAddress(self):
		self.id += 1
		rpcCmd = {
			"method": "LitRPC.Address",
			"params": [{"NumToMake": 0}],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		self.ws.send(json.dumps(rpcCmd))
		resp = json.loads(self.ws.recv())
		return resp["result"]["LegacyAddresses"][0]
	
	def getBal(self):
		self.id += 1
		rpcCmd = {
			"method": "LitRPC.Bal",
			"params": [],
			"jsonrpc": "2.0",
			"id": str(self.id)
		}
		self.ws.send(json.dumps(rpcCmd))
		resp = json.loads(self.ws.recv())
		#TODO: get different kinds of balances
		return resp["result"]["TxoTotal"]

def main(args):
	#subprocess.call(["bitcoind","-daemon","-regtest"])
	#subprocess.call(["./../../../lit","-spv","127.0.0.1","-tip","1080000","-reg"])

	newConn = RegtestConn()
	newConn.mineblock(101)
	newConn.getinfo()

	litConn = LitConn()
	bal = litConn.getBal()
	print(bal)
	addr = litConn.getLegacyAddress()
	print(addr)
	newConn.sendTo(addr, 10.00)
	resp0 = litConn.litSend([addr], [1000000])
	print(resp0)
	resp = litConn.litNewAddr()
	print(resp)

if __name__ == '__main__':
    main(sys.argv)
    