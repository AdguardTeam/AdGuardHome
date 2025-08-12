import path from 'path';
import fs from 'fs';
import HtmlWebpackPlugin from 'html-webpack-plugin';
import { CleanWebpackPlugin } from 'clean-webpack-plugin';
import CopyPlugin from 'copy-webpack-plugin';
import MiniCssExtractPlugin from 'mini-css-extract-plugin';
import * as url from 'url';
import { BUILD_ENVS } from './constants.js';

// eslint-disable-next-line no-underscore-dangle
const __dirname = url.fileURLToPath(new URL('.', import.meta.url));

const RESOURCES_PATH = path.resolve(__dirname);
const ENTRY_REACT = path.resolve(RESOURCES_PATH, 'src/index.tsx');
const ENTRY_INSTALL = path.resolve(RESOURCES_PATH, 'src/install/index.tsx');
const ENTRY_LOGIN = path.resolve(RESOURCES_PATH, 'src/login/index.tsx');
const HTML_PATH = path.resolve(RESOURCES_PATH, 'public/index.html');
const HTML_INSTALL_PATH = path.resolve(RESOURCES_PATH, 'public/install.html');
const HTML_LOGIN_PATH = path.resolve(RESOURCES_PATH, 'public/login.html');
const ASSETS_PATH = path.resolve(RESOURCES_PATH, 'public/assets');

const PUBLIC_PATH = path.resolve(__dirname, '../build/static');
const PUBLIC_ASSETS_PATH = path.resolve(PUBLIC_PATH, 'assets');

const BUILD_ENV = BUILD_ENVS[process.env.BUILD_ENV];

const isDev = BUILD_ENV === BUILD_ENVS.dev;

function getAliasesFromTsconfig(tsconfigPath, resourcesPath) {
    const tsConfig = JSON.parse(fs.readFileSync(tsconfigPath, 'utf8'));
    const aliases = {};
    if (tsConfig.compilerOptions && tsConfig.compilerOptions.paths) {
        Object.entries(tsConfig.compilerOptions.paths).forEach(([alias, targetArr]) => {
            if (alias.endsWith('/*')) {
                const aliasName = alias.replace('/*', '');
                const target = targetArr[0].replace('/*', '');
                aliases[aliasName] = path.resolve(resourcesPath, target);
            }
        });
    }
    return aliases;
}

const tsConfigPath = path.resolve(__dirname, './tsconfig.json');
const aliasesFromTsconfig = getAliasesFromTsconfig(tsConfigPath, RESOURCES_PATH);

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
        extensions: ['.js', '.jsx', '.ts', '.tsx'],
        alias: {
            ...aliasesFromTsconfig,
        },
    },
    module: {
        rules: [
            {
                test: /\.ya?ml$/,
                type: 'json',
                use: 'yaml-loader',
            },
            {
                test: /\.module\.pcss$/i,
                use: [
                    {
                        loader: MiniCssExtractPlugin.loader,
                    },
                    {
                        loader: 'css-loader',
                        options: {
                            importLoaders: 1,
                            modules: {
                                localIdentName: isDev ? '[name]__[local]--[hash:base64:5]' : '[hash:base64]',
                                namedExport: false,
                                exportLocalsConvention: 'asIs',
                            },
                            esModule: true,
                        },
                    },
                    {
                        loader: 'postcss-loader',
                    },
                ],
            },
            {
                test: /\.p?css$/i,
                exclude: /\.module\.pcss$/i,
                use: [
                    {
                        loader: MiniCssExtractPlugin.loader,
                    },
                    {
                        loader: 'css-loader',
                        options: {
                            importLoaders: 1,
                        },
                    },
                    {
                        loader: 'postcss-loader',
                    },
                ],
            },
            {
                test: /\.tsx?$/,
                exclude: /node_modules/,
                use: {
                    loader: 'ts-loader',
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
            filename: isDev ? '[name].css' : '[name].[hash].css',
            chunkFilename: isDev ? '[id].css' : '[id].[hash].css',
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

export default config;
