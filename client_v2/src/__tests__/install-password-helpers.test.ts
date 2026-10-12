import { describe, expect, it } from 'vitest';

import {
    hasAllowedChars,
    hasLowercase,
    hasNumberOrSpecial,
    hasUppercase,
} from 'panel/install/Setup/helpers/helpers';

describe('install password requirement helpers', () => {
    describe('hasLowercase', () => {
        it('accepts lowercase letters of any script', () => {
            expect(hasLowercase('pass')).toBe(true);
            expect(hasLowercase('Пароль')).toBe(true);
        });

        it('rejects strings without lowercase letters', () => {
            expect(hasLowercase('P@SSW0RD🔑')).toBe(false);
            expect(hasLowercase('ПАРОЛЬ')).toBe(false);
            expect(hasLowercase('1234🔑')).toBe(false);
        });
    });

    describe('hasUppercase', () => {
        it('accepts uppercase letters of any script', () => {
            expect(hasUppercase('P@ssw0rd')).toBe(true);
            expect(hasUppercase('Пароль')).toBe(true);
        });

        it('rejects strings without uppercase letters', () => {
            expect(hasUppercase('p@ssw0rd🔑')).toBe(false);
            expect(hasUppercase('пароль🔑🔒')).toBe(false);
        });
    });

    describe('hasNumberOrSpecial', () => {
        it('accepts digits, punctuation, symbols, and emoji', () => {
            expect(hasNumberOrSpecial('password1')).toBe(true);
            expect(hasNumberOrSpecial('P@ssw0rd')).toBe(true);
            expect(hasNumberOrSpecial('P@ssw0rd🔑')).toBe(true);
            expect(hasNumberOrSpecial('Пароль🔑🔒')).toBe(true);
        });

        it('rejects letters of any script and spaces', () => {
            expect(hasNumberOrSpecial('Пароль')).toBe(false);
            expect(hasNumberOrSpecial('Pass word')).toBe(false);
        });
    });

    describe('hasAllowedChars', () => {
        it('accepts emoji and printable characters of any script', () => {
            expect(hasAllowedChars('P@ssw0rd🔑')).toBe(true);
            expect(hasAllowedChars('пароль🔑🔒')).toBe(true);
            // ZWJ sequence used by composed emoji.
            expect(hasAllowedChars('👨‍👩‍👧')).toBe(true);
        });

        it('rejects control characters', () => {
            expect(hasAllowedChars('P@ss\nw0rd')).toBe(false);
            expect(hasAllowedChars('P@ss\tw0rd')).toBe(false);
        });
    });
});
