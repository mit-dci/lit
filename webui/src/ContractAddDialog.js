/**
 * Created by joe on 4/21/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Button from 'material-ui/Button';
import IconButton from 'material-ui/IconButton';

import Input, { InputLabel, InputAdornment } from 'material-ui/Input';
import Dialog, {
  DialogActions,
  DialogContent,
  DialogTitle,
} from 'material-ui/Dialog';
import Menu, {MenuItem} from 'material-ui/Menu';
import Select from 'material-ui/Select';
import AddIcon from '@material-ui/icons/Add';
import Typography from 'material-ui/Typography';
import { FormControl } from 'material-ui/Form';
import Grid from 'material-ui/Grid';
import DateFnsUtils from 'material-ui-pickers/utils/date-fns-utils';
import MuiPickersUtilsProvider from 'material-ui-pickers/utils/MuiPickersUtilsProvider';
import { DateTimePicker } from 'material-ui-pickers';
import RefreshIcon from '@material-ui/icons/Refresh';
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
    minWidth: 400,
  },
  amountBox: {
    display: 'flex',
  },
});

class ContractAddDialog extends React.Component {
  state = {
    menuAnchorEl: null,
    open: false,
    imSelling: false,
    asset: 0,
    priceType: 0,
    settleDate: new Date().setTime(Math.round(Math.floor(new Date().getTime()/1000)/300)*300*1000),
    settleUnix: 0,
    amount: 0,
    price: 0,
    satPrice: 0,
    peerIdx: 0
  };

  handleClickOpen = (event) => {
    this.handleDateChange(new Date());
    this.setState({menuAnchorEl : event.target});
  };

  handleClose = () => {
    this.setState({open: false});
  };

  handleSubmit = () => {
    this.props.handleCreateContract(this.state.imSelling, this.state.asset, parseFloat(this.state.amount), this.state.satPrice, this.state.settleUnix, this.state.peerIdx)
    this.setState({open: false});
  };

  handleChange = name => event => {
    this.setState({
      [name]: event.target.value,
    });
  };

  handleAssetSelect(event, index) {
    this.setState({
      asset: index,
    });
  }

  handlePopoverClose = () => {
    this.setState({
      menuAnchorEl: null,
    });
  };

  handleMenuItemClick = (event, index) => {
    this.setState({
      menuAnchorEl: null,
      open: true
    });
  }

  handlePriceChange = event => {
    var enteredPrice = parseFloat(event.target.value);

    var satPrice = Math.round(enteredPrice * 100000000)
    if(this.state.priceType !== 1) {
      satPrice = Math.round(100000000 / enteredPrice);
    }

    this.setState({
      price: enteredPrice,
      satPrice : satPrice
    });
  }

  handlePriceTypeChange = event => {
    var enteredPriceType = event.target.value;

    var price = this.state.satPrice / 100000000;
    if(enteredPriceType !== 1) {
      price = Math.round(100000000 / this.state.satPrice * 100) / 100;
    }

    this.setState({
      priceType: enteredPriceType,
      price : price
    });
  }

  handlePeerChange = event => {
    this.setState({peerIdx : event.target.value});

  }


  handleDateChange = (date) => {
    var epoch = Math.floor(date.getTime()/1000);
    epoch = Math.round(epoch/300) * 300; // round to 5 minute interval
    date.setTime(epoch*1000);
    this.setState({ settleUnix : epoch, settleDate: date });
  }

  refreshAsset = () => {

    var asset = this.props.assets[this.state.asset];
    if(asset != null) {
      var promise = this.props.fetchAssetValue(asset);
      if(promise !== null) {
        promise.then(val => {
          var price = val / 100000000
          if(this.state.priceType !== 1) {
            price = Math.round(100000000 / val * 100) / 100;
          }

          this.setState({price : price, satPrice: val});
        });
      }
    }
  }

  render() {
    const {classes} = this.props;
    const {menuAnchorEl, peerIdx, settleDate, asset, imSelling, amount, price, priceType} = this.state;

    return (
      <div>
        <div className={classes.buttonBox}>
          <Button variant="fab" onClick={this.handleClickOpen}>
            <AddIcon />
          </Button>
          <Typography variant="caption" className={classes.caption}>
            Contract
          </Typography>
        </div>
        <Menu
          id="contractType-menu"
          anchorEl={menuAnchorEl}
          open={Boolean(menuAnchorEl)}
          onClose={this.handleClose}
        >
        <DialogTitle>
          Contract type:
          </DialogTitle>
            <MenuItem selected={true} onClick={event => this.handleMenuItemClick(event)}>
                Futures contract
              </MenuItem>
          </Menu>
     
        <Dialog
          open={this.state.open}
          onClose={this.handleClose}
          aria-labelledby="form-dialog-title"
        >
          <DialogTitle id="form-dialog-title">New Futures Contract</DialogTitle>
          <DialogContent className={classes.content}>
            <Grid container spacing={8}>
              <Grid item xs={3}>
                <FormControl id="buyorsellfc" className={classes.formControl}>
                  <InputLabel htmlFor="buyorsell">I am:</InputLabel>
                  <Select 
                  inputProps={{
                    name: 'buyorsell',
                    id: 'buyorsell',
                  }} 
                  fullWidth
                  value={imSelling} 
                  onChange={this.handleChange('imSelling')}>
                    <MenuItem value={false}>Buying</MenuItem>
                    <MenuItem value={true}>Selling</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={6}>
                <FormControl id="amountfc" className={classes.formControl}>
                  <InputLabel htmlFor="amount">Amount:</InputLabel>
                  <Input
                    name="amount"
                    id="amount"
               fullWidth
                  inputProps={{style:{textAlign:'right'}}}
                  type="text"
                  value={amount} 
                  onChange={this.handleChange('amount')} 
                  />
                </FormControl>
              </Grid>
              <Grid item xs={3}>
                <FormControl id="assetfc" className={classes.formControl}>
                  <InputLabel htmlFor="asset">Asset:</InputLabel>
                  <Select 
                    inputProps={{
                      name: 'asset',
                      id: 'asset',
                    }}
                    value={asset}
                    onChange={this.handleChange('asset')}
                    fullWidth>
                    {this.props.assets.map((option, index) => (
                      <MenuItem value={index}>{option.name}</MenuItem>
                    ))}
                    
                  </Select>
                </FormControl>
              </Grid>
            </Grid>
          </DialogContent>
          <DialogContent className={classes.content}>
            <Grid container spacing={8}>
              <Grid item xs={12}>
                <MuiPickersUtilsProvider utils={DateFnsUtils}>
                <DateTimePicker
                    value={settleDate}
                    disablePast
                    fullWidth
                    onChange={this.handleDateChange}
                    label="On:"
                  />
                </MuiPickersUtilsProvider>
              </Grid>
              </Grid>
          </DialogContent>
          <DialogContent>
            <Grid container spacing={8}>
            <Grid item xs={7}>
              <FormControl id="pricefc" className={classes.formControl}>
                  <InputLabel htmlFor="price">Priced at:</InputLabel>
                  <Input id="price"
                     inputProps={{style:{textAlign:'right'}}}
                     
                  type="text"
                  value={price} 
                  onChange={this.handlePriceChange} 
                  endAdornment={<InputAdornment position='end'>
                  <IconButton className={classes.button} aria-label="Fetch current price from oracle" onClick={this.refreshAsset}>
                    <RefreshIcon />
                  </IconButton></InputAdornment>}
                  />
                  
                </FormControl>
              </Grid>
              <Grid item xs={5}>
              <FormControl id="priceTypefc" className={classes.formControl}>
                <InputLabel>&nbsp;</InputLabel>
                  <Select 
                    inputProps={{
                      name: 'priceType',
                      id: 'priceType',
                    }} 
                    value={priceType} 
                    onChange={this.handlePriceTypeChange}>
                    <MenuItem value={0}>{this.props.assets[asset] ? this.props.assets[asset].name : ""} per BTC</MenuItem>
                    <MenuItem value={1}>BTC per {this.props.assets[asset] ? this.props.assets[asset].name : ""}</MenuItem>
                  </Select>
                  </FormControl>
              </Grid>
            </Grid>
          </DialogContent>
          <DialogContent>
            <Grid container spacing={8}>
              <Grid item xs={12}>
              
                <FormControl id="peerfc" className={classes.formControl}>
                  <InputLabel>Peer</InputLabel>
                    <Select 
                      inputProps={{
                        name: 'peer',
                        id: 'peer',
                      }} 
                      fullWidth
                      value={peerIdx} 
                      onChange={this.handlePeerChange}>
                      {this.props.connections.map((conn, index) => (
                        <MenuItem value={conn.PeerNumber}>{
                          conn.Nickname === '' ? 'Peer ' + conn.PeerNumber.toString() : conn.Nickname
                        }</MenuItem>
                      ))}
                    </Select>
                </FormControl>
              </Grid>
            </Grid>
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleClose} color="primary">
              Cancel
            </Button>
            <Button onClick={this.handleSubmit} color="primary">
              Save
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}


ContractAddDialog.propTypes = {
  handleCreateContract: PropTypes.func.isRequired,
};


export default withStyles(styles)(ContractAddDialog);
