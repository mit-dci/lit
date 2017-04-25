import rpc from 'node-json-rpc';
import Q from 'q';

//TODO: simulate pinging the local #lit server to make sure it's on
// let dialString = lc.remote + ':' + lc.port
//
// ping.createSession().pingHost (dialString, function (error, target) {
//   if(error) {
//     console.log('dialing' + target + ': ' + error.toString ());
//     process.exit(1)
//   }
// });
//

let callbacks = {};
let requestNonce = 0;

class LitAfClient {
  constructor () {
    this.host = location.hostname;
    this.port = location.port;
    const rpcOptions = {
      host: this.host,
      port: this.port,
      strict: false,
    };
    this.rpccon = new WebSocket('ws://' + this.host + ':' + this.port + '/ws')
    let deferred = Q.defer()
    this.waitForConnection = deferred.promise
    this.rpccon.onopen = () => {
      deferred.resolve();
    };

    this.rpccon.onmessage = (message) => {
      let data = JSON.parse(message.data);
      if(data.error !== null) {
        callbacks[data.id].reject(data.error);
      }else {
        callbacks[data.id].resolve(data.result);
      }

      delete callbacks[data.id];
    };
  }
  send (method, ...args) {
    let deferred = Q.defer();
    let id = requestNonce++;
    this.waitForConnection.then(() => {
      this.rpccon.send(JSON.stringify({'method': method, 'params': args, 'id': id}));
    })
    callbacks[id] = deferred;

    return deferred.promise;
  }
}

let lc = new LitAfClient()

export default lc;