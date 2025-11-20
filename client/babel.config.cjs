module.exports = (api) => {
    api.cache(false);
    return {
        presets: [
            '@babel/preset-env',
            '@babel/preset-typescript',
            ['@babel/preset-react', { runtime: 'automatic' }],
        ],
        plugins: [
            // Keep transform-runtime for helper reuse and polyfill consistency
            '@babel/plugin-transform-runtime',
        ],
    };
};
