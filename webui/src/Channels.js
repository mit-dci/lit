/**
 * Created by joe on 4/4/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Grid from 'material-ui/Grid';
import Typography from 'material-ui/Typography';
import Chip from 'material-ui/Chip';
import Zoom from 'material-ui/transitions/Zoom';
import Blockies from 'react-blockies';
import Avatar from 'material-ui/Avatar';
import ChannelCard from './ChannelCard.js'
import ChannelAddDialog from './ChannelAddDialog.js'
import PeerAddDialog from './PeerAddDialog.js'
import EditableField from './EditableField.js'

const channelGroupStyles = theme => ({
  cardBox: {
    minWidth: 300,
    minHeight: 200,
  },
  addButtonBox: {
    minWidth: 300,
    minHeight: 200,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  }
});

/*
 * A group of Channel Cards that share the same peer (helper component for Channels)
 */
const ChannelGroup = withStyles(channelGroupStyles)((props) => {

  const {classes} = props;

  let channels = [];
  let disabledChannels = [];

  // render enabled channels first (though the entire channel group may be disabled)
  Object.keys(props.channels).forEach(key => {
    let channel = props.channels[key];

    // don't show the channel if it's closed and we're hiding
    if (props.hideClosedChannels && channel.Closed) {
      return;
    }

    if (!props.disabled && !channel.Closed) { // normal open channel
      channels.push(
        <Zoom in key={channel.CIdx}>
          <Grid item xs={3} className={classes.cardBox}>
            <ChannelCard
              disabled={channel.Height <= 0 ? true : false}
              channel={channel}
              handleChannelCommand={props.handleChannelCommand}/>
          </Grid>
        </Zoom>
      );
    } else { // show a disabled channel
      disabledChannels.push(
        <Zoom in key={channel.CIdx}>
          <Grid item xs={3} className={classes.cardBox}>
            <ChannelCard disabled channel={channel}/>
          </Grid>
        </Zoom>
      )
    }
  });
  // add the + button for adding an additional channel to this Peer
  disabledChannels.push(
    <Zoom in key="AddDialog">
      <Grid item xs={3} className={classes.addButtonBox}>
        <ChannelAddDialog
          peerIndex={props.peerIndex}
          handleAddSubmit={props.handleChannelAddSubmit}
        />
      </Grid>
    </Zoom>
  );

  return (channels.concat(disabledChannels));
});


const styles = theme => ({
  root: {
    marginTop: 8,
  },
  peerGroup: {
    marginTop: 8,
    padding: 10,
    backgroundColor: 'lightBlue',
  },
  avatar: {
    margin: theme.spacing.unit,
  },
  peerInfo: {
    display: 'flex',
    alignItems: 'center',
  },
  chip: {
    marginLeft: theme.spacing.unit
  },
  addButtonBox: {
    minWidth: 300,
    minHeight: 200,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
});


/*
 * All the Channels, grouped by Peer
 */
function Channels(props) {
  const {classes} = props;

  let channelsByPeer = sortChannels(props.channels, props.connections);

  let peerChannels = Object.keys(channelsByPeer).map(key => {
      return (
        <div className={classes.peerGroup} key={key}>
          <Grid container>
            <Grid item xs={12}>
              <Grid container>
                <Grid item xs={12} className={classes.peerInfo}>
                  { /*
                    <Avatar className={classes.avatar}>
                      <Blockies
                        seed={key}
                        size={10}
                        scale={3}
                        color="#FF5733"
                        bgColor="#FFC300"
                      />
                    </Avatar>
                    */
                  }
                  <Typography variant="title">
                    <EditableField
                      string={channelsByPeer[key].nickname !== "" ?
                        channelsByPeer[key].nickname :
                        "Peer " + key}
                      handleSubmit={nickname => {
                        props.handlePeerNicknameSubmit(parseInt(key, 10), nickname);
                      }
                      }
                    />
                  </Typography>
                  {channelsByPeer[key].connected &&
                  <Chip label="Connected" className={classes.chip}/>}
                </Grid>
                <ChannelGroup
                  disabled={!channelsByPeer[key].connected}
                  channels={channelsByPeer[key].channels}
                  hideClosedChannels={props.hideClosedChannels}
                  handleChannelCommand={props.handleChannelCommand}
                  peerIndex={key}
                  handleChannelAddSubmit={props.handleChannelAddSubmit}
                />
              </Grid>
            </Grid>
          </Grid>
        </div>
      );
    }
  );

  return (
    <div className={classes.root}>
      {peerChannels}
      <div className={classes.peerGroup}>
        <Grid container>
          <Grid item xs={3}>
            <div className={classes.addButtonBox}>
              <PeerAddDialog
                handleAddSubmit={props.handlePeerAddSubmit}
              />
            </div>
          </Grid>
        </Grid>
      </div>
    </div>
  );
}

Channels.propTypes = {
  channels: PropTypes.array.isRequired,
  disabled: PropTypes.bool,
  hideClosedChannels: PropTypes.bool.isRequired,
  handleChannelCommand: PropTypes.func.isRequired,
  handleChannelAddSubmit: PropTypes.func.isRequired,
  handlePeerAddSubmit: PropTypes.func.isRequired,
  handlePeerNicknameSubmit: PropTypes.func.isRequired,
};

/*
 * Takes the channels and connections from returns an object in the following format:
 * {<Peer Index>: {connected: <true|false>, channels: {<Channel Index>: <Channel Info from Lit>}}...}
 */
function sortChannels(channels, connections) {

  /*
   result: an object keyed by PeerIdx where each element is a object as follows:
   channels: map of channel data keyed by ChannelIdx
   connected: true or false indicated whether peer is currently connected
   */
  let result = {};

  // iterate through connections adding an entry if it's connected and its Nickname
  connections.forEach(conn => {
    result[conn.PeerNumber] = {
      connected: true,
      nickname: conn.Nickname,
      channels: {},
    }
  });

  // now iterate through all the channels assigning them to the appropriate Peer
  channels.forEach(channel => {
    // find the existing entry for the PeerIdx or make a new one (not connected otherwise it would already exist)
    let entry = (channel.PeerIdx in result ? result[channel.PeerIdx] :
      {
        connected: false,
        nickname: "",
      });
    let item = ('channels' in entry ? entry.channels : {});
    item[channel.CIdx] = channel;
    entry.channels = item;
    result[channel.PeerIdx] = entry;
  });

  return result;
}

export default withStyles(styles)(Channels);
