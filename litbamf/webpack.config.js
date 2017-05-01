var fs = require('fs-extra');
var ExtractTextPlugin = require('extract-text-webpack-plugin');
var UglifyJsPlugin = require('webpack-uglify-js-plugin');
var debug = process.env.NODE_ENV === 'debug';
var publicDir = process.env.HOME + '/.lit/litbamf'

module.exports = {
  entry: {
    main:  './client/js/main.js',
  },
  output: {
    path: debug ? publicDir
    : './public/',
    filename: 'js/[name].js' //Template based on keys in entry above
  },
  module: {
    loaders: [
      {
        test:    /\.js$/,
        exclude: /(node_modules)/,
        loader:  'babel',
        query:   {
          presets: ['es2015', 'react'],
        },
      },
      {
        test:    /\.json$/,
        loader:  'json',
      },
      {
        test: /\.scss$/,
        loader: ExtractTextPlugin.extract(
          'style', // The backup style loader
          'css?sourceMap!sass?sourceMap'
        )
      },
    ],
  },
  sassLoader: {
    includePaths: [ './client/sass' ]
  },
  plugins: debug ? [
    new ExtractTextPlugin('style/[name].css')
  ] : [
    new ExtractTextPlugin('style/[name].css'),
    new UglifyJsPlugin({
      cacheFolder: './public/cached_uglify/',
      minimize: true,
      compress: false
    })
  ]
};

fs.copy('./public', publicDir, function (err) {
  if(err)return console.error(err);
  return null;
});
