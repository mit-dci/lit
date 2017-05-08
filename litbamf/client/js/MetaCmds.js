import lc from './LitClient';
import Q from 'q';

class MetaCmds extends React.Component {
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
        <button onClick={this.stop}>stop</button>
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