/**
 * Created by joe on 4/29/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Button from 'material-ui/Button';
import Dialog, {
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from 'material-ui/Dialog';


const styles = theme => ({
  content: {
    minWidth: 320,
  }
});

class ConfirmDialog extends React.Component {

  render() {
    const {classes} = this.props;
    return (
      <div>
        <Dialog
          open={this.props.open}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">
            {this.props.confirmTitle}
          </DialogTitle>
          <DialogContent className={classes.content}>
            <DialogContentText>
              {this.props.confirmMessage}
            </DialogContentText>
          </DialogContent>
          <DialogActions>
            <Button onClick={event => {this.props.handleConfirm(false)}} color="primary">
              Cancel
            </Button>
            <Button onClick={event => {this.props.handleConfirm(true)}} color="primary">
              Confirm
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}

ConfirmDialog.propTypes = {
  open: PropTypes.bool,
  confirmTitle: PropTypes.string.isRequired,
  confirmMessage: PropTypes.string.isRequired,
  handleConfirm: PropTypes.func.isRequired,
};

export default withStyles(styles)(ConfirmDialog);