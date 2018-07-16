/**
 * Created by joe on 4/21/18.
 * Refactored into separate file by gertjaap on 5/1/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import List, {ListItem, ListItemText} from 'material-ui/List';
import Menu, {MenuItem} from 'material-ui/Menu';
import {coinDenominations} from './CoinTypes.js'


const coinStyles = theme => ({
  root: {
    width: '100%',
    maxWidth: 360,
    backgroundColor: theme.palette.background.paper,
  },
});

const coinOptions = ['Coin Type'].concat(Object.keys(coinDenominations));

class CoinMenu extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      anchorEl: null,
      selectedIndex: props.selected,
    };
    this.button = undefined;
  }

  handleClickListItem = event => {
    this.setState({anchorEl: event.currentTarget});
  };

  handleMenuItemClick = (event, index) => {
    this.setState({selectedIndex: index, anchorEl: null});
    this.props.onSelect(index);
  };

  handleClose = () => {
    this.setState({anchorEl: null});
  };

  render() {
    const {classes} = this.props;
    const {anchorEl} = this.state;

    return (
      <div className={classes.root}>
        <List component="nav">
          <ListItem
            button
            aria-haspopup="true"
            aria-controls="lock-menu"
            aria-label="When device is locked"
            onClick={this.handleClickListItem}
          >
            <ListItemText
              primary={coinOptions[this.state.selectedIndex]}
            />
          </ListItem>
        </List>
        <Menu
          id="lock-menu"
          anchorEl={anchorEl}
          open={Boolean(anchorEl)}
          onClose={this.handleClose}
        >
          {coinOptions.map((option, index) => (
            <MenuItem
              key={option}
              disabled={index === 0}
              selected={index === this.state.selectedIndex}
              onClick={event => this.handleMenuItemClick(event, index)}
            >
              {option}
            </MenuItem>
          ))}
        </Menu>
      </div>
    );
  }
}

CoinMenu.propTypes = {
  selected: PropTypes.number.isRequired,
};


export default withStyles(coinStyles)(CoinMenu);