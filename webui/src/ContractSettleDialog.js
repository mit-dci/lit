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

export default class ContractSettleDialog extends React.Component {
  state = {
    open: false,
    oracleValue: 0,
    oracleSignature: ''
  };

  handleClickOpen = () => {
    this.setState({ open: true });
  };

  handleClose = () => {
    this.setState({ open: false });
  };

  handleSubmit = () => {
    this.props.handleSettle(this.state.oracleValue, this.state.oracleSignature);
    this.setState({ open: false });
  };

  handleChange = name => event => {
    this.setState({
      [name]: event.target.value,
    });
  };

  render() {
    return (
      <div>
        <Button onClick={this.handleClickOpen}>Settle</Button>
        <Dialog
          open={this.state.open}
          onClose={this.handleClose}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">Settle contract</DialogTitle>
          <DialogContent>
            <DialogContentText>
              Oracle Value:
            </DialogContentText>
            <Input
              autoFocus
              id="oracleValue"
              label="Oracle Value"
              type="text"
              fullWidth
              onChange={this.handleChange('oracleValue')}
            />
          </DialogContent>
          <DialogContent>
            <DialogContentText>
              Oracle Signature:
            </DialogContentText>
            <Input
              autoFocus
              id="oracleSignature"
              label="Oracle Signature"
              type="text"
              fullWidth
              onChange={this.handleChange('oracleSignature')}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleClose} color="primary">
              Cancel
            </Button>
            <Button onClick={this.handleSubmit} color="primary">
              Settle
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}

ContractSettleDialog.propTypes = {
  handleSettle: PropTypes.func.isRequired,
};
