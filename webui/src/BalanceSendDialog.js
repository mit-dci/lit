/**
 * Created by joe on 4/21/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import Button from 'material-ui/Button';
import Input from 'material-ui/Input';
import Dialog, {
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from 'material-ui/Dialog';
import {coinInfo} from './CoinTypes.js'
import PopUpDialog from './PopUpDialog.js'
import {withStyles} from "material-ui/styles/index";

const styles = theme => ({
  content: {
    minWidth: 500,
  },
});


class BalanceSendDialog extends PopUpDialog {

  constructor(props) {
    super(props);
    this.state = Object.assign(this.state,
      {
        amount: 0,
        address: "",
      });
  }

  resetState() {
    this.setState({
      amount: 0,
      address: "",
    });
    super.resetState();
  }

  handleSubmit () {
    this.props.handleSendSubmit(this.state.address, Math.round(parseFloat(this.state.amount) * coinInfo[this.props.coinType].factor));
    super.handleSubmit();
  };

  render() {
    const {classes} = this.props;

    return (
      <div>
        <Button onClick={this.handleClickOpen.bind(this)}>Send</Button>
        <Dialog
          open={this.state.open}
          onClose={this.handleClose.bind(this)}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">Send to Address</DialogTitle>
          <DialogContent className={classes.content}>
            <DialogContentText>
              Enter the amount to send in {coinInfo[this.props.coinType].denomination}
            </DialogContentText>
            <Input
              autoFocus
              id="amount"
              label="Amount"
              type="text"
              onChange={this.handleChange('amount').bind(this)}
            />
            <p/>
            <DialogContentText>
              Enter the Address to send to
            </DialogContentText>
            <Input
              id="address"
              label="Address"
              type="text"
              fullWidth
              onChange={this.handleChange('address').bind(this)}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleClose.bind(this)} color="primary">
              Cancel
            </Button>
            <Button onClick={this.handleSubmit.bind(this)} color="primary">
              Send
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}

BalanceSendDialog.propTypes = {
  handleSendSubmit: PropTypes.func.isRequired,
  coinType: PropTypes.number.isRequired,
};

export default withStyles(styles)(BalanceSendDialog);
