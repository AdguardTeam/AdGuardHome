import { describe, it, expect, vi } from 'vitest';

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: vi.fn((key: string, values?: Record<string, string | number>) => {
            if (key === 'form_error_format_line') {
                return `Invalid format on line ${values?.line}`;
            }
            if (key === 'form_error_format_lines') {
                return `Invalid format on lines ${values?.lines}`;
            }
            if (key === 'form_error_format') {
                return 'Invalid format';
            }
            return key;
        }),
    },
}));

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
        expect(validateIpPerLine('not-an-ip')).toBe('Invalid format');
    });

    it('returns "Invalid format on line 2" when second line is invalid', () => {
        expect(validateIpPerLine('192.168.1.1\nbad-ip')).toBe('Invalid format on line 2');
    });

    it('returns "Invalid format on lines 1, 3" when multiple lines invalid', () => {
        expect(validateIpPerLine('bad1\n192.168.1.1\nbad2')).toBe('Invalid format on lines 1, 3');
    });

    it('returns "Invalid format" for single invalid line with trailing newline', () => {
        expect(validateIpPerLine('bad\n')).toBe('Invalid format');
    });

    it('returns "Invalid format" for single invalid line with leading newline', () => {
        expect(validateIpPerLine('\nbad')).toBe('Invalid format');
    });

    it('returns "Invalid format on lines 1, 2" when both lines invalid', () => {
        expect(validateIpPerLine('bad\nbad2')).toBe('Invalid format on lines 1, 2');
    });

    it('returns "Invalid format on line 2" when second line invalid in multi-content input', () => {
        expect(validateIpPerLine('192.168.1.1\nbad')).toBe('Invalid format on line 2');
    });

    it('returns undefined for all-blank input', () => {
        expect(validateIpPerLine('\n\n')).toBeUndefined();
    });

    it('handles blank line between two invalid lines', () => {
        expect(validateIpPerLine('bad1\n\nbad2')).toBe('Invalid format on lines 1, 3');
    });
});
