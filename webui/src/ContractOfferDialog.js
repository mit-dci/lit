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

export default class ContractOfferDialog extends React.Component {
  state = {
    open: false,
    peeridx: 0
  };

  handleClickOpen = () => {
    this.setState({ open: true });
  };

  handleClose = () => {
    this.setState({ open: false });
  };

  handleSubmit = () => {
    this.props.handleOfferSubmit(this.state.peeridx);
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
        <Button onClick={this.handleClickOpen}>Offer</Button>
        <Dialog
          open={this.state.open}
          onClose={this.handleClose}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">Offer contract</DialogTitle>
          <DialogContent>
            <DialogContentText>
              Peer index:
            </DialogContentText>
            <Input
              autoFocus
              id="peeridx"
              label="Peer index"
              type="text"
              fullWidth
              onChange={this.handleChange('peeridx')}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleClose} color="primary">
              Cancel
            </Button>
            <Button onClick={this.handleSubmit} color="primary">
              Offer
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}

ContractOfferDialog.propTypes = {
  handleOfferSubmit: PropTypes.func.isRequired,
};
