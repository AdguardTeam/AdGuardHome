const yaml = require('yaml');
const fs = require('fs');

const ZERO_HOST = '0.0.0.0';
const LOCALHOST = '127.0.0.1';
const DEFAULT_PORT = 80;

const importConfig = () => {
    try {
        const doc = yaml.parse(fs.readFileSync('../AdguardHome.yaml', 'utf8'));
        const { bind_host, bind_port } = doc;
        return {
            bind_host,
            bind_port,
        };
    } catch (e) {
        return {
            bind_host: ZERO_HOST,
            bind_port: DEFAULT_PORT,
        };
    }
};

const getDevServerConfig = () => {
    const { bind_host: host, bind_port: port } = importConfig();
    const { DEV_SERVER_PORT } = process.env;

    const devServerHost = host === ZERO_HOST ? LOCALHOST : host;
    const devServerPort = 3000 || port + 8000;

    return {
        host: devServerHost,
        port: devServerPort
    };
};

module.exports = {
    importConfig,
    getDevServerConfig
};
