module.exports = (api) => {
    api.cache(false);
    return {
        presets: [
            '@babel/preset-env',
            '@babel/preset-typescript',
            'babel-preset-solid',
        ],
    };
};
