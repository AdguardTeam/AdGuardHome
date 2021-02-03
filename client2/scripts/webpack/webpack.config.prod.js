const path = require('path');
const { merge } = require('webpack-merge');
const baseConfig = require('./webpack.config.base');
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const OptimizeCSSAssetsPlugin = require('optimize-css-assets-webpack-plugin');
const TerserJSPlugin = require('terser-webpack-plugin');
const Webpack = require('webpack');
const CopyPlugin = require('copy-webpack-plugin');

module.exports = merge(baseConfig, {
    mode: 'production',
    devtool: 'source-map',
    output: {
        path: path.resolve(__dirname, '../../../build2/static'),
        filename: '[name].bundle.[hash:5].js',
        publicPath: '/'
    },
    optimization: {
        minimizer: [new TerserJSPlugin({terserOptions: {
            output: {
              comments: false,
            },
          },
          extractComments: false,
        }), new OptimizeCSSAssetsPlugin({})],
        splitChunks: {
            cacheGroups: {
                styles: {
                    name: 'styles',
                    test: /\.css$/,
                    chunks: 'all',
                    enforce: true,
                },
            },
        },
    },
    module: {
        rules: [
            {
                test: (resource) => {
                    return (
                        resource.indexOf('.pcss')+1
                        || resource.indexOf('.css')+1
                        || resource.indexOf('.less')+1
                    ) && !(resource.indexOf('.module.')+1);
                },
                use: [{
                    loader: MiniCssExtractPlugin.loader,
                }, 'css-loader', 'postcss-loader', {
                    loader: 'less-loader',
                    options: {
                        javascriptEnabled: true,
                    },
                }],
                exclude: /node_modules/,
            },
            {
                test: /\.module\.p?css$/,
                use: [
                        {
                            loader: MiniCssExtractPlugin.loader,
                        },
                        {
                            loader: 'css-loader',
                            options: {
                                modules: true,
                                sourceMap: true,
                                importLoaders: 1,
                            },
                        },
                        'postcss-loader',
                ],
                exclude: /node_modules/,
            }
        ]
    },
    plugins: [
        new Webpack.DefinePlugin({
            DEV: false,
        }),
        new MiniCssExtractPlugin({
            filename: '[name].[hash:5].css',
        }),
    ]
});
