import { describe, it, expect } from 'vitest';
import { optionToValue, filterOptions, getItemTestId } from '../common/controls/Select/helpers';

const OPTIONS = [
    { value: 'apple', label: 'Apple' },
    { value: 'banana', label: 'Banana' },
    { value: null, label: 'No value' },
    { value: 42, label: 'Number' },
];

describe('optionToValue', () => {
    it('returns empty string for null', () => {
        expect(optionToValue(null)).toBe('');
    });

    it('returns empty string for undefined', () => {
        expect(optionToValue(undefined)).toBe('');
    });

    it('returns the string for string values', () => {
        expect(optionToValue('hello')).toBe('hello');
    });

    it('returns the string representation for numbers', () => {
        expect(optionToValue(42)).toBe('42');
    });

    it('preserves falsy number 0 (does NOT return empty string)', () => {
        // Bug #1 regression test: 0 is falsy but a valid value — must not be
        // coerced to '' (which would break selection matching).
        expect(optionToValue(0)).toBe('0');
    });

    it('returns "true" for boolean true', () => {
        expect(optionToValue(true)).toBe('true');
    });

    it('returns "false" for boolean false', () => {
        // false is falsy but a valid value — must not be coerced to ''.
        expect(optionToValue(false)).toBe('false');
    });

    it('stringifies objects via toString()', () => {
        expect(optionToValue({})).toBe('[object Object]');
    });
});

describe('filterOptions', () => {
    it('returns all options when query is empty', () => {
        expect(filterOptions(OPTIONS, '')).toEqual(OPTIONS);
    });

    it('returns the same array reference when query is empty (no copy)', () => {
        const result = filterOptions(OPTIONS, '');
        expect(result).toBe(OPTIONS);
    });

    it('matches by label (case-insensitive)', () => {
        const result = filterOptions(OPTIONS, 'app');
        expect(result).toEqual([{ value: 'apple', label: 'Apple' }]);
    });

    it('matches partial labels', () => {
        const result = filterOptions(OPTIONS, 'ban');
        expect(result).toHaveLength(1);
        expect(result[0].label).toBe('Banana');
    });

    it('returns empty array when no match', () => {
        const result = filterOptions(OPTIONS, 'xyz');
        expect(result).toEqual([]);
    });

    it('does NOT match by value (only by label)', () => {
        const opts = [{ value: 'hidden-id', label: 'Visible label' }];
        const result = filterOptions(opts, 'hidden');
        expect(result).toEqual([]);
    });

    it('matches case-insensitively (uppercase query)', () => {
        const result = filterOptions(OPTIONS, 'BANANA');
        expect(result).toHaveLength(1);
    });

    it('handles options with non-string labels by stringifying them', () => {
        const opts = [{ value: 1, label: 123 }];
        const result = filterOptions(opts, '12');
        expect(result).toHaveLength(1);
    });
});

describe('getItemTestId', () => {
    it('returns undefined when no prefix is given', () => {
        expect(getItemTestId(undefined, 'apple')).toBeUndefined();
    });

    it('returns "prefix-value" for string values', () => {
        expect(getItemTestId('option', 'apple')).toBe('option-apple');
    });

    it('returns "prefix-" for null values (empty value after dash)', () => {
        // Uses optionToValue which converts null → ''.
        expect(getItemTestId('option', null)).toBe('option-');
    });

    it('returns "prefix-42" for number values', () => {
        expect(getItemTestId('option', 42)).toBe('option-42');
    });

    it('preserves falsy number 0', () => {
        expect(getItemTestId('option', 0)).toBe('option-0');
    });

    it('returns undefined when prefix is empty string', () => {
        expect(getItemTestId('', 'apple')).toBeUndefined();
    });
});
