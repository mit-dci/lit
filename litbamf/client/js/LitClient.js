import rpc from 'node-json-rpc';
import Q from 'q';

let callbacks = {};
let requestNonce = 0;

class LitAfClient {
  constructor () {
    this.host = location.hostname;
    this.port = location.port;
    this.rpccon = new WebSocket('ws://' + this.host + ':' + this.port + '/ws');
    let deferred = Q.defer();
    this.waitForConnection = deferred.promise;
    this.rpccon.onopen = () => {
      deferred.resolve();
    };

    this.rpccon.onmessage = (message) => {
      let data = JSON.parse(message.data);
      if(data.error !== null) {
        callbacks[data.id].reject(data.error);
        delete callbacks[data.id];
      }else if(data.id === null) {
        //go to the special chat message handler, but don't delete the callback
        callbacks[data.id](data.result);
        console.log('dadad');
      }else {
        callbacks[data.id].resolve(data.result);
        delete callbacks[data.id];
      }

    };
  }
  send (method, ...args) {
    let deferred = Q.defer();
    let id = requestNonce++;
    this.waitForConnection.then(() => {
      this.rpccon.send(JSON.stringify({'method': method, 'params': args, 'id': id}));
    });
    callbacks[id] = deferred;

    return deferred.promise;
  }
  register (id, callback) {
    callbacks[id] = callback;
  }
}

let lc = new LitAfClient();

export default lc;