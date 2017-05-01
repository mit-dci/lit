import lc from './LitClient';
import Q from 'q';

class MetaCmds extends React.Component {
  ls () {
    console.log('all of them');
    Q.spread([
      lc.send('LitRPC.ListConnections'),
      lc.send('LitRPC.ChannelList'),
      lc.send('LitRPC.TxoList'),
      lc.send('LitRPC.GetListeningPorts'),
      lc.send('LitRPC.Address'),
      lc.send('LitRPC.Bal'),
      lc.send('LitRPC.SyncHeight'),
    ], (connections, channels, txos, ports, addresses, balances, syncHeight) => {
      console.log('peers: ' + JSON.stringify(connections));
      console.log('channels: ' + JSON.stringify(channels));
      console.log('txo list: ' + JSON.stringify(txos));
      console.log('listening ports: ' + JSON.stringify(ports));
      console.log('addresses' + JSON.stringify(addresses));
      console.log('balance' + JSON.stringify(balances));
      console.log('sync height' + JSON.stringify(syncHeight));
    })
    .fail(err => {
      console.error(err);
    });
  }
  stop () {
    lc.send('LitRPC.Stop')
    .then(res => {
      window.location = '/exit';
    })
    .fail(err => {
      console.error(err);
    });
  }
  render () {
    return (
      <div>
        <h2>Meta Commands</h2>

        <button onClick={this.ls}>ls</button>
        <button onClick={this.stop}>stop</button>
        <button onClick={this.exit}>exit</button>
      </div>
    );
  }
}

// client.call(
//   {'jsonrpc': '2.0', 'method': 'myMethod', 'params': [1,2], 'id': 0},
//   function (err, res) {
//     // Did it all work ?
//     if (err) { console.log(err); }
//     else { console.log(res); }
//   }
// );

export default MetaCmds;