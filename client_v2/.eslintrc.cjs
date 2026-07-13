const path = require('path');

module.exports = {
    plugins: ['prettier', 'solid', 'import'],
    extends: [
        'prettier',
        'eslint:recommended',
        'plugin:@typescript-eslint/recommended',
        'plugin:solid/typescript',
    ],
    parser: '@typescript-eslint/parser',
    env: {
        node: true,
        browser: true,
        commonjs: true,
    },
    settings: {
        'import/core-modules': ['Twosky'],
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
                caughtErrorsIgnorePattern: '^_',
            },
        ],
        'import/extensions': 'off',
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
