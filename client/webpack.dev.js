const merge = require('webpack-merge');
const common = require('./webpack.common.js');

module.exports = merge(common, {
    devtool: 'inline-source-map',
    module: {
        rules: [{
            test: /\.js$/,
            exclude: /node_modules/,
            loader: 'eslint-loader',
            options: {
                emitWarning: true,
                configFile: 'dev.eslintrc',
            },
        }],
    },
});
