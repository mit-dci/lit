/**
 * Created by joe on 4/21/18.
 */
import React from 'react';
import PropTypes from 'prop-types';
import { withStyles } from 'material-ui/styles';
import Grid from 'material-ui/Grid';
import Typography from 'material-ui/Typography';
import BalanceCard from './BalanceCard';


const styles = theme => ({
  balancesGroup: {
    marginTop: 8,
    padding: 10,
  },
  cardBox: {
    minWidth: 450,
  },
});

function Balances(props) {
  const {classes} = props;
  let balances = props.balances.map((balance, index) => {
    return (
      <Grid item xs={4} key={index} className={classes.cardBox}>
        <BalanceCard
          balance={balance}
          coinRates={props.coinRates}
          handleSendSubmit={props.handleSendSubmit}
          newAddress={props.newAddress}
        />
      </Grid>
    );
  });
  return (
    <div className={classes.balancesGroup}>
      <Grid container>
        <Grid item xs={12}>
          <Typography variant="title">
            Balances
          </Typography>
        </Grid>
        {balances}
      </Grid>
    </div>
  );
}

Balances.propTypes = {
  classes: PropTypes.object.isRequired,
  balances: PropTypes.array.isRequired,
  handleSendSubmit: PropTypes.func.isRequired,
  coinRates: PropTypes.object.isRequired,
  newAddress: PropTypes.func.isRequired,
};



export default withStyles(styles)(Balances);
