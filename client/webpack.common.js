const path = require('path');
const autoprefixer = require('autoprefixer');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const flexBugsFixes = require('postcss-flexbugs-fixes');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');
const CopyPlugin = require('copy-webpack-plugin');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');

const RESOURCES_PATH = path.resolve(__dirname);
const ENTRY_REACT = path.resolve(RESOURCES_PATH, 'src/index.js');
const ENTRY_INSTALL = path.resolve(RESOURCES_PATH, 'src/install/index.js');
const ENTRY_LOGIN = path.resolve(RESOURCES_PATH, 'src/login/index.js');
const HTML_PATH = path.resolve(RESOURCES_PATH, 'public/index.html');
const HTML_INSTALL_PATH = path.resolve(RESOURCES_PATH, 'public/install.html');
const HTML_LOGIN_PATH = path.resolve(RESOURCES_PATH, 'public/login.html');
const ASSETS_PATH = path.resolve(RESOURCES_PATH, 'public/assets');

const PUBLIC_PATH = path.resolve(__dirname, '../build/static');
const PUBLIC_ASSETS_PATH = path.resolve(PUBLIC_PATH, 'assets');

const BUILD_ENVS = {
    dev: 'development',
    prod: 'production',
};

const BUILD_ENV = BUILD_ENVS[process.env.BUILD_ENV];

const config = {
    mode: BUILD_ENV,
    target: 'web',
    context: RESOURCES_PATH,
    entry: {
        main: ENTRY_REACT,
        install: ENTRY_INSTALL,
        login: ENTRY_LOGIN,
    },
    output: {
        path: PUBLIC_PATH,
        filename: '[name].[chunkhash].js',
    },
    resolve: {
        modules: ['node_modules'],
        alias: {
            MainRoot: path.resolve(__dirname, '../'),
            ClientRoot: path.resolve(__dirname, './src'),
        },
    },
    module: {
        rules: [
            {
                test: /\.css$/i,
                use: [
                    'style-loader',
                    MiniCssExtractPlugin.loader,
                    {
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
                                    flexbox: 'no-2009',
                                }),
                            ],
                        },
                    },
                ],
            },
            {
                test: /\.js$/,
                exclude: /node_modules/,
                use: {
                    loader: 'babel-loader',
                    options: {
                        cacheDirectory: true,
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
        new CleanWebpackPlugin({
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
        new HtmlWebpackPlugin({
            inject: true,
            cache: false,
            chunks: ['login'],
            filename: 'login.html',
            template: HTML_LOGIN_PATH,
        }),
        new MiniCssExtractPlugin({
            filename: '[name].[contenthash].css',
        }),
        new CopyPlugin({
            patterns: [
                {
                    from: ASSETS_PATH,
                    to: PUBLIC_ASSETS_PATH,
                },
            ],
        }),
    ],
};

module.exports = config;
