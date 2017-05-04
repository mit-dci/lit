import {Navbar} from './Globals.js';
import lc from './LitClient';
import Q from 'q';
require('../sass/overview.scss');

class Overview extends React.Component {
  constructor (props) {
    super(props);

    this.state =  {
      peers: '',
      port: '',
      available: 0,
      total: 0,
      syncHeight: 0,
    };
  }
  update () {
    Q.spread([
      lc.send('LitRPC.ListConnections'),
      lc.send('LitRPC.GetListeningPorts'),
      lc.send('LitRPC.Bal'),
      lc.send('LitRPC.SyncHeight'),
    ], (connections, ports, balances, syncHeight) => {
      console.log('peers: ' + JSON.stringify(connections));
      console.log('listening ports: ' + JSON.stringify(ports));
      console.log('balance' + JSON.stringify(balances));
      console.log('sync height' + JSON.stringify(syncHeight));

      let peers = connections.Connections !== null ? connections.Connections.length : 0;
      let port = ports.LisIpPorts !== null ? ports.LisIpPorts[0] : 'not listening';
      this.setState({
        peers: peers,
        port: port,
        available: balances.MatureWitty,
        total: balances.TxoTotal,
        syncHeight: syncHeight.SyncHeight,
      });
    })
    .fail(err => {
      console.error(err);
    });
  }
  render () {
    return (
      <div>
        <Navbar />

        <main>
          <div className='stats'>
            <div>
              <div>Block Height <span><img src='/images/cube.svg' /></span></div>
              <div><span>{this.state.syncHeight}</span></div>
              </div>
            <div>
              <div>Listening Ports <span><img src='/images/cable.svg' /></span></div>
              <div><span>{this.state.port}</span></div>
              </div>
            <div>
              <div>Peers <span><img src='/images/users.svg' /></span></div>
              <div><span>{this.state.peers}</span></div>
              </div>
          </div>
          <h2>Balances:</h2>
          <div className='stats'>
            <div>
              <div>Available <span><img src='/images/correct.svg' /></span></div>
              <div><span>{this.state.available}</span></div>
              </div>
            <div>
              <div>Pending <span><img src='/images/hourglass.svg' /></span></div>
              <div><span>{this.state.total - this.state.available}</span></div>
              </div>
            <div>
              <div>Total <span><img src='/images/the-sum-of.svg' /></span></div>
              <div><span>{this.state.total}</span></div>
              </div>
          </div>
        </main>

      </div>
    );
  }
  componentDidMount () {
    this.update();
  }
}

export default Overview;