import { describe, it, expect, beforeEach } from 'vitest';

import { THEMES } from 'panel/helpers/constants';
import { getTheme, setUITheme } from 'panel/helpers/helpers';
import { mockMatchMedia } from './matchMedia';

const appliedTheme = () => document.documentElement.dataset.theme;

describe('getTheme', () => {
    beforeEach(() => localStorage.clear());

    it('defaults to the "auto" theme when nothing is stored', () => {
        expect(getTheme()).toBe(THEMES.auto);
    });
});

describe('setUITheme', () => {
    beforeEach(() => {
        localStorage.clear();
        document.documentElement.removeAttribute('data-theme');
        document.documentElement.style.colorScheme = '';
        mockMatchMedia(false);
    });

    it('applies and persists an explicit theme', () => {
        setUITheme(THEMES.dark);

        expect(appliedTheme()).toBe(THEMES.dark);
        expect(document.documentElement.style.colorScheme).toBe(THEMES.dark);
        expect(getTheme()).toBe(THEMES.dark);
    });

    it('resolves "auto" to the light OS preference and keeps it in storage', () => {
        setUITheme(THEMES.auto);

        expect(appliedTheme()).toBe(THEMES.light);
        expect(getTheme()).toBe(THEMES.auto);
    });

    it('resolves "auto" to the dark OS preference and keeps it in storage', () => {
        mockMatchMedia(true);

        setUITheme(THEMES.auto);

        expect(appliedTheme()).toBe(THEMES.dark);
        expect(document.documentElement.style.colorScheme).toBe(THEMES.dark);
        expect(getTheme()).toBe(THEMES.auto);
    });
});
