import { describe, expect, it } from 'vitest';

import {
    msToSeconds,
    msToMinutes,
    msToHours,
    secondsToMilliseconds,
    splitByNewLine,
    trimLinesAndRemoveEmpty,
    normalizeRulesTextarea,
    captitalizeWords,
    getWebAddress,
    getInterfaceIp,
    isIpInCidr,
    parseSubnetMask,
    subnetMaskToBitMask,
} from '../../helpers/helpers';

describe('ms helpers', () => {
    it('converts ms -> seconds', () => {
        expect(msToSeconds(1500)).toBe(1);
    });
    it('converts ms -> minutes', () => {
        expect(msToMinutes(120_000)).toBe(2);
    });
    it('converts ms -> hours', () => {
        expect(msToHours(3_600_000)).toBe(1);
    });
});

describe('secondsToMilliseconds', () => {
    it('multiplies by 1000', () => {
        expect(secondsToMilliseconds(3)).toBe(3000);
    });
    it('returns falsy input as-is', () => {
        expect(secondsToMilliseconds(0)).toBe(0);
        // The current implementation returns `seconds` unchanged if falsy
        expect(secondsToMilliseconds(undefined as unknown as number)).toBe(undefined);
    });
});

describe('splitByNewLine', () => {
    it('splits and removes empty lines', () => {
        expect(splitByNewLine('a\nb\n\nc')).toStrictEqual(['a', 'b', 'c']);
    });
    it('returns [] for falsy input', () => {
        expect(splitByNewLine('')).toStrictEqual([]);
        expect(splitByNewLine(undefined as unknown as string)).toStrictEqual([]);
    });
});

describe('trimLinesAndRemoveEmpty', () => {
    it('trims lines', () => {
        expect(trimLinesAndRemoveEmpty('  a \n\n b ')).toBe('a\nb');
    });
});

describe('normalizeRulesTextarea', () => {
    it('strips leading newlines and collapses repeated blank lines', () => {
        expect(normalizeRulesTextarea('\na\n\nb')).toBe('a\nb');
    });
});

describe('captitalizeWords', () => {
    it('capitalizes each word splitting by space, dash, or underscore', () => {
        expect(captitalizeWords('safe_browsing mode-test')).toBe('Safe Browsing Mode Test');
    });
});

describe('getWebAddress', () => {
    it('builds http url omitting standard port 80', () => {
        expect(getWebAddress('192.168.1.1', 80)).toBe('http://192.168.1.1');
    });
    it('appends non-standard port', () => {
        expect(getWebAddress('192.168.1.1', 8080)).toBe('http://192.168.1.1:8080');
    });
    it('brackets IPv6 with zone encoding', () => {
        expect(getWebAddress('fe80::1%eth0', 80)).toBe('http://[fe80::1%25eth0]');
    });
});

describe('getInterfaceIp', () => {
    it('prefers IPv4 over IPv6', () => {
        expect(getInterfaceIp({ ip_addresses: ['10.0.0.1', 'fe80::1'] })).toBe('10.0.0.1');
    });
    it('skips IPv6 link-local when IPv4 present', () => {
        expect(getInterfaceIp({ ip_addresses: ['192.168.1.1', 'fe80::1'] })).toBe('192.168.1.1');
    });
    it('falls back to IPv6 global without zone', () => {
        expect(getInterfaceIp({ ip_addresses: ['2001:db8::1'] })).toBe('2001:db8::1');
    });
    it('returns undefined when no addresses', () => {
        expect(getInterfaceIp({ ip_addresses: [] })).toBeUndefined();
    });
});

describe('isIpInCidr', () => {
    it('matches IP inside CIDR', () => {
        expect(isIpInCidr('192.168.1.5', '192.168.1.0/24')).toBe(true);
    });
    it('rejects IP outside CIDR', () => {
        expect(isIpInCidr('10.0.0.1', '192.168.1.0/24')).toBe(false);
    });
});

describe('parseSubnetMask', () => {
    it('returns prefix length for valid mask', () => {
        expect(parseSubnetMask('255.255.255.0')).toBe(24);
    });
    it('returns null for invalid mask string', () => {
        expect(parseSubnetMask('not-a-mask')).toBeNull();
    });
});

describe('subnetMaskToBitMask', () => {
    it('computes prefix length from dotted mask', () => {
        expect(subnetMaskToBitMask('255.255.255.0')).toBe(24);
    });
});
