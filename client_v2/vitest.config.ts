import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

const rootDir = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
    resolve: {
        alias: {
            panel: path.resolve(rootDir, 'src'),
            Twosky: path.resolve(rootDir, '../.twosky.json'),
        },
    },
    test: {
        environment: 'jsdom',
        include: ['src/__tests__/**/*.{test,spec}.{ts,tsx}'],
        setupFiles: ['src/__tests__/setup.ts'],
        css: false,
    },
});
