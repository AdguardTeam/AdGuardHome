import { describe, it, expect } from 'vitest';

import { withAlpha } from 'panel/helpers/colors';

describe('withAlpha', () => {
    it('replaces the alpha of an rgb() color', () => {
        expect(withAlpha('rgb(103, 178, 121)', 0.3)).toBe('rgba(103, 178, 121, 0.3)');
    });

    it('normalizes an rgba() color to the given alpha', () => {
        expect(withAlpha('rgba(246, 114, 71, 0.8)', 0)).toBe('rgba(246, 114, 71, 0)');
    });

    it('expands a short hex color', () => {
        expect(withAlpha('#abc', 0.5)).toBe('rgba(170, 187, 204, 0.5)');
    });

    it('converts a full hex color', () => {
        expect(withAlpha('#F67247', 0.3)).toBe('rgba(246, 114, 71, 0.3)');
    });

    it('returns unsupported color formats unchanged', () => {
        expect(withAlpha('currentColor', 0.5)).toBe('currentColor');
    });
});
