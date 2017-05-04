import {Navbar} from './Globals.js';
import lc from './LitClient';
import Q from 'q';
require('../sass/channels.scss');

let Actions = Reflux.createActions([])

class PeerStore extends Reflux.Store {
  constructor () {
    super();

    let selectedIdx = window.sessionStorage.selectedPeerIdx || -1;

    this.state = {
      peers: [],
      selectedIdx: selectedIdx,
    };
  }
}

class PeerModal extends Reflux.Component {
  constructor (props) {
    super(props);

    this.state = {
      address: '',
    };
    this.store = PeerStore;
  }
  connect () {
    lc.send('LitRPC.Connect', {'LNAddr': this.state.address, 'Nickname': this.state.nickname}).then(res => {
      window.location = window.location.href.split('#')[0];
      this.state.peers.append({
        address: this.state.address,
        nickname: this.state.nickname,
        channels: [],
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
    return (
      <section className="css-modal css-modal--transition--fade" id="peer-modal"
        data-stackable="false"
        role="dialog"
        aria-labelledby="label-fade"
        aria-hidden="true">

        <div className="css-modal_inner">
          <header className="css-modal_header">
            <h2 id="label-fade">Add Peer</h2>
          </header>

          <div className="css-modal_content">
            <div>
              <input id="address" type="text" placeholder="pubkeyhash@hostname:port"
                value={this.state.address} onChange={this.handleChange.bind(this)}></input>
            </div>
            <div><button onClick={this.connect.bind(this)}>Go</button></div>
          </div>
          <div>
            <a href="#!" className="css-modal_close css-modal_close_button"
              title="Close this modal">&times;</a>
          </div>
        </div>
        <a href="#!" className="css-modal_close css-modal_close_area"
          title="Close this modal">&times;</a>
      </section>
    );
  }
}

class NicknameModal extends Reflux.Component {
  constructor (props) {
    super(props);

    this.state = {
      nickname: '',
    };
    this.store = PeerStore;
  }
  nickname () {
    console.log(this.state.selectedIdx);
    lc.send('LitRPC.AssignNickname', {'Peer': parseInt(this.state.selectedIdx), 'Nickname': this.state.nickname}).then(res => {
      window.location = window.location.href.split('#')[0];
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
      <section className="css-modal css-modal--transition--fade" id="nickname-modal"
        data-stackable="false"
        role="dialog"
        aria-labelledby="label-fade"
        aria-hidden="true">

        <div className="css-modal_inner">
          <header className="css-modal_header">
            <h2 id="label-fade">Edit Nicknamer</h2>
          </header>

          <div className="css-modal_content">
            <div>
              <input id="nickname" type="text" placeholder="nickname"
                value={this.state.nickname} onChange={this.handleChange.bind(this)}></input>
            </div>
            <div><button onClick={this.nickname.bind(this)}>Save</button></div>
          </div>
          <div>
            <a href="#!" className="css-modal_close css-modal_close_button"
              title="Close this modal">&times;</a>
          </div>
        </div>
        <a href="#!" className="css-modal_close css-modal_close_area"
          title="Close this modal">&times;</a>
      </section>
    );
  }
}

class PeerList extends Reflux.Component {
  constructor (props) {
    super(props);

    this.store = PeerStore;
  }
  update () {
    lc.send('LitRPC.ListConnections').then(connections => {
      let peers = connections.Connections !== null ? connections.Connections : [];
      console.log(peers);
      this.setState({
        peers: peers,
      });
    })
    .fail(err => {
      console.error(err);
    });
  }
  editNickname (idx) {
    this.setState({selectedIdx: idx});
    window.sessionStorage.selectedPeerIdx = idx;
    window.location = '#nickname-modal';
  }
  render () {
    let peerElements = this.state.peers.map((peer, i) => {
      let idx = i + 1;
      return (
        <li>
          <button>
            <span>{peer.Nickname}</span>
            <button onClick={this.editNickname.bind(this, idx)}>
              <i className="material-icons">mode_edit</i>
            </button>
          </button>
        </li>
      );
    });

    return (
      <ul id="peerList">
        {peerElements}
        <li id="peer-add">
          <a href="#peer-modal">+</a>
        </li>
      </ul>
    );
  }
  componentDidMount () {
    this.update();
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
    let channelElements = [<ChannelElement key="dumchan" capacity="lots o money" balance="less money" state="42"/>];

    return (
      <div>
        <Navbar />

        <div id="chanbox">
          <PeerList />
          <div id="channelList">
            <table>
              <thead>
                <tr>
                  <th className="chan-capacity">Capacity</th>
                  <th className="chan-balance">Balance</th>
                  <th className="chan-state">State</th>
                  <th className="chan-zap">Zap Funds to Channel</th>
                  <th className="chan-xtra"> X-tra Commands</th>
                </tr>
              </thead>
              <tbody>
                {channelElements}
              </tbody>
            </table>
          </div>
        </div>
        <div id="chatbox">
        </div>

        <PeerModal />
        <NicknameModal />

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

export default ChanCmds;