import { describe, expect, it } from 'vitest';
import {
    validateIdentifier,
    validateUpstreams,
    validateRequiredValue,
    validateIpv4,
    validatePort,
    validateInstallPort,
    validatePlainDns,
    validateIpForGatewaySubnetMask,
    validateIpv4InCidr,
    validatePasswordLength,
    validateIpNotDuplicate,
    validateIpPerLine,
    validateBetween,
    validateMinValue,
    validateCacheSize,
    validateRewriteNotExists,
    validateRewriteNotSame,
    validateLeaseTime,
} from 'panel/helpers/validators';

describe('validateIdentifier', () => {
    it('returns required error for empty string', () => {
        const result = validateIdentifier('', [], 0);
        expect(result).toBeTruthy();
    });

    it('returns required error for whitespace-only string', () => {
        const result = validateIdentifier('   ', [], 0);
        expect(result).toBeTruthy();
    });

    it('returns undefined for valid IPv4', () => {
        const result = validateIdentifier('192.168.1.1', ['192.168.1.1'], 0);
        expect(result).toBeUndefined();
    });

    it('returns undefined for valid IPv6', () => {
        const result = validateIdentifier('::1', ['::1'], 0);
        expect(result).toBeUndefined();
    });

    it('returns undefined for valid MAC', () => {
        const result = validateIdentifier('aa:bb:cc:dd:ee:ff', ['aa:bb:cc:dd:ee:ff'], 0);
        expect(result).toBeUndefined();
    });

    it('returns undefined for valid CIDR', () => {
        const result = validateIdentifier('192.168.1.0/24', ['192.168.1.0/24'], 0);
        expect(result).toBeUndefined();
    });

    it('returns undefined for valid ClientID', () => {
        const result = validateIdentifier('my-client-01', ['my-client-01'], 0);
        expect(result).toBeUndefined();
    });

    it('returns format error for invalid identifier', () => {
        const result = validateIdentifier('not a valid id!!!', [], 0);
        expect(result).toBeTruthy();
    });

    it('returns duplicate error for repeated identifier', () => {
        const result = validateIdentifier('test-id', ['other-id', 'test-id'], 0);
        expect(result).toBeTruthy();
    });

    it('returns undefined for unique identifiers', () => {
        const result = validateIdentifier('test-id', ['other-id', 'another-id'], 0);
        expect(result).toBeUndefined();
    });
});

describe('validateUpstreams', () => {
    it('returns undefined for an empty value', () => {
        const result = validateUpstreams('');
        expect(result).toBeUndefined();
    });

    it('returns undefined for comment lines', () => {
        const result = validateUpstreams('# this is a comment');
        expect(result).toBeUndefined();
    });

    it('returns undefined for blank lines', () => {
        const result = validateUpstreams('\n\n');
        expect(result).toBeUndefined();
    });

    it('returns undefined for a plain IP address', () => {
        const result = validateUpstreams('1.1.1.1');
        expect(result).toBeUndefined();
    });

    it('returns undefined for an IP with port', () => {
        const result = validateUpstreams('1.1.1.1:53');
        expect(result).toBeUndefined();
    });

    it('returns undefined for a protocol URL', () => {
        const result = validateUpstreams('tls://dns.example.com');
        expect(result).toBeUndefined();
    });

    it('returns undefined for a hostname', () => {
        const result = validateUpstreams('dns.example.com');
        expect(result).toBeUndefined();
    });

    it('returns error for a line without dot or colon', () => {
        const result = validateUpstreams('not-a-valid-upstream');
        expect(result).toBe('Invalid format');
    });

    it('returns error on the correct line number for mixed content', () => {
        const result = validateUpstreams('1.1.1.1\nbadline\ntls://ok.com');
        expect(result).toBe('Invalid format on line 2');
    });

    it('skips comments and only flags real lines', () => {
        const result = validateUpstreams('# comment\nbadline\n1.1.1.1');
        expect(result).toBe('Invalid format on line 2');
    });

    it('returns "Invalid format" for single invalid line with trailing newline', () => {
        const result = validateUpstreams('badline\n');
        expect(result).toBe('Invalid format');
    });

    it('returns "Invalid format" for single invalid line with leading newline', () => {
        const result = validateUpstreams('\nbadline');
        expect(result).toBe('Invalid format');
    });

    it('returns "Invalid format on lines 1, 2" when both invalid', () => {
        const result = validateUpstreams('bad1\nbad2');
        expect(result).toBe('Invalid format on lines 1, 2');
    });

    it('returns "Invalid format on line 2" when second line invalid in multi-content', () => {
        const result = validateUpstreams('1.1.1.1\nbad');
        expect(result).toBe('Invalid format on line 2');
    });

    it('handles blank line between two invalid lines', () => {
        const result = validateUpstreams('bad1\n\nbad2');
        expect(result).toBe('Invalid format on lines 1, 3');
    });

    it('returns "Invalid format" for comment-then-invalid (one content line)', () => {
        const result = validateUpstreams('# comment\nbadline');
        expect(result).toBe('Invalid format');
    });

    it('returns "Invalid format" for invalid-then-comment (one content line)', () => {
        const result = validateUpstreams('badline\n# comment');
        expect(result).toBe('Invalid format');
    });
});

describe('validateRequiredValue', () => {
    it('returns undefined for non-empty string', () => {
        expect(validateRequiredValue('hello')).toBeUndefined();
    });

    it('returns error for empty string', () => {
        expect(validateRequiredValue('')).toBeTruthy();
    });

    it('returns error for whitespace-only string', () => {
        expect(validateRequiredValue('   ')).toBeTruthy();
    });

    it('returns undefined for 0 (zero is valid)', () => {
        expect(validateRequiredValue(0)).toBeUndefined();
    });

    it('returns error for undefined', () => {
        expect(validateRequiredValue(undefined)).toBeTruthy();
    });

    it('returns undefined for truthy number', () => {
        expect(validateRequiredValue(42)).toBeUndefined();
    });

    it('returns error for false', () => {
        expect(validateRequiredValue(false)).toBeTruthy();
    });
});

describe('validateIpv4', () => {
    it('returns undefined for valid IPv4', () => {
        expect(validateIpv4('192.168.1.1')).toBeUndefined();
    });

    it('returns error for invalid IPv4', () => {
        expect(validateIpv4('999.999.999.999')).toBeTruthy();
    });

    it('returns undefined for empty string (skip)', () => {
        expect(validateIpv4('')).toBeUndefined();
    });

    it('returns undefined for undefined (skip)', () => {
        expect(validateIpv4(undefined)).toBeUndefined();
    });
});

describe('validatePort', () => {
    it('returns undefined for port in web range', () => {
        expect(validatePort(8080)).toBeUndefined();
    });

    it('returns undefined for boundary 0 (disabled)', () => {
        expect(validatePort(0)).toBeUndefined();
    });

    it('returns undefined for boundary 1', () => {
        expect(validatePort(1)).toBeUndefined();
    });

    it('returns undefined for boundary 80', () => {
        expect(validatePort(80)).toBeUndefined();
    });

    it('returns undefined for boundary 65535', () => {
        expect(validatePort(65535)).toBeUndefined();
    });

    it('returns error for port above 65535', () => {
        expect(validatePort(65536)).toBeTruthy();
    });

    it('returns error for negative port', () => {
        expect(validatePort(-1)).toBeTruthy();
    });

    it('returns undefined for undefined (no value)', () => {
        expect(validatePort(undefined)).toBeUndefined();
    });
});

describe('validateInstallPort', () => {
    it('returns undefined for valid port', () => {
        expect(validateInstallPort(80)).toBeUndefined();
    });

    it('returns undefined for boundary 1', () => {
        expect(validateInstallPort(1)).toBeUndefined();
    });

    it('returns undefined for boundary 65535', () => {
        expect(validateInstallPort(65535)).toBeUndefined();
    });

    it('returns error for 0 (install requires real port)', () => {
        expect(validateInstallPort(0)).toBeTruthy();
    });

    it('returns error for port above 65535', () => {
        expect(validateInstallPort(65536)).toBeTruthy();
    });
});

describe('validatePlainDns', () => {
    it('returns undefined when both encryption and plain DNS are on', () => {
        expect(validatePlainDns(true, { enabled: true })).toBeUndefined();
    });

    it('returns undefined when plain DNS is on, encryption off', () => {
        expect(validatePlainDns(true, { enabled: false })).toBeUndefined();
    });

    it('returns undefined when encryption is on, plain DNS off', () => {
        expect(validatePlainDns(false, { enabled: true })).toBeUndefined();
    });

    it('returns error when both are off', () => {
        expect(validatePlainDns(false, { enabled: false })).toBeTruthy();
    });
});

describe('validateIpForGatewaySubnetMask', () => {
    const validContext = {
        v4: { gateway_ip: '192.168.1.1', subnet_mask: '255.255.255.0' },
    };

    it('returns undefined for IP in subnet', () => {
        expect(validateIpForGatewaySubnetMask('192.168.1.5', validContext)).toBeUndefined();
    });

    it('returns error for IP outside subnet', () => {
        expect(validateIpForGatewaySubnetMask('10.0.0.1', validContext)).toBeTruthy();
    });

    it('returns undefined when context is missing', () => {
        expect(validateIpForGatewaySubnetMask('192.168.1.5', {})).toBeUndefined();
    });

    it('returns undefined for empty value', () => {
        expect(validateIpForGatewaySubnetMask('', validContext)).toBeUndefined();
    });

    // REGRESSION: guard must use nested allValues.v4.gateway_ip, not flat
    it('validates IP using nested context only (no flat keys)', () => {
        const nestedOnly = {
            v4: { gateway_ip: '192.168.1.1', subnet_mask: '255.255.255.0' },
        };
        // IP outside subnet should return error even without flat keys
        expect(validateIpForGatewaySubnetMask('10.0.0.1', nestedOnly)).toBeTruthy();
    });
});

describe('validateIpv4InCidr', () => {
    it('returns undefined for IP within CIDR', () => {
        expect(validateIpv4InCidr('192.168.1.5', { cidr: '192.168.1.0/24' })).toBeUndefined();
    });

    it('returns error for IP outside CIDR', () => {
        expect(validateIpv4InCidr('10.0.0.1', { cidr: '192.168.1.0/24' })).toBeTruthy();
    });
});

describe('validatePasswordLength', () => {
    it('returns true for too-short password (< 8 UTF-8 bytes)', () => {
        expect(validatePasswordLength('short')).toBe(true);
    });

    it('returns undefined for valid password', () => {
        expect(validatePasswordLength('longenough')).toBeUndefined();
    });

    it('returns undefined for empty string (skipped)', () => {
        expect(validatePasswordLength('')).toBeUndefined();
    });

    it('returns undefined for undefined (skipped)', () => {
        expect(validatePasswordLength(undefined)).toBeUndefined();
    });

    it('returns true for password exceeding 72 UTF-8 bytes', () => {
        expect(validatePasswordLength('a'.repeat(73))).toBe(true);
    });

    it('counts UTF-8 bytes, not characters (multibyte é = 2 bytes)', () => {
        // 'é' repeated 3 times = 6 UTF-8 bytes < 8 → too short
        expect(validatePasswordLength('é'.repeat(3))).toBe(true);
        // 'é' repeated 4 times = 8 UTF-8 bytes → valid
        expect(validatePasswordLength('é'.repeat(4))).toBeUndefined();
    });
});

describe('validateIpNotDuplicate', () => {
    const leases = [{ ip: '192.168.1.10' }, { ip: '192.168.1.20' }];

    it('returns undefined for unique IP', () => {
        expect(validateIpNotDuplicate(leases)('192.168.1.30')).toBeUndefined();
    });

    it('returns error for duplicate IP', () => {
        expect(validateIpNotDuplicate(leases)('192.168.1.10')).toBeTruthy();
    });

    it('ignores the edit IP (editing existing lease)', () => {
        expect(validateIpNotDuplicate(leases, '192.168.1.10')('192.168.1.10')).toBeUndefined();
    });

    it('returns undefined for empty leases', () => {
        expect(validateIpNotDuplicate([])('192.168.1.1')).toBeUndefined();
    });

    it('returns undefined for empty value', () => {
        expect(validateIpNotDuplicate(leases)('')).toBeUndefined();
    });
});

describe('validateIpPerLine', () => {
    it('returns undefined for valid multi-line IPs', () => {
        expect(validateIpPerLine('192.168.1.1\n10.0.0.1')).toBeUndefined();
    });

    it('returns error when a line is invalid', () => {
        expect(validateIpPerLine('192.168.1.1\nnot-an-ip')).toBeTruthy();
    });

    it('returns undefined for empty string', () => {
        expect(validateIpPerLine('')).toBeUndefined();
    });

    it('skips blank lines', () => {
        expect(validateIpPerLine('\n\n')).toBeUndefined();
    });

    it('skips blank lines between valid IPs', () => {
        expect(validateIpPerLine('192.168.1.1\n\n10.0.0.1')).toBeUndefined();
    });
});

describe('numeric range validators', () => {
    it('validateBetween returns error when value is out of range', () => {
        expect(validateBetween(-1, 0, 32)).toBeTruthy();
        expect(validateBetween(33, 0, 32)).toBeTruthy();
        expect(validateBetween(0, 0, 32)).toBeUndefined();
        expect(validateBetween(32, 0, 32)).toBeUndefined();
        expect(validateBetween(16, 0, 32)).toBeUndefined();
    });

    it('validateMinValue returns error when value is below min', () => {
        expect(validateMinValue(0, 1)).toBeTruthy();
        expect(validateMinValue(1, 1)).toBeUndefined();
        expect(validateMinValue(10, 1)).toBeUndefined();
    });
});

describe('validateCacheSize', () => {
    it('returns undefined when cache is disabled', () => {
        expect(validateCacheSize(0, false)).toBeUndefined();
        expect(validateCacheSize(999999999999, false)).toBeUndefined();
    });

    it('returns error for 0 when enabled', () => {
        const result = validateCacheSize(0, true);
        expect(result).toBeTruthy();
    });

    it('returns undefined for a valid size when enabled', () => {
        expect(validateCacheSize(1000, true)).toBeUndefined();
    });

    it('returns error for value exceeding UINT32_MAX', () => {
        const result = validateCacheSize(4294967296, true);
        expect(result).toBeTruthy();
    });

    it('returns undefined at UINT32_MAX boundary', () => {
        expect(validateCacheSize(4294967295, true)).toBeUndefined();
    });
});

describe('validateRewriteNotExists', () => {
    it('returns undefined for a non-existing domain', () => {
        const result = validateRewriteNotExists('new.example.com', [
            { domain: 'existing.example.com' },
        ]);
        expect(result).toBeUndefined();
    });

    it('returns error for a domain that already exists', () => {
        const result = validateRewriteNotExists('example.com', [{ domain: 'example.com' }]);
        expect(result).toBeTruthy();
    });

    it('returns undefined when editing the same rewrite', () => {
        const result = validateRewriteNotExists(
            'example.com',
            [{ domain: 'example.com' }],
            'example.com',
        );
        expect(result).toBeUndefined();
    });

    it('returns error when editing and changing to an existing other domain', () => {
        const result = validateRewriteNotExists(
            'other.example.com',
            [{ domain: 'example.com' }, { domain: 'other.example.com' }],
            'example.com',
        );
        expect(result).toBeTruthy();
    });

    it('returns undefined for an empty domain', () => {
        const result = validateRewriteNotExists('', [{ domain: 'example.com' }]);
        expect(result).toBeUndefined();
    });

    it('case-insensitive duplicate check', () => {
        const result = validateRewriteNotExists('Example.COM', [{ domain: 'example.com' }]);
        expect(result).toBeTruthy();
    });
});

describe('validateRewriteNotSame', () => {
    it('returns undefined when domain and answer differ', () => {
        const result = validateRewriteNotSame('example.com', '192.168.1.1');
        expect(result).toBeUndefined();
    });

    it('returns error when domain equals answer', () => {
        const result = validateRewriteNotSame('example.com', 'example.com');
        expect(result).toBeTruthy();
    });

    it('returns error case-insensitively', () => {
        const result = validateRewriteNotSame('Example.COM', 'example.com');
        expect(result).toBeTruthy();
    });

    it('returns undefined for empty values', () => {
        expect(validateRewriteNotSame('', 'example.com')).toBeUndefined();
        expect(validateRewriteNotSame('example.com', '')).toBeUndefined();
        expect(validateRewriteNotSame('', '')).toBeUndefined();
    });
});

describe('validateUpstreams with bang prefix', () => {
    it('rejects lines starting with ! as invalid upstreams', () => {
        expect(validateUpstreams('! comment line\n8.8.8.8')).toBeTruthy();
    });

    it('rejects only ! lines as invalid', () => {
        expect(validateUpstreams('! first\n! second')).toBeTruthy();
    });

    it('rejects !-prefixed URLs that would otherwise pass address check', () => {
        expect(validateUpstreams('! https://dns10.quad9.net/dns-query')).toBeTruthy();
    });

    it('rejects !-prefixed upstream with dot in comment', () => {
        expect(validateUpstreams('! dns.example.com:53')).toBeTruthy();
    });

    it('still accepts # comments', () => {
        expect(validateUpstreams('# comment\n8.8.8.8')).toBeUndefined();
    });

    it('rejects ! and invalid non-comment lines', () => {
        expect(validateUpstreams('! good\ninvalidline')).toBeTruthy();
    });
});

describe('validateLeaseTime', () => {
    it('accepts 1 (minimum)', () => {
        expect(validateLeaseTime(1)).toBeUndefined();
    });

    it('rejects 0', () => {
        expect(validateLeaseTime(0)).toBeTruthy();
    });

    it('accepts UINT32_MAX (4294967295)', () => {
        expect(validateLeaseTime(4294967295)).toBeUndefined();
    });

    it('rejects UINT32_MAX + 1 (4294967296)', () => {
        expect(validateLeaseTime(4294967296)).toBeTruthy();
    });

    it('rejects empty string', () => {
        expect(validateLeaseTime('')).toBeTruthy();
    });

    it('rejects undefined', () => {
        expect(validateLeaseTime(undefined)).toBeTruthy();
    });

    it('rejects NaN', () => {
        expect(validateLeaseTime(NaN)).toBeTruthy();
    });
});
