import {Navbar} from './Globals.js';
import lc from './LitClient';
import Q from 'q';
require('../sass/channels.scss');

class PeerElement extends React.Component {
  constructor (props) {
    super(props)
  }
  render () {
    return (
      <li>
        {this.props.name}
      </li>
    )
  }
}

class ChannelElement extends React.Component {
  constructor (props) {
    super(props)
  }
  render () {
    return (
      <tr>
        <td className="chan-capacity">{this.props.capacity}</td>
        <td className="chan-balance">{this.props.balance}</td>
        <td className="chan-state">{this.props.state}</td>
        <td className="chan-zap">
          <input></input>
          <button>Zap</button>
        </td>
        <td className="chan-xtra">
          <button >X</button>
        </td>
      </tr>
    );
  }
}

class ChanCmds extends React.Component {
  constructor (props) {
    super(props);
    this.state = {
      fundPeer: 0,
      fundCapacity: 1,
      fundInitialSend: 1,
      pushChannel: 0,
      pushAmount: 1,
      pushTimes: 1,
      closeChannel: '',
      breakChannel: '',
    };
  }
  fund () {
    lc.send('LitRPC.Fund', this.state.fundPeer,
    this.state.fundCapacity, this.state.fundInitialSend)
    .then(res => {
      console.log(res);
    })
    .fail(err => {
      console.error(err);
    });
  }
  push () {
    lc.send('LitRPC.Push', this.state.pushChannel,
    this.state.pushAmount, this.state.pushTimes)
    .then(res => {
      console.log(res);
    })
    .fail(err => {
      console.error(err);
    });
  }
  CloseChannel () {
    lc.send('LitRPC.CloseChannel', this.state.closeChannel)
    .then(res => {
      console.log(res);
    })
    .fail(err => {
      console.error(err);
    });
  }
  breakChannel () {
    lc.send('LitRPC.BreakChannel', this.state.breakChannel)
    .then(res => {
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
    let peerElements = [<PeerElement name="dummyname"/>];
    let channelElements = [<ChannelElement capacity="lots o money" balance="less money" state="42"/>];

    return (
      <div>
        <Navbar />

        <div id="chanbox">
          <ul id="peerList">
            {peerElements}
          </ul>
          <div id="channelList">
            <table>
              <tr>
                <th className="chan-capacity">Capacity</th>
                <th className="chan-balance">Balance</th>
                <th className="chan-state">State</th>
                <th className="chan-zap">Zap Funds to Channel</th>
                <th className="chan-xtra"> X-tra Commands</th>
              </tr>
              {channelElements}
            </table>
          </div>
        </div>
        <div id="chatbox">

        </div>
        <button onClick={this.fund}>fund</button>
        <input type="text" id="fundPeer" value={this.state.fundPeer}
          onChange={this.handleChange}></input>
        <input type="text" id="fundCapacity" value={this.state.fundCapacity}
          onChange={this.handleChange}></input>
        <input type="text" id="fundInitialSend" value={this.state.fundInitialSend}
          onChange={this.handleChange}></input>
        <button onClick={this.push}>push</button>
        <input type="text" id="pushChannel" value={this.state.pushChannel}
          onChange={this.handleChange}></input>
        <input type="text" id="pushAmount" value={this.state.pushAmount}
          onChange={this.handleChange}></input>
        <input type="text" id="pushTimes" value={this.state.pushTimes}
          onChange={this.handleChange}></input>
        <button onClick={this.close}>close</button>
        <input type="text" id="closeChannel" value={this.state.closeChannel}
          onChange={this.handleChange}></input>
        <button onClick={this.break}>break</button>
        <input type="text" id="breakChannel" value={this.state.breakChannel}
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

export default ChanCmds;