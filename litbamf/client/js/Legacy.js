import Q from 'q';
import lc from './LitClient';
import {Navbar} from './Globals.js';
require('../sass/legacy.scss');

let Actions = Reflux.createActions(['setTransactions', 'setAddresses']);

class Store extends Reflux.Store {
  constructor () {
    super();

    this.state = {
      transactions: [],
      addresses: {
        legacy: [],
        wit: [],
      },
    };

    this.listenables = Actions;
  }
  onSetTransactions (txs) {
    this.setState({
      transactions: txs,
    });
  }
  onSetAddresses (legacy, wit) {
    this.setState({
      addresses: {
        legacy: legacy,
        wit: wit,
      },
    });
  }
}

class ActionList extends Reflux.Component {
  constructor (props) {
    super(props);

    this.store = Store;
  }
  update () {
    Q.spread([
      lc.send('LitRPC.TxoList'),
      lc.send('LitRPC.Address', {'NumToMake': 0}),
    ], (txs, addrs) => {
      txs = txs.Txos || [];
      addrs = addrs || [];
      let transactions = txs.map(tx => {
        return tx.OutPoint.split(';')[0];
      });

      Actions.setTransactions(transactions.reverse());
      Actions.setAddresses(
        addrs.LegacyAddresses.reverse(),
        addrs.WitAddresses.reverse()
      );
    })
    .fail(err => {
      console.error(err);
    });
  }
  render () {
    let txList = this.state.transactions.map(tx => {
      return (
        <p key={tx}>{tx}</p>
      );
    });
    let legacyList = this.state.addresses.legacy.map(addr => {
      return (
        <p key={addr}>{addr}</p>
      );
    });

    let witList = this.state.addresses.wit.map(addr => {
      return (
        <p key={addr}>{addr}</p>
      );
    });
    return (
      <div id="listsbox">
        <div id="transactionbox">
          <h3>Transactions</h3>
          <div>{txList}</div>
        </div>
        <div id="addressbox">
          <div>
            <h3>Legacy Addresses</h3>
            <div>{legacyList}</div>
          </div>
          <div>
            <h3>Lightning Addresses</h3>
            <div>{witList}</div>
          </div>
        </div>
      </div>
    );
  }
  componentDidMount () {
    this.update();
  }
}

class Legacy extends Reflux.Component {
  constructor (props) {
    super(props);
    this.state = {
      sendAddress: '',
      sendSatoshis: '',
      balances: [],
    };

    this.store = Store;
  }
  update () {
    lc.send('LitRPC.Balance').then(balances => {
      balances = balances.Balances.map(bal => {
        return {
          coinType: bal.CoinType,
          legacy: bal.TxoTotal,
          channel: bal.ChanTotal,
          syncHeight: bal.SyncHeight,
        };
      }).filter(bal => bal.coinType == 65537);

      this.setState({
        balances: balances,
      });
    })
    .fail(err => {
      console.error(err);
    });
  }
  address () {
    lc.send('LitRPC.Address', {'NumToMake': 1}).then(addrs => {
      Actions.setAddresses([
        ...addrs.LegacyAddresses,
        ...this.state.addresses.legacy,
      ], [
        ...addrs.WitAddresses,
        ...this.state.addresses.wit,
      ]);
    })
    .fail(err => {
      console.error(err);
    });
  }
  send () {
    console.log(this.state);
    lc.send(
      'LitRPC.Send', {
        'DestAddrs': [this.state.sendAddress],
        'Amts': [parseInt(this.state.sendSatoshis)],
      }
    )
    .then(txs => {
      Actions.setTransactions([
        ...txs.Txids,
        ...this.state.transactions,
      ]);

      this.setState({
        sendAddress: '',
        sendSatoshis: '',
      });
    })
    .fail(err => {
      console.error(err);
    });
  }
  handleChange (event) {
    let state = {};
    state[event.target.id] = event.target.value;
    this.setState(state);
  }
  render () {
    let balances = this.state.balances.map(bal => {
      return (
        <div key={bal.coinType} id='balances'>
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

    return (
      <div>
        <Navbar page="legacy" />

        <h2>Balances:</h2>
        {balances}

        <h2>Wallet Commands</h2>
        <div id="walletbox">
          <div>
            <h3>Get a new address</h3>
            <div>
              <button onClick={this.address.bind(this)}>+</button>
            </div>
          </div>
          <div>
            <h3>Send Transaction</h3>
            <label>
              <span>Pay to</span>
              <input type="text" id="sendAddress" value={this.state.sendAddress}
                placeholder="address"
                onChange={this.handleChange.bind(this)}></input>
            </label>
            <label>
              <span>Amount</span>
              <input type="text" id="sendSatoshis" value={this.state.sendSatoshis}
                placeholder="amount"
                onChange={this.handleChange.bind(this)}></input>
            </label>
            <div>
              <button onClick={this.send.bind(this)}>send</button>
            </div>
          </div>
        </div>

        <ActionList key={this.state.actionTrigger} />
      </div>
    );
  }
  componentDidMount () {
    this.update();
  }
}

export default Legacy;
