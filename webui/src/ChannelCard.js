/**
 * Created by joe on 4/11/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Card, {CardActions, CardContent, CardHeader} from 'material-ui/Card';
import Button from 'material-ui/Button';
import Typography from 'material-ui/Typography';
import { LinearProgress } from 'material-ui/Progress';
import {formatCoin} from './CoinTypes.js'
import ChannelPayDialog from './ChannelPayDialog';
import ChannelMenu from './ChannelMenu.js'
import './ChannelCard.css' // highlight css style (@keyframes can't be done in MUI styles)

const styles = theme => ({
  progressRoot: {
    height: 10,
  },
  progressColorPrimary: {
    backgroundColor: theme.palette.secondary.light,
  },
  tool: {
    display: 'flex',
  },
  card: {
  },
  cardDisabled: {
    backgroundColor: 'lightGrey',
  },
  balance: {
    fontSize: 14,
  },
  actions: {
    display: 'flex',
    justifyContent: 'space-between',
  },
  pay: {
    marginLeft: 'auto',
  },
  divider: {}
});

/*
 * Component for displaying the actual channel balances. Text right now, but will be graphical at some point!
 */
const ChannelBalance = withStyles(styles)((props) => {
  const {classes} = props;
  return (
    <div className={(props.highlight ? "BalHighlight" : "")}>
      <LinearProgress
        variant="determinate"
        value={100 * props.myBalance / props.capacity}
        classes={{
          root: classes.progressRoot,
          colorPrimary: classes.progressColorPrimary,
        }}
        />
      <Typography variant="body2" className={classes.balance}>
        Your Balance: {formatCoin(props.myBalance, props.coinType)}
      </Typography>
      <Typography className={classes.balance} color="textSecondary">
        Their Balance: {formatCoin(props.capacity - props.myBalance, props.coinType)}
      </Typography>
      <Typography className={classes.balance} color="textSecondary">
        Capacity: {formatCoin(props.capacity, props.coinType)}
      </Typography>
    </div>
  );
});

ChannelBalance.propTypes = {
  myBalance: PropTypes.number.isRequired,
  capacity: PropTypes.number.isRequired,
};


/*
 * Main ChannelCard component
 */
class ChannelCard extends React.Component {

  state = {
    myBalance: 0,
    highlight: false,
  };

  // handler to pass down to ChannelPayDialog that invokes props.handleChannelCommand
  handlePaySubmit(amount, data) {
    // note that the func passed down through props needs the channel
    this.props.handleChannelCommand(this.props.channel, 'push', amount, data);
  }

  // handler to pass down to ChannelMenu that invokes props.handleChannelCommand
  handleChannelMenu(command) {
    // note that the func passed down through props needs the channel
    this.props.handleChannelCommand(this.props.channel, command);
  }

  // Notice when a new balance is coming in so we can trigger the highlight animation
  componentWillReceiveProps(nextProps) {
    if (this.state.myBalance === 0) { // don't highlight if it's the first real balance
      this.setState({myBalance: nextProps.channel.MyBalance});
    } else if (nextProps.channel.MyBalance !== this.state.myBalance) {
      this.setState({
        myBalance: nextProps.channel.MyBalance,
        highlight: true
      });
      setTimeout(() => {
        this.setState({highlight: false})
      }, 1000); // bit icky, but reset the highlight state
    }
  }

  render() {

    const {classes} = this.props;

    let menuButton, payButton, channelStatus; // React components

    // if the channel card is disabled, disable the controls and render it with the disabled style
    if (this.props.disabled) {
      menuButton = (
        <ChannelMenu
          disabled
          handleChannelMenu = {this.handleChannelMenu.bind(this)}
        />
      );
      payButton = (
        <Button disabled className={classes.pay}>Pay</Button>
      );
    } else {
      menuButton = (
        <ChannelMenu
          handleChannelMenu = {this.handleChannelMenu.bind(this)}
        />
      );
      payButton = (
        <div className={classes.pay}>
          <ChannelPayDialog
            handlePaySubmit={this.handlePaySubmit.bind(this)}
            coinType={this.props.channel.CoinType}
          />
        </div>
      );
    }

    // render the channel status -- TODO need to understand other statuses
    channelStatus = null;
    if (this.props.channel.Closed) {
      channelStatus = (
        <Typography>
          Closed
        </Typography>
      );
    } else if (this.props.channel.Height <= 0) {
      channelStatus = (
        <Typography>
          Pending
        </Typography>
      );
    }

    return (
      <div>
        <Card
          raised={true}
          className={ classes.card + (this.props.disabled ? " " + classes.cardDisabled : "") +
          (this.state.highlight ? " BackHighlight" : "") }
        >
          <CardHeader title={"Channel " + this.props.channel.CIdx}
                      action={menuButton}/>
          <CardContent>
            <ChannelBalance
              highlight={this.state.highlight}
              capacity={this.props.channel.Capacity}
              coinType={this.props.channel.CoinType}
              myBalance={this.props.channel.MyBalance}
            />
          </CardContent>
          <CardActions className={classes.action} disableActionSpacing>
            {channelStatus}
            {payButton}
          </CardActions>
        </Card>
      </div>
    );
  }
}

ChannelCard.propTypes = {
  classes: PropTypes.object.isRequired,
  channel: PropTypes.object.isRequired,
  handleChannelCommand: PropTypes.func,
};

export default withStyles(styles)(ChannelCard);
