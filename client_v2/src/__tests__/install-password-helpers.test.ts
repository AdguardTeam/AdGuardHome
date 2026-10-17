import { describe, expect, it } from 'vitest';

import { hasLowercase, hasUppercase } from 'panel/install/Setup/helpers/helpers';

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
});
