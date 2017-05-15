import {Navbar} from './Globals.js';
import lc from './LitClient';
import Q from 'q';
require('../sass/channels.scss');

let Actions = Reflux.createActions(['setPeers', 'setChannels', 'setSelectedPeerIdx', 'setSelectedChannelIdx']);

class Store extends Reflux.Store {
  constructor () {
    super();

    let selectedPeerIdx = window.sessionStorage.selectedPeerIdx || -1;
    let selectedChannelIdx = window.sessionStorage.selectedChannelIdx || -1;

    this.state = {
      peers: {},
      channels: [],
      selectedPeerIdx: selectedPeerIdx,
      selectedChannelIdx: selectedChannelIdx,
    };

    this.listenables = Actions;
  }
  onSetPeers (peers) {
    this.setState({
      peers: peers,
    });
  }
  onSetChannels (channels) {
    this.setState({
      channels: channels,
    });
  }
  onSetSelectedPeerIdx (idx) {
    this.setState({
      setSelectedPeerIdx: idx,
    });
  }
  onSetSelectedChannelIdx (idx) {
    this.setState({
      setSelectedChannelIdx: idx,
    });
  }
}

class PeerModal extends Reflux.Component {
  constructor (props) {
    super(props);

    this.state = {
      address: '',
    };
    this.store = Store;
  }
  connect () {
    lc.send('LitRPC.Connect', {
      'LNAddr': this.state.address,
      'Nickname': this.state.nickname,
    }).then(res => {
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
      <section className="css-modal css-modal--transition--fade" id="peer-modal"
        data-stackable="false"
        role="dialog"
        aria-labelledby="label-fade"
        aria-hidden="true">

        <div className="css-modal_inner">
          <header className="css-modal_header">
            <h2 id="label-fade">Add Peer</h2>
          </header>

          <form action="javascript:void(0);" className="css-modal_content">
            <div>
              <input id="address" type="text" placeholder="pubkeyhash@hostname:port"
                value={this.state.address} onChange={this.handleChange.bind(this)}></input>
            </div>
            <div><button type="submit" onClick={this.connect.bind(this)}>Go</button></div>
          </form>
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
    this.store = Store;
  }
  nickname () {
    lc.send('LitRPC.AssignNickname', {
      'Peer': parseInt(this.state.selectedPeerIdx),
      'Nickname': this.state.nickname,
    }).then(res => {
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
        role="dialog"
        aria-labelledby="label-fade"
        aria-hidden="true">

        <div className="css-modal_inner">
          <header className="css-modal_header">
            <h2 id="label-fade">Edit Nicknamer</h2>
          </header>

          <form action="javascript:void(0);" className="css-modal_content">
            <div>
              <input id="nickname" type="text" placeholder="nickname"
                value={this.state.nickname} onChange={this.handleChange.bind(this)}></input>
            </div>
            <div><button type="submit" onClick={this.nickname.bind(this)}>Save</button></div>
          </form>
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

    this.store = Store;
  }
  update () {
    lc.send('LitRPC.ListConnections').then(connections => {
      connections = connections.Connections !== null ? connections.Connections : [];
      let peers = {};
      connections.forEach(conn => {
        peers[conn.PeerNumber] = conn;
      });
      Actions.setPeers(peers);
    })
    .fail(err => {
      console.error(err);
    });
  }
  editNickname (idx) {
    this.setState({selectedPeerIdx: idx});
    window.sessionStorage.selectedPeerIdx = idx;
    window.location = '#nickname-modal';
  }
  changePeer (event) {
    Actions.setSelectedPeerIdx(event.target.value);
  }
  render () {
    let peerElements = Object.keys(this.state.peers).map(key => {
      let peer = this.state.peers[key];
      let idx = peer.PeerNumber;
      return (
        <li key={idx}>
          <label className={this.state.selectedPeerIdx == idx ? 'checked' : ''}>
            <span>{peer.Nickname}</span>
            <input type="radio" onChange={this.changePeer.bind(this)} name="peer" value={idx} />
          </label>
          <div>
            <button onClick={this.editNickname.bind(this, idx)}>
              <i className="material-icons">mode_edit</i>
            </button>
          </div>
        </li>
      );
    });

    return (
      <div id="peerList">
        <ul>
          {peerElements}
        </ul>
        <div id="peer-add">
          <a href="#peer-modal">+</a>
        </div>
      </div>
    );
  }
  componentDidMount () {
    this.update();
  }
}

class ChannelElement extends Reflux.Component {
  constructor (props) {
    super(props);

    this.state = {
      pushAmount: 0,
    };
  }
  push () {
    lc.send('LitRPC.Push', {
      'ChanIdx': parseInt(this.props.idx),
      'Amt': parseInt(this.state.pushAmount),
    })
    .then(res => {
      this.state.pushAmount = 0;
      this.props.update();
    })
    .fail(err => {
      console.error(err);
    });
  }
  changePushAmount (event) {
    this.setState({
      pushAmount: event.target.value,
    });
  }
  xtraCommands (idx) {
    Actions.setSelectedChannelIdx(this.props.idx);
    window.sessionStorage.selectedChannelIdx = this.props.idx;
    window.location = '#xtra-modal';
  }
  render () {
    return (
      <tr>
        <td className="chan-capacity">{this.props.capacity}</td>
        <td className="chan-balance">{this.props.balance}</td>
        <td className="chan-state">{this.props.state}</td>
        <td className="chan-zap">
          <form action="javascript:void(0);">
            <input type="number" placeholder="amount"
              onChange={this.changePushAmount.bind(this)}
              value={this.state.pushAmount}></input>
            <button type="submit" onClick={this.push.bind(this)}>Zap</button>
          </form>
        </td>
        <td className="chan-xtra">
          <button onClick={this.xtraCommands.bind(this)}>X</button>
        </td>
      </tr>
    );
  }
}

class ChannelList extends Reflux.Component {
  constructor (props) {
    super(props);

    this.store = Store;
  }
  update () {
    lc.send('LitRPC.ChannelList').then(_channels => {
      let channels = _channels.Channels !== null ? _channels.Channels : [];
      channels = channels.filter(chan => chan.PeerIdx == this.state.selectedPeerIdx);
      Actions.setChannels(channels);
    })
    .fail(err => {
      console.error(err);
    });
  }
  render () {
    let channelElements = this.state.channels.map((chan, i) => {
      return <ChannelElement key={i} idx={chan.CIdx} capacity={chan.Capacity}
        balance={chan.MyBalance} state={chan.StateNum} update={this.update.bind(this)}/>;
    });


    return (
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
        <div id="channel-add">
          <a href="#channel-modal">+</a>
        </div>
      </div>
    );
  }
  componentDidMount () {
    this.update();
  }
}

class ChannelModal extends Reflux.Component {
  constructor (props) {
    super(props);

    this.state = {
      capacity: '',
      initial: '',
    };
    this.store = Store;
  }
  connect () {
    lc.send('LitRPC.FundChannel', {
      'Peer': parseInt(this.state.selectedPeerIdx),
      'Capacity': parseInt(this.state.capacity),
      'Initial': parseInt(this.state.initial),
    })
    .then(res => {
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
      <section className="css-modal css-modal--transition--fade" id="channel-modal"
        role="dialog"
        aria-labelledby="label-fade"
        aria-hidden="true">

        <div className="css-modal_inner">
          <header className="css-modal_header">
            <h2 id="label-fade">Open Channel</h2>
          </header>

          <form action="javascript:void(0);" className="css-modal_content">
            <div>
              <input id="capacity" type="text" placeholder="channel capacity"
                value={this.state.capacity} onChange={this.handleChange.bind(this)}></input>
              <input id="initial" type="text" placeholder="initial transfer"
                value={this.state.initial} onChange={this.handleChange.bind(this)}></input>
            </div>
            <div><button type="submit" onClick={this.connect.bind(this)}>Go</button></div>
          </form>
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

class XtraModal extends Reflux.Component {
  constructor (props) {
    super(props);

    this.store = Store;
  }
  closeChannel () {
    lc.send('LitRPC.CloseChannel', {
      'ChanIdx': parseInt(this.state.selectedChannelIdx),
    })
    .then(res => {
      window.location = window.location.href.split('#')[0];
    })
    .fail(err => {
      console.error(err);
    });
  }
  breakChannel () {
    let justice = window.confirm('Are you sure you want to break the channel?');
    if(justice) {
      lc.send('LitRPC.BreakChannel', {
        'ChanIdx': parseInt(this.state.selectedChannelIdx),
      })
      .then(res => {
        window.location = window.location.href.split('#')[0];
      })
      .fail(err => {
        console.error(err);
      });
    }
  }
  render () {
    return (
      <section className="css-modal css-modal--transition--fade" id="xtra-modal"
        role="dialog"
        aria-labelledby="label-fade"
        aria-hidden="true">

        <div className="css-modal_inner">
          <header className="css-modal_header">
            <h2 id="label-fade">Xtra Commands</h2>
          </header>

          <div className="css-modal_content">
            <div>
              <p>If the other party is online then you can close the channel
                and you funds will be available shortly.</p>
              <div><button onClick={this.closeChannel.bind(this)}>Close</button></div>
            </div>
            <div>
              <p>At anytime you can break the channel, but your funds will be
                locked for a long time.</p>
              <div><button onClick={this.breakChannel.bind(this)}>Break</button></div>
            </div>
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

class Chatbox extends Reflux.Component {
  constructor (props) {
    super(props);

    this.state = {
      message: '',
      conversations: {},
    };
    this.store = Store;
  }
  say () {
    lc.send('LitRPC.Say', {
      'Peer': parseInt(this.state.selectedPeerIdx),
      'Message': this.state.message,
    }).then(res => {
      let date = new Date();
      let time = date.getHours() + ':' + date.getMinutes();

      let conversations = this.state.conversations;
      let conversation = conversations[this.state.selectedPeerIdx] || [];
      conversation.push({
        isMyMessage: true,
        name: 'You',
        message: this.state.message,
        time: time,
      });
      conversations[this.state.selectedPeerIdx] = conversation;
      this.setState({conversations: conversations});
      this.state.message = '';
    })
    .fail(err => {
      console.error(err);
    });
  }
  chatHandler (message) {
    let peer = this.state.peers[message.PeerIdx];
    //if we receive a message before the peers list is updated
    //it can't be handled yet
    if(peer === undefined) {
      return;
    }

    let date = new Date();
    let time = date.getHours() + ':' + date.getMinutes();

    let conversations = this.state.conversations;
    let conversation = conversations[message.PeerIdx] || [];
    conversation.push({
      isMyMessage: false,
      name: peer.Nickname,
      message: message.Text,
      time: time,
    });
    conversations[message.PeerIdx] = conversation;
    this.setState({conversations: conversations});
  }
  changeMessage (event) {
    this.setState({message: event.target.value});
  }
  render () {

    let placeholder = 'Message ';
    let peer = this.state.peers[this.state.selectedPeerIdx];
    placeholder += peer === undefined ? '' : peer.Nickname;

    let selectedConvo = this.state.conversations[this.state.selectedPeerIdx] || [];
    let chat = selectedConvo.map(convo => {
      return (
        <div className={convo.isMyMessage ? 'myMessage' : ''}>
          <span className="nickname">[{convo.time}] {convo.name}:</span>
          <span className="message">{convo.message}</span>
        </div>
      );
    });

    return (
      <div id="chatbox">
        <div>
          {chat}
        </div>
        <form action="javascript:void(0);">
          <input placeholder={placeholder} onChange={this.changeMessage.bind(this)} value={this.state.message} />
          <button type="submit" onClick={this.say.bind(this)} hidden></button>
        </form>
      </div>
    );
  }
  componentDidMount () {
    lc.register(null, this.chatHandler.bind(this));
  }
}

class Channels extends Reflux.Component {
  constructor (props) {
    super(props);
    this.state = {
      address: '',
      listening: false,
    };
  }
  getListeningPorts () {
    lc.send('LitRPC.GetListeningPorts').then(ports => {
      this.setState({
        listening: ports.LisIpPorts !== null,
        address: ports.Adr,
      });
    });
  }
  listen () {
    //don't need to bother making the call if we are already listening
    if(this.state.listening) {
      return;
    }

    lc.send('LitRPC.Listen').then(res => {
      this.getListeningPorts();
    })
    .fail(err => {
      window.alert(err);
    });
  }
  render () {
    return (
      <div>
        <Navbar page="channels" />
        <div id="identity">
          <h4>Your Channel identity: </h4>
          <h4>{this.state.address}</h4>

          <div>
            <h4>Listening for connections: </h4>
            <input id="listeningSelector" type="checkbox" className="tgl tgl-flat"
              checked={this.state.listening ? 'checked' : ''}
              onChange={this.listen.bind(this)} />
            <label htmlFor="listeningSelector" className="tgl-btn"></label>
          </div>
        </div>
        <div>
          <PeerList />
          <div id="boxes">
            <ChannelList />
            <Chatbox />
          </div>
        </div>

        <PeerModal />
        <NicknameModal />
        <ChannelModal />
        <XtraModal />
      </div>
    );
  }
  componentDidMount () {
    this.getListeningPorts();
  }
}

export default Channels;
