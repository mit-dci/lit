/**
 * Created by joe on 4/11/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Card, {CardActions, CardContent, CardHeader} from 'material-ui/Card';
import Button from 'material-ui/Button';
import ContractOfferDialog from './ContractOfferDialog.js';
import ContractSettleDialog from './ContractSettleDialog.js';
import ContractMenu from './ContractMenu.js'
import Chip from 'material-ui/Chip';
import './ContractCard.css' // highlight css style (@keyframes can't be done in MUI styles)

const styles = theme => ({
  tool: {
    display: 'flex',
  },
  card: {},
  cardDisabled: {
    minWidth: 240,
    backgroundColor: 'lightGrey',
  },
  balance: {
    fontSize: 14,
  },
  actions: {
    display: 'flex',
  },
  offer: {
    marginLeft: 'auto',
  },
  settle: {
    marginLeft: 'auto',
  },
  accept: {
    marginLeft: 'auto',
  },
  decline: {
    marginLeft: 'auto',
  },
  divider: {}
});


/*
 * Main ChannelCard component
 */
class ContractCard extends React.Component {

  state = {
    myBalance: 0,
    highlight: false,
  };

  // handler to offer the contract to a peer
  handleOfferSubmit(peerIdx) {
    this.props.handleContractCommand(this.props.contract, 'offer', peerIdx);
  }

  handleAccept() {
    this.props.handleContractCommand(this.props.contract, 'accept');
  }

  handleDecline() {
    this.props.handleContractCommand(this.props.contract, 'decline');
  }

  handleSettle(oracleValue, oracleSig) {
    this.props.handleContractCommand(this.props.contract, 'settle', oracleValue, oracleSig);
  }


  // handler to pass down to ChannelMenu that invokes props.handleChannelCommand
  handleChannelMenu(command) {
    // note that the func passed down through props needs the channel
    this.props.handleChannelCommand(this.props.channel, command);
  }

  componentWillReceiveProps(nextProps) {
    
  }

  render() {

    const {classes} = this.props;
   
    var contractStatus = ['Draft','Offered to Peer ','Offered by Peer ','Declined','Accepted','Acknowledged','Active','Settling','Closed']

    let menuButton, offerButton, acceptButton, declineButton, settleButton; // React components

    if (this.props.contract.Status === 0) { // Draft contracts can be offered.
      offerButton = (
        <div className={classes.offer}>
          <ContractOfferDialog
            handleOfferSubmit={this.handleOfferSubmit.bind(this)}
          />
        </div>
      )
    } 

    if (this.props.contract.Status === 2) { // I can accept or decline contracts offered to me
      acceptButton = (
        <Button className={classes.settle} onClick={this.handleAccept.bind(this)}>Accept</Button>
      );
      declineButton = (
        <Button className={classes.settle} onClick={this.handleDecline.bind(this)}>Decline</Button>
      );
    } else {
      
    }

    if (this.props.contract.Status === 6) { // Active contracts can be settled
      settleButton = (
        <div className={classes.offer}>
          <ContractSettleDialog
            handleSettle={this.handleSettle.bind(this)}
          />
        </div>
      );
    } 

    menuButton = (
      <ContractMenu
        handleChannelMenu = {this.handleChannelMenu.bind(this)}
      />
    );

    var statusLabel = contractStatus[this.props.contract.Status];
    if(this.props.contract.Status === 1 || this.props.contract.Status === 2)
    {
      statusLabel += this.props.contract.PeerIdx;
    }
      
    console.log(this.props.contract);

    return (
      <div className={classes.cardBox}>
        <Card raised={true} className={classes.card}>
          <CardHeader title={"Contract " + this.props.contract.Idx}
                      action={menuButton}/>
          <CardContent>
            Status: <Chip label={statusLabel} />
          </CardContent>
          <CardActions className={classes.action} disableActionSpacing>
          
            {offerButton}
            {settleButton}
            {acceptButton}
            {declineButton}
          </CardActions>
        </Card>
      </div>
    );
  }
}

ContractCard.propTypes = {
  classes: PropTypes.object.isRequired,
  contract: PropTypes.object.isRequired,
  handleOfferCommand: PropTypes.func,
};

export default withStyles(styles)(ContractCard);
