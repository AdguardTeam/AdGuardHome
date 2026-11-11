import { describe, it, expect, beforeEach } from 'vitest';
import { readFileSync } from 'node:fs';
import { join, resolve } from 'node:path';

import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import {
    THEME_BOOTSTRAP_PARAMETER,
    THEME_BOOTSTRAP_SOURCE,
    THEME_BOOTSTRAP_TAG,
} from '../../scripts/theme-bootstrap-template.js';

// Every entry inlines the one pre-paint theme script from
// `public/theme-bootstrap.js`; these tests run it the way a browser would.
const publicDir = resolve(process.cwd(), 'public');

const ENTRIES = ['index.html', 'login.html', 'install.html', 'forgot_password.html'];

const THEME_KEY = LOCAL_STORAGE_KEYS.THEME;

const entryHtml = (file: string): string => readFileSync(join(publicDir, file), 'utf8');

const runBootstrap = () => {
    new Function(THEME_BOOTSTRAP_SOURCE)();
};

const appliedTheme = () => document.documentElement.dataset.theme;

describe('theme bootstrap', () => {
    beforeEach(() => {
        localStorage.clear();
        document.documentElement.removeAttribute('data-theme');
        document.documentElement.style.colorScheme = '';
    });

    it('reads the theme key the app writes', () => {
        expect(THEME_BOOTSTRAP_SOURCE).toContain(`'${THEME_KEY}'`);
    });

    it.each(ENTRIES)('inlines the shared script in %s', (file) => {
        // Guard against a copy being pasted back into an entry point.
        expect(entryHtml(file)).toContain(`<%= ${THEME_BOOTSTRAP_PARAMETER} %>`);
    });

    it('wraps the shared source in a raw script tag', () => {
        // `%=` interpolates raw; `<%-` would escape the tag into visible text.
        expect(THEME_BOOTSTRAP_TAG.startsWith('<script>')).toBe(true);
        expect(THEME_BOOTSTRAP_TAG.endsWith('</script>')).toBe(true);
        expect(THEME_BOOTSTRAP_TAG).not.toContain('&lt;');
    });

    it('restores the stored theme before paint', () => {
        localStorage.setItem(THEME_KEY, JSON.stringify('dark'));

        runBootstrap();

        expect(appliedTheme()).toBe('dark');
        expect(document.documentElement.style.colorScheme).toBe('dark');
    });

    it('falls back to the OS preference without a stored theme', () => {
        runBootstrap();

        // `window.matchMedia` is stubbed with `matches: false` by the test setup.
        expect(appliedTheme()).toBe('light');
    });

    it('resolves the "auto" theme through the OS preference', () => {
        localStorage.setItem(THEME_KEY, JSON.stringify('auto'));

        runBootstrap();

        expect(appliedTheme()).toBe('light');
    });

    it('accepts an unparsable stored value as a plain theme name', () => {
        localStorage.setItem(THEME_KEY, 'dark');

        runBootstrap();

        expect(appliedTheme()).toBe('dark');
    });
});
