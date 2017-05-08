import lc from './LitClient';

class NetCmds extends React.Component {
  constructor (props) {
    super(props);
    this.state = {
      connectPubkey: '',
      connectHost: '',
      connectPort: '',
      sayText: '',
    };
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

export default NetCmds;