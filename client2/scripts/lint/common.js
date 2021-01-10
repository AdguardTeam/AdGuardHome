module.exports = {
    parser: '@typescript-eslint/parser',
    parserOptions: {
        project: './tsconfig.json',
        ecmaFeatures: {
            jsx: true
        },
        extraFileExtensions: ['mjs', 'tsx', 'ts'],
        ecmaVersion: 2020,
        sourceType: 'module'
    },
    plugins: ['react', '@typescript-eslint', 'import'],
    env: {
        browser: true,
        commonjs: true,
        es6: true,
        es2020: true,
        jest: true,
    },
    settings: {
        react: {
            pragma: 'React',
            version: 'detect',
        },
        'import/resolver': {
            typescript: {
                alwaysTryTypes: true
            }
        },
        'import/parsers': {
            '@typescript-eslint/parser': ['.ts', '.tsx'],
        },
    },
    rules: {
        '@typescript-eslint/explicit-module-boundary-types': 0,
        '@typescript-eslint/explicit-function-return-type': [0, { allowExpressions: true }],
        '@typescript-eslint/indent': ['error', 4],
        '@typescript-eslint/interface-name-prefix': [0, { prefixWithI: 'never' }],
        '@typescript-eslint/no-explicit-any': [0],
        '@typescript-eslint/naming-convention': [2, {
            selector: 'enum', format: ['UPPER_CASE', 'PascalCase'],
        }],
        '@typescript-eslint/no-non-null-assertion': 0,
        'arrow-body-style': 'off',
        'consistent-return': 0,
        curly: [2, 'all'],
        'default-case': 0,
        'import/no-cycle': 0,
        'import/prefer-default-export': 'off',
        'import/no-named-as-default': 0,
        indent: [0, 4],
        'no-alert': 2,
        'no-console': 2,
        'no-debugger': 2,
        'no-underscore-dangle': 'off',
        'no-useless-escape': 'off',
        'object-curly-newline': 'off',
        'react-hooks/exhaustive-deps': 0,
        'react/display-name': 0,
        'react/jsx-indent-props': ['error', 4],
        'react/jsx-indent': ['error', 4],
        'react/jsx-one-expression-per-line': 'off',
        'react/jsx-props-no-spreading': 0,
        'react/prop-types': 'off',
        'react/state-in-constructor': 'off',
    },
    extends: [
        'airbnb-base',
        'airbnb-typescript/base',
        'airbnb/hooks',
        'plugin:react/recommended',
        'plugin:@typescript-eslint/eslint-recommended',
        'plugin:@typescript-eslint/recommended',
        'plugin:import/errors',
        'plugin:import/warnings',
        'plugin:import/typescript',
    ],
    globals: {},
};
