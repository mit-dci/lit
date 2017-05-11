#!/usr/bin/env python3
# Copyright (c) 2017 The lit developers
# Distributed under the MIT software license, see the accompanying
# file LICENSE or http://www.opensource.org/licenses/mit-license.php.
"""Python interface to lit"""

import socket
import websocket
import json
import sys
import requests

DEBUG = False

def dprint(x):
	if DEBUG:
		print(x)

def checkPortOpen(port):
	open = True
	s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	try:
		s.bind(("127.0.0.1", port))
	except socket.error as e:
		open = False
		if e.errno == 98:
			print("Port " + str(port) + " used by another process, please close it before continuing tests")
		else:
			# something else raised the socket.error exception
			print(e)
	s.close()
	return open

class RegtestConn:
	rpcuser = "regtestuser"
	rpcpass = "regtestpass"
	rpcport = 18332
	serverURL = "http://" + rpcuser + ":" + rpcpass + "@127.0.0.1:" + str(rpcport)
	header = {"Content-type": "application/json"}

	def __init__(self):
		self.id = 0
	
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

