import React from 'react';
import PropTypes from 'prop-types';
import { withStyles } from 'material-ui/styles';
import AppBar from 'material-ui/AppBar';
import Toolbar from 'material-ui/Toolbar';
import Typography from 'material-ui/Typography';
import Blockies from 'react-blockies';
import Avatar from 'material-ui/Avatar';
import SettingsDialog from './SettingsDialog.js';

const styles = theme => ({
  root: {
    flexGrow: 1,
  },
  avatar: {
    marginRight: theme.spacing.unit,
  },
  flex: {
    flex: 1,
  },
  menuButton: {
   // marginRight: theme.spacing.unit,
  },
});

function LitAppBar(props) {
  const { classes } = props;
  return (
    <div className={classes.root}>
      <AppBar
        position="static"
        color={props.settings.appBarColorPrimary ? "primary" : "secondary"}
      >
        <Toolbar>
          <Avatar className={classes.avatar}>
            <Blockies
              seed={props.address}
              size={10}
              scale={3}
              color="#FF5733"
              bgColor="#FFC300"
            />
          </Avatar>
          <Typography variant="title" color="inherit" className={classes.flex}>
            Lit Node {props.address}
          </Typography>
        <SettingsDialog
          settings={props.settings}
          handleSettingsSubmit={props.handleSettingsSubmit}
          />
        </Toolbar>
      </AppBar>
    </div>
  );
}

LitAppBar.propTypes = {
  classes: PropTypes.object.isRequired,
  address: PropTypes.string.isRequired,
  settings: PropTypes.object.isRequired,
  handleSettingsSubmit: PropTypes.func.isRequired,
};

export default withStyles(styles)(LitAppBar);
