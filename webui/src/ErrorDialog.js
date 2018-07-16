/**
 * Error Dialog
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
});

class ErrorDialog extends React.Component {

  render () {
    const {classes} = this.props;

    return(
      <div>
        <Dialog
          open={this.props.errorMessage !== null}
          onClose={this.props.handleSubmit.bind(this)}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">Whoops! An Error Occurred</DialogTitle>
          <DialogContent className={classes.content}>
            <DialogContentText>
              {this.props.errorMessage}
            </DialogContentText>
          </DialogContent>
          <DialogActions>
            <Button onClick={this.props.handleSubmit.bind(this)} color="primary">
              OK
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}

ErrorDialog.propTypes = {
  classes: PropTypes.object.isRequired,
  errorMessage: PropTypes.string,
  handleSubmit: PropTypes.func.isRequired,
};

export default withStyles(styles)(ErrorDialog);
