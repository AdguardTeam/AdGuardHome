import path from 'node:path';
import { fileURLToPath } from 'node:url';
import solid from 'vite-plugin-solid';
import { defineConfig } from 'vitest/config';

const rootDir = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
    plugins: [solid()],
    resolve: {
        conditions: ['development', 'browser'],
        // Force Vite to use a single solid-js instance across all deps,
        // preventing "multiple instances of Solid" from @zag-js/solid, etc.
        dedupe: ['solid-js', 'solid-js/web', 'solid-js/store'],
        alias: {
            panel: path.resolve(rootDir, 'src'),
            Twosky: path.resolve(rootDir, '../.twosky.json'),
            'solid-js': path.resolve(rootDir, 'node_modules/solid-js'),
            'solid-js/web': path.resolve(rootDir, 'node_modules/solid-js/web'),
            'solid-js/store': path.resolve(rootDir, 'node_modules/solid-js/store'),
        },
    },
    ssr: {
        noExternal: [
            '@solidjs/testing-library',
            '@solidjs/router',
            'solid-js',
            // Force @zag-js/* into the same module graph so they share
            // a single solid-js instance, preventing "multiple instances
            // of Solid" errors in CI.
            '@zag-js/solid',
        ],
    },
    test: {
        environment: 'jsdom',
        include: ['src/__tests__/**/*.{test,spec}.{ts,tsx}'],
        setupFiles: ['src/__tests__/setup.ts'],
        css: false,
    },
});
