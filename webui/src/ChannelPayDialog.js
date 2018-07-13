/**
 * Created by joe on 4/21/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from "material-ui/styles/index";
import Button from 'material-ui/Button';
import Input, { InputLabel } from 'material-ui/Input';
import Dialog, {
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from 'material-ui/Dialog';
import { FormControl } from 'material-ui/Form';
import {coinInfo} from './CoinTypes.js'
import PopUpDialog from './PopUpDialog.js'

const styles = theme => ({
  content: {
    margin: theme.spacing.unit,
    minWidth: 350,
  },
  formControl: {
    marginTop: theme.spacing.unit,
  },
  amount: {
    maxWidth: 200,
  },
  data: {
    minWidth: 300,
  },
});

class ChannelPayDialog extends PopUpDialog {

  constructor(props) {
    super(props);
    this.state = Object.assign(this.state,
      {
        amount: 0,
        data: "",
      });
  }

  resetState() {
    this.setState({
      amount: 0,
      data: "",
    });
    super.resetState();
  }

  handleSubmit () {
    this.props.handlePaySubmit(Math.round(parseFloat(this.state.amount) * coinInfo[this.props.coinType].factor), this.state.data);
    super.handleSubmit();
  };

  render() {
    const {classes} = this.props;

    return (
      <div>
        <Button onClick={this.handleClickOpen.bind(this)}>Pay</Button>
        <Dialog
          open={this.state.open}
          onClose={this.handleClose.bind(this)}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">Pay to Channel</DialogTitle>
          <DialogContent className={classes.content}>
            <FormControl className={classes.formControl}>
              <InputLabel htmlFor="amount">Amount in {coinInfo[this.props.coinType].denomination}</InputLabel>
              <Input
                autoFocus
                id="amount"
                label="Amount"
                type="text"
                onChange={this.handleChange('amount').bind(this)}
                className={classes.amount}
              />
            </FormControl>
            <p/>
            <FormControl className={classes.formControl}>
            <InputLabel htmlFor="data">Channel Data (optional)</InputLabel>
            <Input
              id="data"
              label="Data"
              type="text"
              fullWidth
              onChange={this.handleChange('data').bind(this)}
              className={classes.data}
            />
            </FormControl>
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleClose.bind(this)} color="primary">
              Cancel
            </Button>
            <Button onClick={this.handleSubmit.bind(this)} color="primary">
              Pay
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}

ChannelPayDialog.propTypes = {
  handlePaySubmit: PropTypes.func.isRequired,
  coinType: PropTypes.number.isRequired,
};

export default withStyles(styles)(ChannelPayDialog);
