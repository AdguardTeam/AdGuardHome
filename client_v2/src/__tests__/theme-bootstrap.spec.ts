import { describe, it, expect, beforeEach } from 'vitest';
import { readFileSync } from 'node:fs';
import { join, resolve } from 'node:path';

import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';

/**
 * The theme must be applied before the first paint, so every HTML entry point
 * carries an inline bootstrap script.  These tests execute that script the way
 * a browser would and check that it restores the stored theme instead of
 * always falling back to the OS preference.
 */
const publicDir = resolve(process.cwd(), 'public');

const ENTRIES = ['index.html', 'login.html', 'install.html', 'forgot_password.html'];

const THEME_KEY = LOCAL_STORAGE_KEYS.THEME;

/** Returns the first inline `<script>` body of an entry point. */
const bootstrapScript = (file: string): string => {
    const html = readFileSync(join(publicDir, file), 'utf8');
    const match = /<script>([\s\S]*?)<\/script>/.exec(html);
    if (!match) {
        throw new Error(`No inline bootstrap script found in ${file}`);
    }

    return match[1];
};

const runBootstrap = (file: string) => {
    new Function(bootstrapScript(file))();
};

const appliedTheme = () => document.documentElement.dataset.theme;

describe('theme bootstrap', () => {
    beforeEach(() => {
        localStorage.clear();
        document.documentElement.removeAttribute('data-theme');
        document.documentElement.style.colorScheme = '';
    });

    it.each(ENTRIES)('reads the theme key the app writes in %s', (file) => {
        expect(bootstrapScript(file)).toContain(`'${LOCAL_STORAGE_KEYS.THEME}'`);
    });

    it.each(ENTRIES)('restores the stored theme before paint in %s', (file) => {
        localStorage.setItem(THEME_KEY, JSON.stringify('dark'));

        runBootstrap(file);

        expect(appliedTheme()).toBe('dark');
        expect(document.documentElement.style.colorScheme).toBe('dark');
    });

    it.each(ENTRIES)('falls back to the OS preference without a stored theme in %s', (file) => {
        runBootstrap(file);

        // `window.matchMedia` is stubbed with `matches: false` by the test setup.
        expect(appliedTheme()).toBe('light');
    });

    it.each(ENTRIES)('resolves the "auto" theme through the OS preference in %s', (file) => {
        localStorage.setItem(THEME_KEY, JSON.stringify('auto'));

        runBootstrap(file);

        expect(appliedTheme()).toBe('light');
    });
});
