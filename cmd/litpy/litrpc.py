#!/usr/bin/python3


import websocket
import json
import sys




def main(args):
	
	rpcCmd = {
	   "method": "LitRPC.Address",
	   "params": [{"NumToMake": 0}]
	}

	rpcCmd.update({"jsonrpc": "2.0", "id": "94"})
	
	ws = websocket.WebSocket()
	ws.connect("ws://127.0.0.1:8001/ws")
	
	ws.send(json.dumps(rpcCmd))
	result = json.loads(ws.recv())
	
	#~ result = ws.recv()
	print("got a result")
	#~ print(result)
	print(result["result"]["WitAddresses"][2])
	
	rpc2 = {
	   "method": "LitRPC.Bal",
	   "params": []
	}
	rpc2.update({"jsonrpc": "2.0", "id": "92"})
	
	ws.send(json.dumps(rpc2))
	result = json.loads(ws.recv())
	print(result)
	
if __name__ == '__main__':
    main(sys.argv)
