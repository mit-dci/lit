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
    minWidth: 500,
  },
});


class PeerAddDialog extends PopUpDialog {

  constructor(props) {
    super(props);
    this.state = Object.assign(this.state,
      {
        address: "",
      });
  }

  resetState() {
    this.setState({
      address: "",
    });
    super.resetState();
  }

  handleSubmit () {
    this.props.handleAddSubmit(this.state.address);
    super.handleSubmit();
  };

  render() {
    const {classes} = this.props;
    return (
      <div>
        <div className={classes.buttonBox}>
          <Button variant="fab" color="primary" onClick={this.handleClickOpen.bind(this)}>
            <AddIcon />
          </Button>
          <Typography variant="caption" className={classes.caption}>
            Connection
          </Typography>
        </div>
        <Dialog
          open={this.state.open}
          onClose={this.handleClose.bind(this)}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">Connect to New Peer</DialogTitle>
          <DialogContent className={classes.content}>
            <DialogContentText>
              Enter peer address
            </DialogContentText>
            <Input
              autoFocus
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
              Connect
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}

PeerAddDialog.propTypes = {
  handleAddSubmit: PropTypes.func.isRequired,
};

export default withStyles(styles)(PeerAddDialog);

