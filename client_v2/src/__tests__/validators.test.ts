import { describe, it, expect } from 'vitest';

import {
    validateHostname,
    validateHostnameNotDuplicate,
    validateLeaseTime,
} from 'panel/helpers/validators';

describe('validateHostname', () => {
    it('accepts hostnames up to the 253-char backend limit', () => {
        expect(validateHostname('a'.repeat(253))).toBeUndefined();
    });

    it('rejects hostnames longer than 253 chars', () => {
        expect(validateHostname('a'.repeat(254))).toBe(
            'Hostname must not be longer than 253 characters',
        );
    });

    it('keeps empty values valid', () => {
        expect(validateHostname('')).toBeUndefined();
    });
});

describe('validateHostnameNotDuplicate', () => {
    const leases = [{ ip: '192.168.1.50', hostname: 'router' }];

    it('rejects a hostname that already exists', () => {
        expect(validateHostnameNotDuplicate(leases)('router')).toBe(
            'This hostname is already added',
        );
    });

    it('is case-insensitive', () => {
        expect(validateHostnameNotDuplicate(leases)('Router')).toBe(
            'This hostname is already added',
        );
    });

    it('excludes the lease being edited', () => {
        expect(validateHostnameNotDuplicate(leases, 'router')('router')).toBeUndefined();
    });

    it('accepts a new hostname', () => {
        expect(validateHostnameNotDuplicate(leases)('printer')).toBeUndefined();
    });
});

describe('validateLeaseTime', () => {
    const maxLeaseTime = 4294967295;
    const formattedMax = maxLeaseTime.toLocaleString();

    it('rejects non-integer values', () => {
        expect(validateLeaseTime('1.5')).toBe(`Enter a value from 1 and ${formattedMax}`);
    });

    it('accepts an integer number of seconds', () => {
        expect(validateLeaseTime('86400')).toBeUndefined();
    });

    it('rejects values above uint32 max', () => {
        expect(validateLeaseTime(String(maxLeaseTime + 1))).toBe(
            `Enter a value from 1 and ${formattedMax}`,
        );
    });
});
