const path = require('path');
const autoprefixer = require('autoprefixer');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const ExtractTextPlugin = require('extract-text-webpack-plugin');
const webpack = require('webpack');
const flexBugsFixes = require('postcss-flexbugs-fixes');
const CleanWebpackPlugin = require('clean-webpack-plugin');
const CopyPlugin = require('copy-webpack-plugin');

const RESOURCES_PATH = path.resolve(__dirname);
const ENTRY_REACT = path.resolve(RESOURCES_PATH, 'src/index.js');
const ENTRY_INSTALL = path.resolve(RESOURCES_PATH, 'src/install/index.js');
const HTML_PATH = path.resolve(RESOURCES_PATH, 'public/index.html');
const HTML_INSTALL_PATH = path.resolve(RESOURCES_PATH, 'public/install.html');
const FAVICON_PATH = path.resolve(RESOURCES_PATH, 'public/favicon.ico');

const PUBLIC_PATH = path.resolve(__dirname, '../build/static');

const config = {
    target: 'web',
    context: RESOURCES_PATH,
    entry: {
        main: ENTRY_REACT,
        install: ENTRY_INSTALL,
    },
    output: {
        path: PUBLIC_PATH,
        filename: '[name].[chunkhash].js',
    },
    resolve: {
        modules: ['node_modules'],
    },
    module: {
        rules: [
            {
                test: /\.css$/,
                use: ExtractTextPlugin.extract({
                    fallback: 'style-loader',
                    use: [{
                        loader: 'css-loader',
                        options: {
                            importLoaders: 1,
                        },
                    },
                    {
                        loader: 'postcss-loader',
                        options: {
                            ident: 'postcss',
                            plugins: () => [
                                flexBugsFixes,
                                autoprefixer({
                                    browsers: [
                                        '>1%',
                                        'last 4 versions',
                                        'Firefox ESR',
                                        'not ie < 9',
                                    ],
                                    flexbox: 'no-2009',
                                }),
                            ],
                        },
                    },
                    ],
                }),
            },
            {
                test: /\.js$/,
                exclude: /node_modules/,
                use: {
                    loader: 'babel-loader',
                    options: {
                        cacheDirectory: true,
                        presets: [
                            ['env', {
                                modules: false,
                            }],
                            'react',
                            'stage-2',
                        ],
                        plugins: ['transform-runtime', 'transform-object-rest-spread'],
                    },
                },
            },
            {
                exclude: [/\.js$/, /\.html$/, /\.json$/, /\.css$/],
                use: {
                    loader: 'url-loader',
                    options: {
                        fallback: 'file-loader',
                        name: 'media/[name].[hash:8].[ext]',
                        limit: 10 * 1024,
                    },
                },
            },
        ],
    },
    plugins: [
        new webpack.DefinePlugin({
            'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
        }),
        new CleanWebpackPlugin(['*.*'], {
            root: PUBLIC_PATH,
            verbose: false,
            dry: false,
        }),
        new HtmlWebpackPlugin({
            inject: true,
            cache: false,
            chunks: ['main'],
            template: HTML_PATH,
        }),
        new HtmlWebpackPlugin({
            inject: true,
            cache: false,
            chunks: ['install'],
            filename: 'install.html',
            template: HTML_INSTALL_PATH,
        }),
        new ExtractTextPlugin({
            filename: '[name].[contenthash].css',
        }),
        new CopyPlugin([
            { from: FAVICON_PATH, to: PUBLIC_PATH },
        ]),
    ],
};

module.exports = config;
