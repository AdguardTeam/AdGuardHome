import fs from 'node:fs';
import { copyFile, mkdir, readdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import type { StorybookConfig } from 'storybook-solidjs-vite';

import { getAliasesFromTsconfig } from '../scripts/tsconfig-aliases.js';

const rootDir = path.dirname(fileURLToPath(import.meta.url));
const clientDir = path.resolve(rootDir, '..');

// The same reader webpack and Vitest use, so tsconfig.json stays the only
// place an alias has to be registered.
const aliases = getAliasesFromTsconfig(path.resolve(clientDir, 'tsconfig.json'), clientDir);

// Alias targets outside `client_v2` (only `.twosky.json` today) must be listed
// too: Vite checks `fs.allow` per file, so the rest of the checkout, its
// git-ignored config included, stays unreadable over `/@fs/`.
const externalAliasTargets = Object.values(aliases).filter(
    (target) => target !== clientDir && !target.startsWith(`${clientDir}${path.sep}`),
);

const FONTS_DIR = path.resolve(clientDir, 'src/common/styles/fonts');
const FONTS_URL = '/common/styles/fonts/';

const config: StorybookConfig = {
    stories: ['../src/**/*.stories.tsx'],
    addons: ['@storybook/addon-docs'],
    framework: 'storybook-solidjs-vite',

    async viteFinal(viteConfig) {
        viteConfig.resolve ??= {};
        // @zag-js/solid and @ark-ui/solid import solid-js themselves, so a
        // second copy would break reactivity inside the preview.
        viteConfig.resolve.dedupe = [...(viteConfig.resolve.dedupe ?? []), 'solid-js'];

        const alias = viteConfig.resolve.alias;
        viteConfig.resolve.alias = Array.isArray(alias)
            ? [
                  ...alias,
                  ...Object.entries(aliases).map(([find, replacement]) => ({ find, replacement })),
              ]
            : { ...alias, ...aliases };

        // CSS modules are exported with their original class names by default,
        // which is what webpack does too (`s.height_xs`).  Keep it that way:
        // `localsConvention: 'asIs'` is not a Vite value and empties the map.

        viteConfig.server ??= {};
        viteConfig.server.fs = {
            ...viteConfig.server.fs,
            allow: [...(viteConfig.server.fs?.allow ?? []), clientDir, ...externalAliasTargets],
        };

        // `client_v2/public` holds the application's HTML entry points; copying
        // them into the catalog would overwrite Storybook's own `index.html`.
        viteConfig.publicDir = false;

        // `fonts.css` addresses the font files by their published path under
        // `build/static/`, which the preview does not serve.  Serve them from
        // the same URL in development and place them next to the built CSS, so
        // the relative URLs resolve in both modes.
        let outDir = path.resolve(clientDir, 'storybook-static');
        let isBuild = false;

        viteConfig.plugins = [
            ...(viteConfig.plugins ?? []),
            {
                name: 'agh-home-storybook-fonts',
                configResolved(resolved) {
                    outDir = resolved.build.outDir;
                    isBuild = resolved.command === 'build';
                },
                configureServer(server) {
                    server.middlewares.use((req, res, next) => {
                        if (!req.url?.startsWith(FONTS_URL)) {
                            return next();
                        }

                        const file = path.join(FONTS_DIR, path.basename(req.url.split('?')[0]));
                        if (!fs.existsSync(file)) {
                            return next();
                        }

                        res.setHeader('Content-Type', 'font/woff2');
                        fs.createReadStream(file).pipe(res);
                    });
                },
                async closeBundle() {
                    if (!isBuild) {
                        return;
                    }

                    const targetDir = path.join(outDir, 'assets', 'common/styles/fonts');
                    await mkdir(targetDir, { recursive: true });

                    for (const name of await readdir(FONTS_DIR)) {
                        await copyFile(path.join(FONTS_DIR, name), path.join(targetDir, name));
                    }
                },
            },
        ];

        return viteConfig;
    },
};

export default config;
