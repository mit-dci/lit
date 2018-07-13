/**
 * Created by joe on 4/16/18.
 */

/*
 * Utility functions
 */

// TODO - still needs some thinking!

const coinInfo = {
  0: {
    name: 'Bitcoin',
    denomination: 'mBTC',
    exchangeSymbol: 'BTC',
    factor: 100000,
    decimals: 2,
  },
  1: {
    name: 'Bitcoin Testnet3',
    denomination: 'mBTC-t',
    exchangeSymbol: 'BTC',
    factor: 100000,
    decimals: 2,
  },
  28: {
    name: 'Vertcoin',
    denomination: 'VTC',
    exchangeSymbol: 'VTC',
    factor: 100000000,
    decimals: 2,
  },
  257: {
    name: 'Bitcoin Regtest',
    denomination: 'mBTC-r',
    exchangeSymbol: 'BTC',
    factor: 100000,
    decimals: 2,
  },
};

const coinDenominations = (() => {
  let result = {};
  for (let i in coinInfo) {
    result[coinInfo[i].denomination] = i;
  }
  return result;
})();

const coinTypes = Object.keys(coinInfo).map(key => {
  return parseInt(key, 10);
});

function formatCoin(amount, coinType) {
  let info = coinInfo[coinType];

  if (info === null || info === undefined) {
    return Number(amount).toLocaleString() + " Type " + coinType;
  }

  return Number(amount / info.factor).toLocaleString(undefined, {
    minimumFractionDigits: info.decimals,
    maximumFractionDigits: info.decimals
  }) + " " + info.denomination;
}

function formatUSD(coinAmount, coinType, exchangeRates) {
  let info = coinInfo[coinType];

  return Number(coinAmount / info.factor * exchangeRates[coinType]).toLocaleString(undefined, {
    style: "currency",
    currency: "USD"
  });
}

export {coinInfo, coinDenominations, coinTypes, formatCoin, formatUSD};

