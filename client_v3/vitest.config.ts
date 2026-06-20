import path from 'node:path';
import { fileURLToPath } from 'node:url';
import solid from 'vite-plugin-solid';
import { defineConfig } from 'vitest/config';

const rootDir = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
    plugins: [solid()],
    resolve: {
        conditions: ['development', 'browser'],
        alias: {
            panel: path.resolve(rootDir, 'src'),
            Twosky: path.resolve(rootDir, '../.twosky.json'),
            // Force all solid-js imports to resolve to a single instance,
            // preventing "multiple instances of Solid" errors in CI.
            'solid-js': path.resolve(rootDir, 'node_modules/solid-js'),
            'solid-js/web': path.resolve(rootDir, 'node_modules/solid-js/web'),
            'solid-js/store': path.resolve(rootDir, 'node_modules/solid-js/store'),
        },
    },
    ssr: {
        noExternal: ['@solidjs/testing-library', '@solidjs/router', 'solid-js'],
    },
    test: {
        environment: 'jsdom',
        include: ['src/__tests__/**/*.{test,spec}.{ts,tsx}'],
        setupFiles: ['src/__tests__/setup.ts'],
        css: false,
    },
});
