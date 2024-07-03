module.exports = (api) => {
    api.cache(false);
    return {
        presets: ['@babel/preset-env', '@babel/preset-typescript', '@babel/preset-react'],
        plugins: [
            '@babel/plugin-transform-runtime',
            '@babel/plugin-transform-class-properties',
            '@babel/plugin-transform-object-rest-spread',
            '@babel/plugin-transform-nullish-coalescing-operator',
            '@babel/plugin-transform-optional-chaining',
            'react-hot-loader/babel',
        ],
    };
};
