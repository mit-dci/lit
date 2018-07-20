// Slightly modified copy from litbamf - unsure of original author!
// refactored to use native Promises

let callbacks = {};
let requestNonce = 0;

class LitAfClient {
  constructor (rpchost, rpcport) {
    this.host = rpchost;
    this.port = rpcport;

    // open the connection by setting wait for connection to be a promise that is resolved when open
    this.waitForConnection = new Promise (resolve => {
      this.rpccon = new WebSocket('ws://' + this.host + ':' + this.port + '/ws');
      this.rpccon.onopen = () => {
        resolve();
      }
    });

    // set up the received message callback to resolve or reject the sending promise
    this.rpccon.onmessage = (message) => {
      let data = JSON.parse(message.data);
      if(data.error !== null) {
        callbacks[data.id].reject(data.error);
        delete callbacks[data.id];
      } else if(data.id === null) {
        //go to the special chat message handler, but don't delete the callback
        callbacks[data.id].resolve(data.result);
      } else {
        callbacks[data.id].resolve(data.result);
        delete callbacks[data.id];
      }

    };
  }

  // send by creating a new promise and storing the resolve and reject f's for use by the receiving callback
  send (method, ...args) {
    let id = requestNonce++;
    let promise = new Promise((resolve, reject) => {
      this.waitForConnection.then(() => {
        this.rpccon.send(JSON.stringify({'method': method, 'params': args, 'id': id}));
      });
      callbacks[id] = {resolve: resolve, reject: reject};
    });
    return promise;
  }

}

export default LitAfClient;
