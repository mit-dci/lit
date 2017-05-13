import {Navbar} from './Globals.js';
import lc from './LitClient';
import Q from 'q';
require('../sass/overview.scss');

class Overview extends Reflux.Component {
  constructor (props) {
    super(props);

    this.state =  {
      peers: '',
      port: null,
      balances: [],
    };
  }
  update () {
    Q.spread([
      lc.send('LitRPC.ListConnections'),
      lc.send('LitRPC.GetListeningPorts'),
      lc.send('LitRPC.Balance'),
    ], (connections, ports, balances) => {
      let peers = connections.Connections !== null ? connections.Connections.length : 0;
      let port = ports.LisIpPorts !== null ? 'listening on ' + ports.LisIpPorts[0] : null;
      balances = balances.Balances.map(bal => {
        return {
          coinType: bal.CoinType,
          legacy: bal.TxoTotal,
          channel: bal.ChanTotal,
          syncHeight: bal.SyncHeight,
        };
      }).filter(bal => bal.coinType == 65537);

      this.setState({
        peers: peers,
        port: port,
        balances: balances,
      });
    })
    .fail(err => {
      console.error(err);
    });
  }
  listen () {
    lc.send('LitRPC.Listen').then(res => {
      this.update();
    })
    .fail(err => {
      console.error(err);
    });
  }
  render () {
    let balances = this.state.balances.map(bal => {
      return (
        <div key={bal.coinType} className='stats'>
          <div>
            <div>Legacy <span><img src='/images/correct.svg' /></span></div>
            <div><span>{bal.legacy}</span></div>
          </div>
          <div>
            <div>Channel <span><img src='/images/hourglass.svg' /></span></div>
            <div><span>{bal.channel}</span></div>
          </div>
          <div>
            <div>Total <span><img src='/images/the-sum-of.svg' /></span></div>
            <div><span>{bal.legacy + bal.channel}</span></div>
          </div>
        </div>
      );
    });

    let syncHeight = this.state.balances.length > 0 ? this.state.balances[0].syncHeight : 0;

    return (
      <div>
        <Navbar page="overview" />

        <main>
          <div className='stats'>
            <div>
              <div><span>Block Height</span> <span><img src='/images/cube.svg' /></span></div>
              <div><span>{syncHeight}</span></div>
            </div>
            <div>
              <div><span>Listening Ports</span> <span><img src='/images/cable.svg' /></span></div>
              <div>
                <span>{this.state.port || 'off'}</span>
                <div>
                  <input id="listeningSelector" type="checkbox" className="tgl tgl-flat"
                    checked={this.state.port !== null ? 'checked' : ''}
                    onChange={this.listen.bind(this)} />
                  <label htmlFor="listeningSelector" className="tgl-btn"></label>
                </div>
              </div>
            </div>
            <div>
              <div><span>Peers</span> <span><img src='/images/users.svg' /></span></div>
              <div><span>{this.state.peers}</span></div>
            </div>
          </div>
          <h2>Balances:</h2>
          {balances}
        </main>

      </div>
    );
  }
  componentDidMount () {
    this.update();
  }
}

export default Overview;