import { describe, it, expect } from 'vitest';

import { resolveCssColor, withAlpha } from 'panel/helpers/colors';

describe('resolveCssColor', () => {
    it('returns the fallback when the custom property is not defined', () => {
        expect(resolveCssColor('--missing-test-color', '#123456')).toBe('#123456');
    });
});

describe('withAlpha', () => {
    it('adds alpha to an rgb() value', () => {
        expect(withAlpha('rgb(103, 178, 121)', 0.3)).toBe('rgba(103, 178, 121, 0.3)');
    });

    it('replaces alpha in an rgba() value', () => {
        expect(withAlpha('rgba(1, 2, 3, 0.4)', 0)).toBe('rgba(1, 2, 3, 0)');
    });

    it('expands a short hex value', () => {
        expect(withAlpha('#fff', 0.5)).toBe('rgba(255, 255, 255, 0.5)');
    });

    it('adds alpha to a full hex value', () => {
        expect(withAlpha('#67b279', 0)).toBe('rgba(103, 178, 121, 0)');
    });

    it('returns unsupported color formats unchanged', () => {
        expect(withAlpha('currentColor', 0.3)).toBe('currentColor');
    });
});
