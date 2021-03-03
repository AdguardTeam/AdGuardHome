const history = require('connect-history-api-fallback');
const { merge } = require('webpack-merge');
const path = require('path');
const proxy = require('http-proxy-middleware');
const Webpack = require('webpack');

const { getDevServerConfig } = require('./helpers');
const baseConfig = require('./webpack.config.base');

const devHost = process.env.DEV_HOST
const target = getDevServerConfig();

const options = {
    target: devHost || `http://${target.host}:${target.port}`, // target host
    changeOrigin: true, // needed for virtual hosted sites
};
const apiProxy = proxy.createProxyMiddleware(options);

module.exports = merge(baseConfig, {
    mode: 'development',
    output: {
        path: path.resolve(__dirname, '../../build2'),
        filename: '[name].bundle.js',
    },
    optimization: {
        noEmitOnErrors: true,
    },
    devServer: {
        port: 4000,
        historyApiFallback: true,
        before: (app) => {
            app.use('/control', apiProxy);
            app.use(history({
                rewrites: [
                    {
                        from: /\.(png|jpe?g|gif)$/,
                        to: (context) => {
                            const name = context.parsedUrl.pathname.split('/');
                            return `/images/${name[name.length - 1]}`
                        }
                    }, {
                        from: /\.(woff|woff2)$/,
                        to: (context) => {
                            const name = context.parsedUrl.pathname.split('/');
                            return `/${name[name.length - 1]}`
                        }
                    }, {
                        from: /\.(js|css)$/,
                        to: (context) => {
                            const name = context.parsedUrl.pathname.split('/');
                            return `/${name[name.length - 1]}`
                        }
                    }
                ],
            }));
        }
    },
    devtool: 'eval-source-map',
    module: {
        rules: [
            {
                enforce: 'pre',
                test: /\.tsx?$/,
                exclude: /node_modules/,
                loader: 'eslint-loader',
                options: {
                    configFile: path.resolve(__dirname, '../lint/dev.js'),
                }
            },
            {
                test: (resource) => {
                    return (
                        resource.indexOf('.pcss')+1
                        || resource.indexOf('.css')+1
                        || resource.indexOf('.less')+1
                    ) && !(resource.indexOf('.module.')+1);
                },
                use: ['style-loader', 'css-loader', 'postcss-loader', {
                    loader: 'less-loader',
                    options: {
                        javascriptEnabled: true,
                    },
                }],
            },
            {
                test: /\.module\.p?css$/,
                use: [
                        'style-loader',
                        {
                            loader: 'css-loader',
                            options: {
                                modules: true,
                                sourceMap: true,
                                importLoaders: 1,
                                modules: {
                                    localIdentName: "[name]__[local]___[hash:base64:5]",
                                }
                            },
                        },
                        'postcss-loader',
                ],
                exclude: /node_modules/,
            },
        ]
    },
    plugins: [
        new Webpack.DefinePlugin({
            DEV: true,
            'process.env.DEV_SERVER_PORT': JSON.stringify(3000),
        }),
        new Webpack.HotModuleReplacementPlugin(),
        new Webpack.ProgressPlugin(),
    ],
});
