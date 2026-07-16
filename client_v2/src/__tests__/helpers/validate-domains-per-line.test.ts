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

import { validateDomainsPerLine } from 'panel/helpers/validators';

describe('validateDomainsPerLine', () => {
    it('returns undefined for empty string', () => {
        expect(validateDomainsPerLine('')).toBeUndefined();
    });

    it('returns undefined for plain domain', () => {
        expect(validateDomainsPerLine('example.org')).toBeUndefined();
    });

    it('returns undefined for wildcard domain', () => {
        expect(validateDomainsPerLine('*.example.org')).toBeUndefined();
    });

    it('returns undefined for AdGuard URL filter rule', () => {
        expect(validateDomainsPerLine('||example.org^')).toBeUndefined();
    });

    it('returns undefined for regex pattern', () => {
        expect(validateDomainsPerLine('/regex.pattern/')).toBeUndefined();
    });

    it('returns undefined for comment line', () => {
        expect(validateDomainsPerLine('# this is a comment')).toBeUndefined();
    });

    it('rejects !-prefixed filter rule that would otherwise pass dot check', () => {
        expect(validateDomainsPerLine('! ||example.org^')).toBeTruthy();
    });

    it('rejects only ! lines as invalid', () => {
        expect(validateDomainsPerLine('! first\n! second')).toBeTruthy();
    });

    it('returns undefined for mixed valid lines with comments', () => {
        expect(
            validateDomainsPerLine('# comment\nexample.org\n||ads.example.org^'),
        ).toBeUndefined();
    });

    it('returns "Invalid format" for entry without dot', () => {
        expect(validateDomainsPerLine('notadomain')).toBe('Invalid format');
    });

    it('returns "Invalid format on line N" when specific line has no dot', () => {
        expect(validateDomainsPerLine('example.org\nnodot')).toBe('Invalid format on line 2');
    });

    it('returns "Invalid format on lines N, M" when multiple lines invalid', () => {
        expect(validateDomainsPerLine('nodot1\nexample.org\nnodot2')).toBe(
            'Invalid format on lines 1, 3',
        );
    });

    it('returns "Invalid format" for single invalid line with trailing newline', () => {
        expect(validateDomainsPerLine('notadomain\n')).toBe('Invalid format');
    });

    it('returns "Invalid format" for single invalid line with leading newline', () => {
        expect(validateDomainsPerLine('\nnotadomain')).toBe('Invalid format');
    });

    it('returns "Invalid format on lines 1, 2" when both lines invalid', () => {
        expect(validateDomainsPerLine('nodot1\nnodot2')).toBe('Invalid format on lines 1, 2');
    });

    it('returns "Invalid format on line 2" when second line invalid in multi-content input', () => {
        expect(validateDomainsPerLine('example.org\nnodot')).toBe('Invalid format on line 2');
    });

    it('returns undefined for all-blank input', () => {
        expect(validateDomainsPerLine('\n\n')).toBeUndefined();
    });

    it('handles blank line between two invalid lines', () => {
        expect(validateDomainsPerLine('nodot1\n\nnodot2')).toBe('Invalid format on lines 1, 3');
    });

    it('returns "Invalid format" for comment-then-invalid (one content line)', () => {
        expect(validateDomainsPerLine('# comment\nnotadomain')).toBe('Invalid format');
    });

    it('returns "Invalid format" for invalid-then-comment (one content line)', () => {
        expect(validateDomainsPerLine('notadomain\n# comment')).toBe('Invalid format');
    });
});
