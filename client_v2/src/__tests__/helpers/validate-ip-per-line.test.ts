import { describe, it, expect, vi } from 'vitest';

import { copy } from './copy';

// The validator reports localized copy; serve it from the base locale so the
// assertions can name the key instead of re-stating the text.
vi.mock('panel/common/intl', async () => (await import('./copy')).createIntlMock());

import { validateIpPerLine } from 'panel/helpers/validators';

describe('validateIpPerLine', () => {
    it('returns undefined for empty string', () => {
        expect(validateIpPerLine('')).toBeUndefined();
    });

    it('returns undefined for valid IPs', () => {
        expect(validateIpPerLine('192.168.1.1')).toBeUndefined();
        expect(validateIpPerLine('192.168.1.1\n10.0.0.1')).toBeUndefined();
        expect(validateIpPerLine('::1\n192.168.1.1')).toBeUndefined();
    });

    it('returns "Invalid format" for single invalid line', () => {
        expect(validateIpPerLine('not-an-ip')).toBe(copy('form_error_format'));
    });

    it('returns "Invalid format on line 2" when second line is invalid', () => {
        expect(validateIpPerLine('192.168.1.1\nbad-ip')).toBe(
            copy('form_error_format_line', { line: 2 }),
        );
    });

    it('returns "Invalid format on lines 1, 3" when multiple lines invalid', () => {
        expect(validateIpPerLine('bad1\n192.168.1.1\nbad2')).toBe(
            copy('form_error_format_lines', { lines: '1, 3' }),
        );
    });

    it('returns "Invalid format" for single invalid line with trailing newline', () => {
        expect(validateIpPerLine('bad\n')).toBe(copy('form_error_format'));
    });

    it('returns "Invalid format" for single invalid line with leading newline', () => {
        expect(validateIpPerLine('\nbad')).toBe(copy('form_error_format'));
    });

    it('returns "Invalid format on lines 1, 2" when both lines invalid', () => {
        expect(validateIpPerLine('bad\nbad2')).toBe(
            copy('form_error_format_lines', { lines: '1, 2' }),
        );
    });

    it('returns "Invalid format on line 2" when second line invalid in multi-content input', () => {
        expect(validateIpPerLine('192.168.1.1\nbad')).toBe(
            copy('form_error_format_line', { line: 2 }),
        );
    });

    it('returns undefined for all-blank input', () => {
        expect(validateIpPerLine('\n\n')).toBeUndefined();
    });

    it('handles blank line between two invalid lines', () => {
        expect(validateIpPerLine('bad1\n\nbad2')).toBe(
            copy('form_error_format_lines', { lines: '1, 3' }),
        );
    });
});
