/**
 * Created by joe on 4/21/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from "material-ui/styles/index";
import IconButton from 'material-ui/IconButton';
import { FormControl, FormControlLabel } from 'material-ui/Form';
import Button from 'material-ui/Button';
import Input, { InputLabel } from 'material-ui/Input';
import Checkbox from 'material-ui/Checkbox';
import Dialog, {
  DialogActions,
  DialogContent,
  DialogTitle,
} from 'material-ui/Dialog';
import SettingsIcon from 'material-ui-icons/Settings';
import PopUpDialog from './PopUpDialog.js'

const styles = theme => ({
  content: {
    display: 'flex',
    flexWrap: 'wrap',
  },
  formControl: {
    margin: theme.spacing.unit,
  },
});


class SettingsDialog extends PopUpDialog {

  constructor(props) {
    super(props);
    this.state = Object.assign(this.state,
      {
        settings: props.settings,
      });
  }

  handleSubmit() {
    this.props.handleSettingsSubmit(this.state.settings);
    super.handleSubmit();
  };

  // overrides PopUpDialog.handleChange
  handleChange(name) {
    return (event => {
      let settings = Object.assign({}, this.state.settings);
      settings[name] = event.target.value;
      this.setState({settings: settings});
    });
  }

  handleCheckboxChange(name) {
    return (event => {
      let settings = Object.assign({}, this.state.settings);
      settings[name] = event.target.checked;
      this.setState({settings: settings});
    });
  };

  render() {
    const {classes} = this.props;

    return (
      <div>
        <IconButton
          color="inherit"
          aria-label="Menu"
          onClick={this.handleClickOpen.bind(this)}
        >
          <SettingsIcon/>
        </IconButton>
        <Dialog
          open={this.state.open}
          onClose={this.handleClose.bind(this)}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">Settings</DialogTitle>
          <DialogContent className={classes.content}>
              <FormControl className={classes.formControl}>
                <InputLabel htmlFor="rpcAddress">RPC Address</InputLabel>
                <Input
                  id="rpcAddress"
                  value={this.state.settings.rpcAddress}
                  onChange={this.handleChange('rpcAddress').bind(this)} />
              </FormControl>
            <FormControl className={classes.formControl}>
              <InputLabel htmlFor="rpcPort">RPC Port</InputLabel>
              <Input
                id="rpcPort"
                value={this.state.settings.rpcPort}
                onChange={this.handleChange('rpcPort').bind(this)} />
            </FormControl>
            <FormControl className={classes.formControl}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={this.state.settings.rpcRefresh}
                    onChange={this.handleCheckboxChange('rpcRefresh')}
                    value="rpcRefresh"
                  />
                }
                label="Automatically Refresh"
              />
            </FormControl>
            <FormControl className={classes.formControl}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={this.state.settings.appBarColorPrimary}
                    onChange={this.handleCheckboxChange('appBarColorPrimary')}
                    value="appBarColorPrimary"
                  />
                }
                label="Primary App Bar Color"
              />
            </FormControl>
            <FormControl className={classes.formControl}>
            <FormControlLabel
              control={
                <Checkbox
                  checked={this.state.settings.hideClosedChannels}
                  onChange={this.handleCheckboxChange('hideClosedChannels')}
                  value="hideClosedChannels"
                />
              }
              label="Hide Closed Channels"
            />
          </FormControl>
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleClose.bind(this)} color="secondary">
              Cancel
            </Button>
            <Button onClick={this.handleSubmit.bind(this)} color="primary">
              Apply
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}

SettingsDialog.propTypes = {
  handleSettingsSubmit: PropTypes.func.isRequired,
  settings: PropTypes.object.isRequired,
};

export default withStyles(styles)(SettingsDialog);
