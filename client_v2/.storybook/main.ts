import type { StorybookConfig } from '@storybook/react-webpack5';
import * as path from 'path';

// Get the project root directory
const projectRoot = path.resolve(process.cwd());

const config: StorybookConfig = {
    stories: ['../src/**/*.mdx', '../src/**/*.stories.@(js|jsx|mjs|ts|tsx)'],
    addons: ['@storybook/addon-webpack5-compiler-swc', '@storybook/addon-docs'],
    framework: {
        name: '@storybook/react-webpack5',
        options: {},
    },
    webpackFinal: async (config) => {
        // Modify existing CSS rules to support PCSS files
        if (config.module?.rules) {
            config.module.rules.forEach((rule) => {
                if (rule && typeof rule !== 'string' && rule.test) {
                    // Find CSS rules and extend them to handle PCSS
                    if (rule.test instanceof RegExp) {
                        // Extend CSS test to include PCSS files
                        if (rule.test.test('.css')) {
                            rule.test = /\.(css|pcss)$/;
                        }
                        // Extend CSS module test to include PCSS modules
                        if (rule.test.toString().includes('module') && rule.test.test('module.css')) {
                            rule.test = /\.module\.(css|pcss)$/;
                        }
                    }
                }
            });
        }

        // Resolve panel alias to match tsconfig.json paths
        if (config.resolve?.alias) {
            config.resolve.alias = {
                ...config.resolve.alias,
                panel: path.resolve(projectRoot, 'src'),
            };
        } else {
            config.resolve = {
                ...config.resolve,
                alias: {
                    panel: path.resolve(projectRoot, 'src'),
                },
            };
        }

        return config;
    },
};
export default config;
