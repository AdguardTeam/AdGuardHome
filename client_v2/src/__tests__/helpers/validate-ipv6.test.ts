import { describe, expect, it } from 'vitest';
import { isValidIpv6 } from 'panel/helpers/helpers';

describe('isValidIpv6', () => {
    it('accepts standard IPv6 address', () => {
        expect(isValidIpv6('2001:db8::1')).toBe(true);
    });

    it('accepts IPv6 with zone ID', () => {
        expect(isValidIpv6('fe80::1%eth0')).toBe(true);
        expect(isValidIpv6('fe80::1%wlan0')).toBe(true);
        expect(isValidIpv6('fe80::%eth0')).toBe(true);
    });

    // REGRESSION: netip.ParseAddr treats everything after the first `%` as the
    // zone ID without restricting its contents, so the server accepts this
    // value as an address with a zone ID, not as a CIDR.  ipaddr.js only takes
    // [0-9a-z]+ zones and anchors them, so the zone is split off first.
    it('accepts a zone ID with any non-empty contents', () => {
        expect(isValidIpv6('fe80::1%eth0/64')).toBe(true);
        expect(isValidIpv6('2001:db8::1%eth0')).toBe(true);
        expect(isValidIpv6('fe80::1%a-b')).toBe(true);
    });

    it('rejects an empty zone ID', () => {
        expect(isValidIpv6('fe80::1%')).toBe(false);
    });

    it('rejects a zone ID on an address with no group after fe80:', () => {
        expect(isValidIpv6('fe80:%eth0')).toBe(false);
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

    it('returns false for a non-canonical embedded IPv4 tail', () => {
        expect(isValidIpv6('::ffff:192.168.01.1')).toBe(false);
    });

    it('returns true for a canonical embedded IPv4 tail', () => {
        expect(isValidIpv6('::ffff:192.168.1.1')).toBe(true);
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

    it('rejects a non-canonical embedded IPv4 tail', () => {
        expect(validateIp('::ffff:192.168.01.1')).toBeTruthy();
    });
});
