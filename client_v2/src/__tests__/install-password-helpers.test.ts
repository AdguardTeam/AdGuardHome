import { describe, expect, it } from 'vitest';

import {
    hasAllowedChars,
    hasLowercase,
    hasNumberOrSpecial,
    hasUppercase,
} from 'panel/install/Setup/helpers/helpers';

describe('install password requirement helpers', () => {
    describe('hasLowercase', () => {
        it('accepts lowercase letters beyond ASCII', () => {
            expect(hasLowercase('pass')).toBe(true);
            expect(hasLowercase('åäö')).toBe(true);
        });

        it('rejects strings without lowercase letters', () => {
            expect(hasLowercase('P@SSW0RD🔑')).toBe(false);
            expect(hasLowercase('ÅÄÖ')).toBe(false);
            expect(hasLowercase('1234🔑')).toBe(false);
        });
    });

    describe('hasUppercase', () => {
        it('accepts uppercase letters beyond ASCII', () => {
            expect(hasUppercase('P@ssw0rd')).toBe(true);
            expect(hasUppercase('ÅÄÖ')).toBe(true);
        });

        it('rejects strings without uppercase letters', () => {
            expect(hasUppercase('p@ssw0rd🔑')).toBe(false);
            expect(hasUppercase('åäö🔑🔒')).toBe(false);
        });
    });

    describe('hasNumberOrSpecial', () => {
        it('accepts digits, punctuation, symbols, and emoji', () => {
            expect(hasNumberOrSpecial('password1')).toBe(true);
            expect(hasNumberOrSpecial('P@ssw0rd')).toBe(true);
            expect(hasNumberOrSpecial('P@ssw0rd🔑')).toBe(true);
            expect(hasNumberOrSpecial('ÅÄÖ🔑🔒')).toBe(true);
        });

        it('rejects letters beyond ASCII and spaces', () => {
            expect(hasNumberOrSpecial('ÅÄÖ')).toBe(false);
            expect(hasNumberOrSpecial('Pass word')).toBe(false);
        });
    });

    describe('hasAllowedChars', () => {
        it('accepts emoji and printable characters beyond ASCII', () => {
            expect(hasAllowedChars('P@ssw0rd🔑')).toBe(true);
            expect(hasAllowedChars('åäö🔑🔒')).toBe(true);
            // ZWJ sequence used by composed emoji.
            expect(hasAllowedChars('👨‍👩‍👧')).toBe(true);
        });

        it('rejects control characters', () => {
            expect(hasAllowedChars('P@ss\nw0rd')).toBe(false);
            expect(hasAllowedChars('P@ss\tw0rd')).toBe(false);
        });
    });
});
