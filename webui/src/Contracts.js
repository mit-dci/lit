/**
 * Created by gert-jaap on 5/1/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Grid from 'material-ui/Grid';
import Zoom from 'material-ui/transitions/Zoom';

import ContractCard from './ContractCard.js'
import ContractAddDialog from './ContractAddDialog.js'
import OfferCard from './OfferCard.js'

const styles = theme => ({
  root: {
    marginTop: 8,
  },
  peerGroup: {
    marginTop: 8,
    padding: 10,
    backgroundColor: 'lightBlue',
  },
  peerInfo: {
    display: 'flex',
    alignItems: 'center',
  },
  chip: {
    marginLeft: theme.spacing.unit
  },
  addButtonBox: {
    minWidth: 400,
    minHeight: 200,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  cardBox: {
    minWidth: 400,
    minHeight: 200,
  },
});


/*
 * All the Contracts
 */
function Contracts(props) {
  const {classes} = props;

  let offers = props.offers.map((offer, index) => {
    return (
      <Zoom in key={offer.OIdx}>
        <Grid item xs={3} className={classes.cardBox}>
          <OfferCard offer={offer} handleOfferCommand={props.handleOfferCommand}/>
        </Grid>
      </Zoom>
    );
  });

  let contracts = props.contracts.map((contract, index) => {
      return (
        <Zoom in key={contract.Idx}>
          <Grid item xs={3} className={classes.cardBox}>
            <ContractCard contract={contract} handleContractCommand={props.handleContractCommand}/>
          </Grid>
        </Zoom>
      );
  });

  return (
    <div className={classes.root}>
      <div className={classes.peerGroup}>
        <Grid container>
          {offers}
          {contracts}
          <Zoom in key="AddDialog">
            <Grid item xs={4} className={classes.addButtonBox}>
              <ContractAddDialog
                connections={props.connections}
                handleCreateContract={props.handleCreateContract}
                assets={props.assets}
                fetchAssetValue={props.fetchAssetValue}
                />
            </Grid>
          </Zoom>
        </Grid>
      </div>
    </div>
  );
}

Contracts.propTypes = {
  contracts: PropTypes.array.isRequired,
  connections: PropTypes.array.isRequired,
  handleCreateContract: PropTypes.func.isRequired,
  handleOfferCommand: PropTypes.func.isRequired,
  offers: PropTypes.array.isRequired,
  fetchAssetValue: PropTypes.func.isRequired,
  assets: PropTypes.array.isRequired
};

export default withStyles(styles)(Contracts);
