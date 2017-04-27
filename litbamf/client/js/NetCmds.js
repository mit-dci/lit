import lc from './LitClient';

class NetCmds extends React.Component {
  constructor (props) {
    super(props)
    this.state = {
      connectPubkey: '',
      connectHost: '',
      connectPort: '',
      sayText: '',
    };
  }
  connect () {
    let address = `${this.state.connectPubkey}@${this.state.connectHost}:${this.state.connectPort}`;
    lc.send('LitRPC.Connect', address).then(res => {
      console.log(res);
    })
    .fail(err => {
      console.error(err);
    });
  }
  listen () {
    lc.send('LitRPC.Listen').then(res => {
      console.log(res);
    })
    .fail(err => {
      console.error(err);
    });
  }
  say () {
    lc.send('LitRPC.Say').then(res => {
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
        <h2>Network Commands</h2>
        <button onClick={this.connect}>connect</button>
        <input type="text" id="connectPubkey" value={this.state.connectPubkey}
          onChange={this.handleChange}></input>
        <input type="text" id="connectHost" value={this.state.connectHost}
          onChange={this.handleChange}></input>
        <input type="text" id="connectPort" value={this.state.connectPort}
          onChange={this.handleChange}></input>

        <button onClick={this.listen}>listen</button>

        <button onClick={this.say}>say</button>
        <input type="text" id="sayText" value={this.state.sayText}
          onChange={this.handleChange}></input>
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

export default NetCmds;