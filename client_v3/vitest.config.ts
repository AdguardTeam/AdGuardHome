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
