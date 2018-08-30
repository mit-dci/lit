
import json
import logging

import requests

logger = logging.getLogger("testframework")

class BtcClient():

    def __init__(self, ip, port, username, userpw):
        self.msg_id = 0
        self.rpc_url = "http://" + username + ":" + userpw + "@" + ip + ":" + str(port) + "/"

    def send_message(self, method, pos_args, named_args):
        if pos_args and named_args:
            raise AssertionError("RPCs must not use a mix of positional and named arguments")
        elif named_args:
            params = named_args
        else:
            params = list(pos_args)

        logger.debug("Sending bitcoind rpc message to %s:%s %s(%s)".format(self.ip, self.port, method, str(params)))

        self.msg_id += 1
        reqbody = {
            "method": method,
            "params": params,
            "jsonrpc": "2.0",
            "id": str(self.msg_id)
        }

        req = requests.post(self.rpc_url, headers={"Content-type": "application/json"}, data=json.dumps(reqbody))
        if req.status_code != 200:
            raise AssertionError('RPC error: HTTP response code is ' + str(req.status_code))

        if req.json()['error'] is None:
            resp = req.json()['result']
            logger.debug("Received rpc response from %s:%s method: %s Response: %s." % (self.ip, self.port, method, str(resp)))
            return resp
        else:
            raise AssertionError('RPC call failed: ' + str(req.json()['error']))

    def __getattr__(self, name):
        def dispatcher(*args, **kwargs):
            return self.send_message(name, args, kwargs)
        return dispatcher
