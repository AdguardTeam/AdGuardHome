import { describe, it, expect, vi } from 'vitest';

import { copy } from './copy';

// The validator reports localized copy; serve it from the base locale so the
// assertions can name the key instead of re-stating the text.
vi.mock('panel/common/intl', async () => (await import('./copy')).createIntlMock());

import { validateClientsPerLine } from 'panel/helpers/validators';

describe('validateClientsPerLine', () => {
    it('returns undefined for empty string', () => {
        expect(validateClientsPerLine('')).toBeUndefined();
    });

    it('returns undefined for valid IPv4', () => {
        expect(validateClientsPerLine('192.168.1.1')).toBeUndefined();
    });

    it('returns undefined for valid IPv6', () => {
        expect(validateClientsPerLine('::1')).toBeUndefined();
        expect(validateClientsPerLine('2001:db8::1')).toBeUndefined();
    });

    it('returns undefined for valid IPv4 CIDR', () => {
        expect(validateClientsPerLine('192.168.1.0/24')).toBeUndefined();
        expect(validateClientsPerLine('10.0.0.0/8')).toBeUndefined();
    });

    it('returns undefined for valid IPv6 CIDR', () => {
        expect(validateClientsPerLine('2001:db8::/32')).toBeUndefined();
    });

    it('returns undefined for valid ClientID', () => {
        expect(validateClientsPerLine('my-client-id')).toBeUndefined();
        expect(validateClientsPerLine('client-123')).toBeUndefined();
    });

    it('returns undefined for mixed valid lines', () => {
        expect(validateClientsPerLine('192.168.1.1\n10.0.0.0/8\nmy-client-id')).toBeUndefined();
    });

    it('returns "Invalid format" for single invalid line', () => {
        expect(validateClientsPerLine('BAD ENTRY!')).toBe(copy('form_error_format'));
    });

    it('returns "Invalid format on line N" when specific line is invalid', () => {
        expect(validateClientsPerLine('192.168.1.1\nBAD!')).toBe(
            copy('form_error_format_line', { line: 2 }),
        );
    });

    it('returns "Invalid format on lines N, M" when multiple lines invalid', () => {
        expect(validateClientsPerLine('BAD1!\n192.168.1.1\nBAD2!')).toBe(
            copy('form_error_format_lines', { lines: '1, 3' }),
        );
    });

    it('rejects MAC address', () => {
        expect(validateClientsPerLine('aa:bb:cc:dd:ee:ff')).toBe(copy('form_error_format'));
    });

    it('rejects CIDR with bad prefix', () => {
        expect(validateClientsPerLine('192.168.1.0/33')).toBe(copy('form_error_format'));
    });

    it('returns "Invalid format" for single invalid line with trailing newline', () => {
        expect(validateClientsPerLine('BAD ENTRY!\n')).toBe(copy('form_error_format'));
    });

    it('returns "Invalid format" for single invalid line with leading newline', () => {
        expect(validateClientsPerLine('\nBAD ENTRY!')).toBe(copy('form_error_format'));
    });

    it('returns "Invalid format on lines 1, 2" when both lines invalid', () => {
        expect(validateClientsPerLine('BAD1!\nBAD2!')).toBe(
            copy('form_error_format_lines', { lines: '1, 2' }),
        );
    });

    it('returns "Invalid format on line 2" when second line invalid in multi-content input', () => {
        expect(validateClientsPerLine('192.168.1.1\nBAD!')).toBe(
            copy('form_error_format_line', { line: 2 }),
        );
    });

    it('returns undefined for all-blank input', () => {
        expect(validateClientsPerLine('\n\n')).toBeUndefined();
    });

    it('handles blank line between two invalid lines', () => {
        expect(validateClientsPerLine('BAD1!\n\nBAD2!')).toBe('Invalid format on lines 1, 3');
    });
});
