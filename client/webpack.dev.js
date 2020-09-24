const merge = require('webpack-merge');
const yaml = require('js-yaml');
const fs = require('fs');
const common = require('./webpack.common.js');
const { BASE_URL } = require('./constants');

const ZERO_HOST = '0.0.0.0';
const LOCALHOST = '127.0.0.1';
const DEFAULT_PORT = 80;

/**
 * Get document, or throw exception on error
 * @returns {{bind_host: string, bind_port: number}}
 */
const importConfig = () => {
    try {
        const doc = yaml.safeLoad(fs.readFileSync('../AdguardHome.yaml', 'utf8'));
        const { bind_host, bind_port } = doc;
        return {
            bind_host,
            bind_port,
        };
    } catch (e) {
        console.error(e);
        return {
            bind_host: ZERO_HOST,
            bind_port: DEFAULT_PORT,
        };
    }
};

const getDevServerConfig = (proxyUrl = BASE_URL) => {
    const { bind_host: host, bind_port: port } = importConfig();
    const { DEV_SERVER_PORT } = process.env;

    const devServerHost = host === ZERO_HOST ? LOCALHOST : host;
    const devServerPort = DEV_SERVER_PORT || port + 8000;

    return {
        hot: true,
        open: true,
        host: devServerHost,
        port: devServerPort,
        proxy: {
            [proxyUrl]: `http://${devServerHost}:${port}`,
        },
        open: true,
    };
};

module.exports = merge(common, {
    devtool: 'eval-source-map',
    module: {
        rules: [
            {
                test: /\.js$/,
                exclude: /node_modules/,
                loader: 'eslint-loader',
                options: {
                    emitWarning: true,
                    configFile: 'dev.eslintrc',
                },
            },
        ],
    },
    ...(process.env.WEBPACK_DEV_SERVER ? { devServer: getDevServerConfig(BASE_URL) } : undefined),
});
