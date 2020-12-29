module.exports = {
    plugins: [
        ['postcss-import', {}],
        ['postcss-nested', {}],
        ['postcss-custom-media', {}],
        ['postcss-variables', {}],
        ['postcss-calc', {}],
        ['postcss-mixins', {}],
        ['postcss-preset-env', { stage: 3, features: { 'nesting-rules': true } }],
        ['postcss-reporter', { clearMessages: true }],
        ['postcss-inline-svg', {
            paths: ['frontend/icons', 'vendor/adguard/utils-bundle/src/Resources/frontend/icons'],
            svgo: { plugins: [{ cleanupAttrs: true }] } 
        }],
        ['autoprefixer'],
    ]
};
