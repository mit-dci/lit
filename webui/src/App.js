import React, {Component} from 'react';
import CssBaseline from 'material-ui/CssBaseline';
import LitAfClient from './LitClient'
import LitAppBar from './LitAppBar'
import Balances from './Balances'
import Channels from './Channels'
import Contracts from './Contracts'
import {coinInfo} from './CoinTypes'
import ErrorDialog from './ErrorDialog'


class App extends Component {

  constructor(props) {
    super(props);

    let host = "127.0.0.1";
    let port = 8001;
    let queryHost = this.getParameterByName("host");
    let queryPort = this.getParameterByName("port");

    if(queryHost) host = queryHost;
    if(queryPort) port = parseInt(queryPort, 10);

    this.state = {
      lc: null,
      rpcAddress: host,
      rpcPort: port,
      rpcRefresh: true,
      rpcRefreshReference: -1,
      appBarColorPrimary: true,
      hideClosedChannels: true,
      errorMessage: null,

      Connections: [],
      MyPKH: "",
      Channels: [],
      Adr: "",
      LisIpPorts: null,
      Txos: [],
      Balances: [],
      Contracts: [],
      Oracles: [],
      Assets: [],
      Offers: [],
      CoinRates: {},
    };
  }

   getParameterByName(name, url) {
    if (!url) url = window.location.href;
    name = name.replace(/[\[\]]/g, "\\$&");
    var regex = new RegExp("[?&]" + name + "(=([^&#]*)|&|#|$)"),
      results = regex.exec(url);
    if (!results) return null;
    if (!results[2]) return '';
    return decodeURIComponent(results[2].replace(/\+/g, " "));
  }

  /*
   * Error Handling
   */
  displayError(errorMessage) {
    this.setState({errorMessage: errorMessage});
  }

  handleErrorDialogSubmit() {
    this.setState({errorMessage: null});
  }

  /*
   * Update all the UI states by calling individual async updates
   * Note that we unpack the replies into their individual keyword items
   */
  update() {
    this.updateLit();
    this.updateOraclesAndAssets();
    this.updateCoinRates();
  }

  updateLit() {
    this.updateListConnections();
    this.updateChannelList();
    this.updateListeningPorts();
    this.updateTxoList();
    this.updateBalances();
    this.updateContractList();
    this.updateOfferList();
  }

  updateCoinRates() {
    var list = '';
    for (let i in coinInfo) {
      if(list.indexOf(coinInfo[i].exchangeSymbol) > -1) continue;
      if(list !== '') list += ',';
      list += coinInfo[i].exchangeSymbol;
    }

    fetch("https://min-api.cryptocompare.com/data/price?fsym=USD&tsyms=" + list)
      .then(res => res.json())
      .then((result) => {
        var coinRates = {};
        for (let i in coinInfo) {
          coinRates[i] = (1 / result[coinInfo[i].exchangeSymbol]) / 100000000 * coinInfo[i].factor;
        }
        this.setState({ CoinRates : coinRates });
      });
  }

  updateListConnections() {
    this.state.lc.send('LitRPC.ListConnections')
      .then(reply => {
        let connections = reply.Connections !== null ? reply.Connections : [];
        this.setState(
          {
            Connections: connections,
            MyPKH: reply.MyPKH,
          });
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  updateChannelList() {
    this.state.lc.send('LitRPC.ChannelList')
      .then(reply => {
        let channels = reply.Channels !== null ? reply.Channels : [];
        // channels = channels.filter(chan => chan.PeerIdx == this.state.selectedPeerIdx);
        this.setState({Channels: channels});
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  updateContractList() {
    this.state.lc.send('LitRPC.ListContracts')
      .then(reply => {
        let contracts = reply.Contracts !== null ? reply.Contracts : [];
        this.setState({Contracts: contracts});
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  updateOfferList() {
    this.state.lc.send('LitRPC.ListOffers')
      .then(reply => {
        let offers = reply.Offers !== null ? reply.Offers : [];
        this.setState({Offers: offers});
        // console.log("Offers", offers);
      })
      .catch(err => {
        this.displayError(err);
      });
  }
  fetchAssetValue(asset) {
    var oracle = this.state.Oracles.find(o => o.Idx === asset.oracleId);
    if(oracle === null || oracle === undefined) return null;

    return fetch(oracle.Url + "api/datasource/" + asset.datafeedId + "/value")
    .then(res => res.json())
    .then(res => {
      if(res.valueError !== null && res.valueError !== undefined) {
        throw new Error("Error fetching value: " + res.valueError);
      }
      return res.currentValue
    });
  }

  // Every data feed of every oracle is considered an asset rn.
  // Might need a more elegant solution for other types of oracle
  // values later
  refreshAssets() {
    const oracles = this.state.Oracles;
    var assets = [];
    for(var o of oracles) {
      if(o.Url !== null) {
        // fetch data feeds for this oracle
        fetch(o.Url + "api/datasources")
        .then(res => res.json())
        .then((result) => {
          // console.log(result);
          for(var f of result) {
            assets.push( {
              name : f.name,
              oracleId : o.Idx,
              datafeedId : f.id
            });

          }
          // console.log(assets);
          this.setState({Assets: assets});
        });
      }
    }

  }

  updateOraclesAndAssets() {
    this.state.lc.send('LitRPC.ListOracles')
      .then(reply => {
        let oracles = reply.Oracles !== null ? reply.Oracles : [];
        this.setState({Oracles: oracles});
        this.refreshAssets();
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  updateListeningPorts() {
    this.state.lc.send('LitRPC.GetListeningPorts')
      .then(reply => {
        let adr = reply.Adr !== null ? reply.Adr : "";
        this.setState({Adr: adr, LisIpPorts: reply.LisIpPorts});
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  updateTxoList() {
    this.state.lc.send('LitRPC.TxoList')
      .then(reply => {
        let txos = reply.Txos !== null ? reply.Txos : [];
        this.setState({Txos: txos});
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  updateBalances() {
    this.state.lc.send('LitRPC.Balance')
      .then(reply => {
        let balances = reply.Balances !== null ? reply.Balances : [];
        this.setState({Balances: balances});
      })
      .catch(err => {
        this.displayError(err);
      });
  }


  listen() {
    this.state.lc.send('LitRPC.Listen')
      .then(reply => {
        this.updateListeningPorts()
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  /*
   * Create 1 or more new addresses of a given coinType. Returns a Promise
   */
  address(numToMake, coinType) {
    return new Promise((resolve, reject) => {
      this.state.lc.send('LitRPC.Address', {
        'NumToMake': numToMake,
        'CoinType': coinType,
      }).then(reply => {
          resolve(reply);
        }
      )
        .catch(err => {
          this.displayError(err);
        });
    });
  }

  /*
   * Connect to a previously connected peer by giving its index, e.g. con 2 in lit-af
   */
  connectByIndex(index) {
    this.state.lc.send('LitRPC.Connect', {
      'LNAddr': index.toString()
    })
      .then(reply => {
        this.updateListConnections()
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  /*
   * attempt to reconnect any disconnected peers associated with an open channel
   */
  openChannelConnections(channels, connections) {
    channels.forEach(channel => {
        if (!channel.Closed) {
          let connection = connections.find(e => {
            return (e.PeerNumber === channel.PeerIdx);
          });
          // if we can't find the connection then it's not open so we can try to reopen
          if (connection === null) {
            this.connectByIndex(channel.PeerIdx);
          }
        }
      }
    );
  }


  hexStringToByte(str) {
      if (!str) {
        return [];
      }
      
      var a = [];
      for (var i = 0, len = str.length; i < len; i+=2) {
        a.push(parseInt(str.substr(i,2),16));
      }
      
      return a;
  }

  /*
   * click handler for funding a new channel
   */
  handleChannelAddSubmit(peerIdx, coinType, amount, data) {
    this.state.lc.send('LitRPC.FundChannel', {
      'Peer': parseInt(peerIdx, 10),
      'CoinType': coinType,
      'Capacity': parseInt(amount, 10),
      'Data': this.hexStringToByte(data),
    })
      .then(reply => {
        this.updateChannelList();
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  /*
   * click handler for adding new peer
   */
  handlePeerAddSubmit(address) {
    this.state.lc.send('LitRPC.Connect', {
      'LNAddr': address,
    })
      .then(reply => {
        this.updateListConnections();
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  /*
   * click handler for assigning a nickname
   */
  handlePeerNicknameSubmit(peerId, nickname) {
      this.state.lc.send('LitRPC.AssignNickname', {
        'Peer': peerId,
        'Nickname': nickname,
      })
        .then(reply => {
          this.updateListConnections();
        })
        .catch(err => {
          this.displayError(err);
        });
    }

  /*
 * click handler for traditional send to address
 */
  handleSendSubmit(address, amount) {
    // console.log("Send " + address + " " + amount);
    this.state.lc.send('LitRPC.Send', {
      'DestAddrs': [address],
      'Amts': [amount],
    })
      .then(reply => {
        this.updateBalances();
      })
      .catch(err => {
        this.displayError(err);
      });
  }

  /*
   * click handler for channel commands: push, close, break
   * amount is optional and only used for push
   */
  handleChannelCommand(channel, command, amount, data) {
    switch (command) {
      case 'push':
        this.state.lc.send('LitRPC.Push', {
          'ChanIdx': channel.CIdx,
          'Amt': amount,
          'Data': this.hexStringToByte(data),
        })
          .then(reply => {
            this.updateBalances();
            this.updateChannelList();
          })
          .catch(err => {
            this.displayError(err);
          });
        break;
      case 'close':
        // console.log("channel close command!");
        this.state.lc.send('LitRPC.CloseChannel', {
          'ChanIdx': channel.CIdx,
        })
          .then(reply => {
            this.updateChannelList();
          })
          .catch(err => {
            this.displayError(err);
          });
        break;
      case 'break':
        // console.log("channel break command!");
        this.state.lc.send('LitRPC.BreakChannel', {
          'ChanIdx': channel.CIdx,
        })
          .then(reply => {
            this.updateChannelList();
          })
          .catch(err => {
            this.displayError(err);
          });
        break;
      default:
        this.displayError("Unrecognized channel command " + command);
    }
  }

  /*
   * click handler for new future contract
   */
  handleCreateContract(selling, assetIdx, amount, price, settleTime, peerIdx) {
    // Fetch R-point

    let promise = this.fetchOracleKeysForAsset(this.state.Assets[assetIdx], settleTime);
    if(promise === null) throw new Error("Cannot determine R-Point");

    var dlcFwdOffer = {};
    promise.then(res => {
      dlcFwdOffer.OracleA = res[0];
      dlcFwdOffer.OracleR = res[1];
      dlcFwdOffer.SettlementTime = settleTime;
      dlcFwdOffer.ImBuyer = !selling;
      dlcFwdOffer.AssetQuantity = amount;
      dlcFwdOffer.FundAmt = amount * price;
      dlcFwdOffer.PeerIdx = peerIdx;
      dlcFwdOffer.CoinType = 257;

      this.state.lc.send('LitRPC.NewForwardOffer', { Offer : dlcFwdOffer })
    });
  }

  fetchOracleKeysForAsset(asset, timeUnix) {
    let oracle = this.state.Oracles.find(o => o.Idx === asset.oracleId);
    if(oracle === null || oracle === undefined) return null;

    return fetch(oracle.Url + "api/rpoint/" + asset.datafeedId + "/" + timeUnix.toString())
    .then(res => res.json())
    .then(res => {
      var buf = Buffer.from(res.R,'hex');
      var rPoint = [];
      for (var i = 0; i < buf.length; i++) rPoint[i] = buf[i];

      return [oracle.A, rPoint];
    });
  }

  /*
   * click handler for new contract (old)
   */
  handleContractAddSubmit(oracleIdx, settlementTime, dataFeedId, fundingOurs, fundingTheirs, valueAllOurs, valueAllTheirs, coinType) {
    this.state.lc.send('LitRPC.NewContract', {})
    .then(c => {
      this.state.lc.send('LitRPC.SetContractOracle', {
        'CIdx' : c.Contract.Idx,
        'OIdx' : parseInt(oracleIdx, 10)
      })
      .then(reply => {
        this.state.lc.send('LitRPC.SetContractSettlementTime', {
          'CIdx' : c.Contract.Idx,
          'Time' : parseInt(settlementTime, 10)
        })
        .then(reply => {
          this.state.lc.send('LitRPC.SetContractDatafeed', {
            'CIdx' : c.Contract.Idx,
            'Feed' : parseInt(dataFeedId, 10)
          })
          .then(reply => {
            this.state.lc.send('LitRPC.SetContractFunding', {
              'CIdx' : c.Contract.Idx,
              'OurAmount' : fundingOurs,
              'TheirAmount' : fundingTheirs
            })
            .then(reply => {
              this.state.lc.send('LitRPC.SetContractDivision', {
                'CIdx' : c.Contract.Idx,
                'ValueFullyOurs' : parseInt(valueAllOurs, 10),
                'ValueFullyTheirs' : parseInt(valueAllTheirs, 10)
              })
              .then(reply => {
                this.state.lc.send('LitRPC.SetContractCoinType', {
                  'CIdx' : c.Contract.Idx,
                  'CoinType' : parseInt(coinType, 10)
                })
                .then(reply => {
                  this.updateContractList();
                })
                .catch(err => {
                  this.displayError(err);
                });
              })
              .catch(err => {
                this.displayError(err);
              });
            })
            .catch(err => {
              this.displayError(err);
            });
          })
          .catch(err => {
            this.displayError(err);
          });
        })
        .catch(err => {
          this.displayError(err);
        });
      })
      .catch(err => {
        this.displayError(err);
      });
    })
    .catch(err => {
      this.displayError(err);
    });
  }

  /*
   * handler for offer commands
   */
  handleOfferCommand(offer, command, arg1, arg2) {
    switch(command) {
      case 'decline':
        this.state.lc.send('LitRPC.DeclineOffer', {
          'OIdx' : offer.OIdx
        })
        .then(reply => {
          this.updateOfferList();
        })
        .catch(err => {
          this.displayError(err);
        });
        break;
      case 'accept':
        this.state.lc.send('LitRPC.AcceptOffer', {
          'OIdx' : offer.OIdx
        })
        .then(reply => {
          this.updateOfferList();
          // The peers need some time to exchange signatures. Refresh it again in a while,
          // together with the balances since part of our balance will be moved into the
          // contract
          setTimeout(this.updateContractList.bind(this), 6000);
          setTimeout(this.updateBalances.bind(this), 6000);
        })
        .catch(err => {
          this.displayError(err);
        });
        break;
      default:
        this.displayError("Unrecognized contract command " + command);
    }
  }

  /*
   * click handler for offering a contract
   */
  handleContractCommand(contract, command, arg1, arg2) {
    switch(command) {
      case 'settle':
        var buf = Buffer.from(arg2,'hex');
        var sig = [];
        for (var i = 0; i < buf.length; i++) sig[i] = buf[i];
        this.state.lc.send('LitRPC.SettleContract', {
          'CIdx' : contract.Idx,
          'OracleValue' : parseInt(arg1, 10),
          'OracleSig' : sig
        })
        .then(reply => {
          this.updateContractList();
          // The peers need some time to exchange signatures. Refresh it again in a while
          setTimeout(this.updateContractList.bind(this), 3000);
          setTimeout(this.updateContractList.bind(this), 6000);
          // Also update the balances in a while since whatever came out of the contract
          // is now in our balance again
          setTimeout(this.updateBalances.bind(this), 6000);
        })
        .catch(err => {
          this.displayError(err);
        });
        break;
      case 'offer':
        this.state.lc.send('LitRPC.OfferContract', {
          'CIdx' : contract.Idx,
          'PeerIdx' : parseInt(arg1, 10)
        })
        .then(reply => {
          this.updateContractList();
        })
        .catch(err => {
          this.displayError(err);
        });
        break;
      case 'decline':
        this.state.lc.send('LitRPC.DeclineContract', {
          'CIdx' : contract.Idx
        })
        .then(reply => {
          this.updateContractList();
        })
        .catch(err => {
          this.displayError(err);
        });
        break;
      case 'accept':
        this.state.lc.send('LitRPC.AcceptContract', {
          'CIdx' : contract.Idx
        })
        .then(reply => {
          this.updateContractList();
          // The peers need some time to exchange signatures. Refresh it again in a while,
          // together with the balances since part of our balance will be moved into the
          // contract
          setTimeout(this.updateContractList.bind(this), 4000);
          setTimeout(this.updateBalances.bind(this), 4000);
        })
        .catch(err => {
          this.displayError(err);
        });
        break;
      default:
        this.displayError("Unrecognized contract command " + command);
    }
  }

  render() {
    return (
      <div className="App">
        <CssBaseline />
        <ErrorDialog
          errorMessage={this.state.errorMessage}
          handleSubmit={this.handleErrorDialogSubmit.bind(this)}
          />
        <LitAppBar
          address={this.state.Adr}
          settings={{
            rpcAddress: this.state.rpcAddress,
            rpcPort: this.state.rpcPort,
            rpcRefresh: this.state.rpcRefresh,
            appBarColorPrimary: this.state.appBarColorPrimary,
            hideClosedChannels: this.state.hideClosedChannels,
          }}
          handleSettingsSubmit={this.handleSettingsSubmit.bind(this)}
          hexStringToByte={this.hexStringToByte.bind(this)}
        />
        <Balances
          balances={this.state.Balances}
          handleSendSubmit={this.handleSendSubmit.bind(this)}
          coinRates={this.state.CoinRates}
          newAddress={this.address.bind(this)}
        />
        <Channels
          channels={this.state.Channels}
          connections={this.state.Connections}
          hideClosedChannels={this.state.hideClosedChannels}
          handleChannelCommand={this.handleChannelCommand.bind(this)}
          handleChannelAddSubmit={this.handleChannelAddSubmit.bind(this)}
          handlePeerAddSubmit={this.handlePeerAddSubmit.bind(this)}
          handlePeerNicknameSubmit={this.handlePeerNicknameSubmit.bind(this)}
        />
        <Contracts
          contracts={this.state.Contracts}
          offers={this.state.Offers}
          assets={this.state.Assets}
          connections={this.state.Connections}
          fetchAssetValue={this.fetchAssetValue.bind(this)}
          handleContractCommand={this.handleContractCommand.bind(this)}
          handleOfferCommand={this.handleOfferCommand.bind(this)}
          handleCreateContract={this.handleCreateContract.bind(this)}
        />
      </div>
    );
  }

  /*
   * Handler for settings Dialog
   */
  handleSettingsSubmit(settings) {
    this.resetLitConnection(settings.rpcAddress, settings.rpcPort, settings.rpcRefresh);
    this.setState( {
      appBarColorPrimary: settings.appBarColorPrimary,
      hideClosedChannels: settings.hideClosedChannels,
    })
  }

  /*
   * Resets all the host connections including refreshing.
   */
  resetLitConnection(address, port, refresh) {

    let lc = this.state.lc;
    let rpcRefreshReference = this.state.rpcRefreshReference;

    lc = new LitAfClient(address, port);

    if (this.state.rpcRefreshReference === -1) {
      if (refresh) {
        rpcRefreshReference = setInterval(this.updateLit.bind(this), 2000);
      }
    } else {
      if (!refresh) {
        clearInterval(this.state.rpcRefreshReference);
        rpcRefreshReference = -1;
      }
    }

    this.setState( {
      rpcAddress: address,
      rpcPort: port,
      rpcRefresh: refresh,
      lc: lc,
      rpcRefreshReference: rpcRefreshReference,
    }, this.update.bind(this));
  }

  componentDidMount() {
    this.resetLitConnection(this.state.rpcAddress, this.state.rpcPort, this.state.rpcRefresh);
  }
}


export default App;
