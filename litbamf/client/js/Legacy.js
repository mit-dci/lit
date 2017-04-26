import lc from './LitClient';

class Legacy extends React.Component {
  constructor (props) {
    super(props);
    this.state = {
      numAddresses: 1,
      sendAddress: '',
      sendSatoshis: '',
    };
  }
  address () {
    lc.send('LitRPC.Address', this.state.numAddresses).then(res => {
      console.log(res);
    })
    .fail(err => {
      console.error(err);
    });
  }
  send () {
    lc.send('LitRPC.Send', this.state.sendAddress, this.state.sendSatoshis).then(res => {
      console.log(res);
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
    return (
      <div>
        <h2>Wallet Commands</h2>

        <button onClick={this.address}>address</button>
        <input type="text" id="numAddresses" value={this.state.numAddresses}
          onChange={this.handleChange}></input>
        <button onClick={this.send}>send</button>
        <input type="text" id="sendAddress" value={this.state.sendAddress}
          onChange={this.handleChange}></input>
        <input type="text" id="sendSatoshis" value={this.state.sendSatoshis}
          onChange={this.handleChange}></input>
      </div>
    );
  }
}

export default Legacy;