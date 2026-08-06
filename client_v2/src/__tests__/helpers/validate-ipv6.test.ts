import { describe, expect, it } from 'vitest';
import { isValidIpv6 } from 'panel/helpers/helpers';

describe('isValidIpv6', () => {
    it('accepts standard IPv6 address', () => {
        expect(isValidIpv6('2001:db8::1')).toBe(true);
    });

    it('accepts IPv6 with zone ID', () => {
        expect(isValidIpv6('fe80::1%eth0')).toBe(true);
        expect(isValidIpv6('fe80::1%wlan0')).toBe(true);
    });

    it('accepts link-local IPv6 without zone', () => {
        expect(isValidIpv6('fe80::1')).toBe(true);
    });

    it('accepts loopback', () => {
        expect(isValidIpv6('::1')).toBe(true);
    });

    it('rejects invalid IPv6', () => {
        expect(isValidIpv6('not-an-ip')).toBe(false);
        expect(isValidIpv6('192.168.1.1')).toBe(false);
        expect(isValidIpv6('gggg::1')).toBe(false);
    });

    it('rejects empty string', () => {
        expect(isValidIpv6('')).toBe(false);
    });
});

import { validateIpv6, validateIp } from 'panel/helpers/validators';

describe('validateIpv6 with ipaddr.js', () => {
    it('passes for IPv6 with zone ID', () => {
        expect(validateIpv6('fe80::1%eth0')).toBeUndefined();
    });

    it('passes for IPv6 with zone ID (non-link-local)', () => {
        expect(validateIpv6('2001:db8::1%eth0')).toBeUndefined();
    });

    it('passes for IPv6 without zone', () => {
        expect(validateIpv6('2001:db8::1')).toBeUndefined();
    });

    it('fails for invalid IPv6', () => {
        expect(validateIpv6('not-an-ip')).toBeTruthy();
    });

    it('returns undefined for empty value', () => {
        expect(validateIpv6('')).toBeUndefined();
    });
});

describe('validateIp with ipaddr.js', () => {
    it('passes for IPv6 with zone ID', () => {
        expect(validateIp('fe80::1%eth0')).toBeUndefined();
    });

    it('passes for IPv6 with zone ID (non-link-local)', () => {
        expect(validateIp('2001:db8::1%eth0')).toBeUndefined();
    });

    it('passes for IPv4', () => {
        expect(validateIp('192.168.1.1')).toBeUndefined();
    });

    it('fails for invalid IP', () => {
        expect(validateIp('not-an-ip')).toBeTruthy();
    });
});
