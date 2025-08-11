const path = require('path');

module.exports = {
    plugins: ['prettier'],
    extends: [
        'airbnb-base',
        'prettier',
        'eslint:recommended',
        'plugin:react/recommended',
        'plugin:@typescript-eslint/recommended',
        'plugin:storybook/recommended',
    ],
    parser: '@typescript-eslint/parser',
    env: {
        jest: true,
        node: true,
        browser: true,
        commonjs: true,
    },
    settings: {
        react: {
            pragma: 'React',
            version: '16.4',
        },
        'import/resolver': {
            typescript: {
                alwaysTryTypes: true,
                project: [path.resolve(__dirname, 'tsconfig.json')],
            },
            node: {
                extensions: ['.js', '.jsx', '.ts', '.tsx'],
            },
        },
    },
    rules: {
        '@typescript-eslint/no-explicit-any': 'off',
        '@typescript-eslint/no-unused-vars': [
            'error',
            {
                argsIgnorePattern: '^_',
            },
        ],
        'import/extensions': [
            'error',
            'ignorePackages',
            {
                js: 'never',
                jsx: 'never',
                ts: 'never',
                tsx: 'never',
            },
        ],
        'class-methods-use-this': 'off',
        'no-shadow': 'off',
        camelcase: 'off',
        'no-console': [
            'warn',
            {
                allow: ['warn', 'error'],
            },
        ],
        'import/no-extraneous-dependencies': [
            'error',
            {
                devDependencies: true,
            },
        ],
        'import/prefer-default-export': 'off',
        'no-alert': 'off',
        'arrow-body-style': 'off',
        'max-len': [
            'error',
            120,
            2,
            {
                ignoreUrls: true,
                ignoreComments: false,
                ignoreRegExpLiterals: true,
                ignoreStrings: true,
                ignoreTemplateLiterals: true,
            },
        ],
    },
};
