const StyleLintPlugin = require('stylelint-webpack-plugin');
const merge = require('webpack-merge');
const common = require('./webpack.common.js');

module.exports = merge(common, {
    stats: 'minimal',
    performance : {
        hints : false
    },
    module: {
        rules: [
            {
                test: /\.js$/,
                exclude: /node_modules/,
                loader: 'eslint-loader',
                options: {
                    failOnError: true,
                    configFile: 'prod.eslintrc',
                },
            },
        ],
    },
    plugins: [
        new StyleLintPlugin({
            files: '**/*.css',
        }),
    ],
});
