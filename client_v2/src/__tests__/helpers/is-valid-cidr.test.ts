import { describe, expect, it } from 'vitest';
import { isValidCidr } from 'panel/helpers/helpers';

describe('isValidCidr', () => {
    it('accepts IPv4 CIDR ranges', () => {
        expect(isValidCidr('192.168.1.0/24')).toBe(true);
        expect(isValidCidr('10.0.0.0/8')).toBe(true);
        expect(isValidCidr('0.0.0.0/0')).toBe(true);
    });

    it('accepts IPv6 CIDR ranges', () => {
        expect(isValidCidr('2001:db8::/64')).toBe(true);
        expect(isValidCidr('::/0')).toBe(true);
    });

    // REGRESSION: GH #8610 — the retired R_CIDR_IPV6 used a literal `d` where it
    // needed `\d`, so every mixed-notation range was rejected while the
    // equivalent hex form was accepted.
    it('accepts IPv6 CIDR ranges with an embedded dotted-decimal IPv4 address', () => {
        expect(isValidCidr('::ffff:192.168.1.0/120')).toBe(true);
        expect(isValidCidr('::ffff:10.0.0.0/104')).toBe(true);
    });

    it('accepts the hexadecimal form of a mixed-notation range', () => {
        expect(isValidCidr('::ffff:c0a8:100/120')).toBe(true);
    });

    it('rejects IPv6 CIDR ranges with a zone ID', () => {
        // netip.ParsePrefix rejects zones in a prefix, even though
        // netip.ParseAddr accepts them on a bare address.
        expect(isValidCidr('fe80::1%eth0/64')).toBe(false);
    });

    it('rejects prefix lengths spelled with leading zeros', () => {
        expect(isValidCidr('192.168.1.0/024')).toBe(false);
        expect(isValidCidr('2001:db8::/064')).toBe(false);
    });

    it('rejects non-canonical IPv4 addresses', () => {
        expect(isValidCidr('127.1/24')).toBe(false);
        expect(isValidCidr('0x7f.0.0.1/8')).toBe(false);
        expect(isValidCidr('010.0.0.1/8')).toBe(false);
        expect(isValidCidr('12345/8')).toBe(false);
    });

    it('rejects out-of-range octets', () => {
        expect(isValidCidr('::ffff:192.168.1.256/120')).toBe(false);
        expect(isValidCidr('192.168.1.256/24')).toBe(false);
    });

    it('rejects octets spelled with a literal `d`', () => {
        expect(isValidCidr('::ffff:20d.d.d.d/120')).toBe(false);
    });

    it('rejects prefix lengths outside the address family range', () => {
        expect(isValidCidr('192.168.1.0/33')).toBe(false);
        expect(isValidCidr('2001:db8::/129')).toBe(false);
    });

    it('rejects an address without a prefix', () => {
        expect(isValidCidr('192.168.1.0')).toBe(false);
        expect(isValidCidr('2001:db8::')).toBe(false);
    });

    it('rejects empty and malformed input', () => {
        expect(isValidCidr('')).toBe(false);
        expect(isValidCidr('not-a-cidr')).toBe(false);
    });
});
