/**
 * Created by joe on 4/21/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Button from 'material-ui/Button';
import Input from 'material-ui/Input';
import Dialog, {
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from 'material-ui/Dialog';
import AddIcon from '@material-ui/icons/Add';
import Typography from 'material-ui/Typography';
import PopUpDialog from './PopUpDialog.js'
import {coinInfo, coinTypes} from './CoinTypes.js'
import CoinMenu from './CoinMenu.js';

const styles = theme => ({
  buttonBox: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
  },
  caption: {
    margin: 8,
  },
  content: {
    minWidth: 400,
  },
  amountBox: {
    display: 'flex',
  },
});


class ChannelAddDialog extends PopUpDialog {

  constructor(props) {
    super(props);
    this.state = Object.assign(this.state,
      {
        amount: 0,
        coinselect: 1,
        data: "",
      });
  }

  resetState() {
    this.setState({
      amount: 0,
      coinselect: 1,
      data: "",
    });
    super.resetState();
  }

  handleSubmit () {
    let coinType = coinTypes[this.state.coinselect - 1];
    this.props.handleAddSubmit(this.props.peerIndex, coinType, Math.round(coinInfo[coinType].factor * this.state.amount));
    super.handleSubmit();
  }

  handleCoinSelect(index) {
    this.setState({
      coinselect: index,
    });
  }

  render() {
    const {classes} = this.props;
    return (
      <div>
        <div className={classes.buttonBox}>
          <Button variant="fab" color="secondary" onClick={this.handleClickOpen.bind(this)}>
            <AddIcon/>
          </Button>
          <Typography variant="caption" className={classes.caption}>
            Channel
          </Typography>
        </div>
        <Dialog
          open={this.state.open}
          onClose={this.handleClose.bind(this)}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">Add New Channel</DialogTitle>
          <DialogContent className={classes.content}>
            <DialogContentText>
              Enter amount to fund
            </DialogContentText>
            <div className={classes.amountBox}>
              <CoinMenu
                onSelect={this.handleCoinSelect.bind(this)}
                selected={this.state.coinselect}
              />
              <Input
                autoFocus
                id="amount"
                label="Amount"
                type="text"
                fullWidth
                onChange={this.handleChange('amount').bind(this)}
              />
            </div>
          </DialogContent>
          <DialogContent className={classes.content}>
            <DialogContentText>
              Enter channel data
            </DialogContentText>
            <Input
              autoFocus
              id="data"
              label="Data"
              type="text"
              fullWidth
              onChange={this.handleChange('data').bind(this)}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleClose.bind(this)} color="primary">
              Cancel
            </Button>
            <Button onClick={this.handleSubmit.bind(this)} color="primary">
              Fund
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}


ChannelAddDialog.propTypes = {
  handleAddSubmit: PropTypes.func.isRequired,
  peerIndex: PropTypes.string.isRequired,
};


export default withStyles(styles)(ChannelAddDialog);
