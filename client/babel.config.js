module.exports = (api) => {
    api.cache(false);
    return {
        presets: [
            '@babel/preset-env',
            '@babel/preset-react',
        ],
        plugins: [
            '@babel/plugin-proposal-class-properties',
            '@babel/plugin-transform-runtime',
            '@babel/plugin-proposal-object-rest-spread',
            '@babel/plugin-proposal-nullish-coalescing-operator',
            '@babel/plugin-proposal-optional-chaining',
            'react-hot-loader/babel',
        ],
    };
};
