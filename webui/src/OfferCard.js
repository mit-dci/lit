/**
 * Created by joe on 4/11/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Card, {CardActions, CardContent, CardHeader} from 'material-ui/Card';
import Button from 'material-ui/Button';
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
class OfferCard extends React.Component {

  state = {
    myBalance: 0,
    highlight: false,
  };


  handleAccept() {
    this.props.handleOfferCommand(this.props.offer, 'accept');
  }

  handleDecline() {
    this.props.handleOfferCommand(this.props.offer, 'decline');
  }


  componentWillReceiveProps(nextProps) {

  }

  render() {

    const {classes} = this.props;

    let acceptButton, declineButton; // React components

    acceptButton = (
      <Button className={classes.settle} onClick={this.handleAccept.bind(this)}>Accept</Button>
    );
    declineButton = (
      <Button className={classes.settle} onClick={this.handleDecline.bind(this)}>Decline</Button>
    );

    console.log(this.props.contract);

    return (
      <div className={classes.cardBox}>
        <Card raised={true} className={classes.card}>
          <CardHeader title={"Offer " + this.props.offer.OIdx}
                      />
          <CardContent>
            Received from peer {this.props.offer.PeerIdx}
          </CardContent>
          <CardActions className={classes.action} disableActionSpacing>

            {acceptButton}
            {declineButton}
          </CardActions>
        </Card>
      </div>
    );
  }
}

OfferCard.propTypes = {
  classes: PropTypes.object.isRequired,
  offer: PropTypes.object.isRequired,
  handleOfferCommand: PropTypes.func,
};

export default withStyles(styles)(OfferCard);
