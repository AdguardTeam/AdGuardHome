import { defineConfig } from 'vitest/config';
import path from 'path';

export default defineConfig({
    test: {
        environment: 'jsdom',
        include: ['src/__tests__/**'],
    },
    resolve: {
        alias: {
            'panel': path.resolve(__dirname, './src'),
        },
    },
});
