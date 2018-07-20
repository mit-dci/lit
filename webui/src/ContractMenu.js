/**
 * Created by joe on 4/29/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import IconButton from 'material-ui/IconButton';
import MenuIcon from 'material-ui-icons/Menu';
import Menu, { MenuItem } from 'material-ui/Menu';
import ConfirmDialog from './ConfirmDialog';

class ContractMenu extends React.Component {
  state = {
    anchorEl: null,
    confirmCommand: null,
    confirmOpen: false,
    confirmMessage: '',
  };

  handleClick = event => {
    this.setState({ anchorEl: event.currentTarget });
  };

  handleClose = () => {
    this.setState({ anchorEl: null });
  };

  handleMenu = (command) => {
    this.handleClose();

    switch (command) {
      case 'close':
        this.setState({
          confirmMessage: 'Confirming you wish to close this channel!',
        });
        break;
      case 'break':
        this.setState({
          confirmMessage: 'Confirming you wish to BREAK this channel. Note your funds will be tied up for some time!',
        });
        break;
      default:
    }

    this.setState({
      confirmOpen: true,
      confirmCommand: command,
    });
  }

  handleConfirm = (confirmed) => {
    if (confirmed) {
      this.props.handleChannelMenu(this.state.confirmCommand);
    }
    this.setState({ confirmOpen: false });
  }


  render() {
    const { anchorEl } = this.state;

    return (
      <div>
        <IconButton
          disabled={this.props.disabled}
          color="inherit"
          aria-label="Menu"
          aria-owns={anchorEl ? 'simple-menu' : null}
          aria-haspopup="true"
          onClick={this.handleClick}
          >
          <MenuIcon />
        </IconButton>
        <Menu
          id="simple-menu"
          anchorEl={anchorEl}
          open={Boolean(anchorEl)}
          onClose={this.handleClose}
        >
          <MenuItem onClick={event => {this.handleMenu('close')}}>Close</MenuItem>
          <MenuItem onClick={event => {this.handleMenu('break')}}>Break</MenuItem>
        </Menu>
        <ConfirmDialog
          open={this.state.confirmOpen}
          confirmTitle="Are you sure?"
          confirmMessage={this.state.confirmMessage}
          handleConfirm={this.handleConfirm.bind(this)}
          />
      </div>
    );
  }
}

ContractMenu.propTypes = {
  disabled: PropTypes.bool,
  handleChannelMenu: PropTypes.func.isRequired,
};

export default ContractMenu;