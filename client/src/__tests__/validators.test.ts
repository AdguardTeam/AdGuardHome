import { describe, expect, test, vi } from 'vitest';

import {
    getDefaultV4Values,
    getDefaultV6Values,
    isSectionFilled,
    omitEmptySections,
    validateClientId,
    validateGatewaySubnetMask,
    validateIp,
    validateIpForGatewaySubnetMask,
    validateIpv4,
    validateIpv6,
    validateMac,
    validateRequiredIfFilled,
    validateRequiredIfSectionFilled,
} from '../helpers/validators';

vi.mock('i18next', () => ({
    default: {
        t: (key: string) => key,
    },
}));

describe('validateIpForGatewaySubnetMask', () => {
    test('rejects an IPv4 range address outside the configured subnet', () => {
        const formValues = {
            v4: {
                gateway_ip: '192.168.1.1',
                subnet_mask: '255.255.255.0',
                range_start: '10.0.0.100',
                range_end: '10.0.0.200',
                lease_duration: 86400,
            },
        };

        expect(validateIpForGatewaySubnetMask(formValues.v4.range_start, formValues)).toBe('subnet_error');
        expect(validateIpForGatewaySubnetMask(formValues.v4.range_end, formValues)).toBe('subnet_error');
    });

    test('accepts an IPv4 range address inside the configured subnet', () => {
        const formValues = {
            v4: {
                gateway_ip: '192.168.1.1',
                subnet_mask: '255.255.255.0',
                range_start: '192.168.1.100',
                range_end: '192.168.1.200',
            },
        };

        expect(validateIpForGatewaySubnetMask(formValues.v4.range_start, formValues)).toBeUndefined();
        expect(validateIpForGatewaySubnetMask(formValues.v4.range_end, formValues)).toBeUndefined();
    });

    test('skips validation when the gateway or subnet mask is missing', () => {
        expect(validateIpForGatewaySubnetMask('10.0.0.1', { v4: {} })).toBeUndefined();
        expect(validateIpForGatewaySubnetMask('10.0.0.1', undefined)).toBeUndefined();
    });

    test('skips validation when the subnet mask is not a valid mask', () => {
        const formValues = { v4: { gateway_ip: '192.168.1.1', subnet_mask: '1.2.3.4' } };

        expect(validateIpForGatewaySubnetMask('10.0.0.1', formValues)).toBeUndefined();
    });
});

describe('validateClientId', () => {
    test('accepts IPv6 CIDRs with an embedded dotted-decimal IPv4 address', () => {
        expect(validateClientId('::ffff:192.168.1.0/120')).toBeUndefined();
        expect(validateClientId('::ffff:10.0.0.0/104')).toBeUndefined();
    });

    test('accepts the hexadecimal form of the same range', () => {
        expect(validateClientId('::ffff:c0a8:100/120')).toBeUndefined();
    });

    test('accepts plain addresses, CIDRs, MACs and client IDs', () => {
        expect(validateClientId('192.168.1.1')).toBeUndefined();
        expect(validateClientId('2001:db8::/64')).toBeUndefined();
        expect(validateClientId('192.168.1.0/24')).toBeUndefined();
        expect(validateClientId('aa:bb:cc:dd:ee:ff')).toBeUndefined();
        expect(validateClientId('my-client-id')).toBeUndefined();
    });

    test('rejects invalid octets and prefix lengths', () => {
        expect(validateClientId('::ffff:192.168.1.256/120')).toBe('form_error_client_id_format');
        expect(validateClientId('192.168.1.256/24')).toBe('form_error_client_id_format');
        expect(validateClientId('2001:db8::/129')).toBe('form_error_client_id_format');
    });

    test('rejects malformed identifiers', () => {
        expect(validateClientId('not a client id')).toBe('form_error_client_id_format');
    });

    // REGRESSION: netip.ParseAddr rejects IPv4 addresses with leading-zero
    // octets, so the backend answers 400 for identifiers the form accepted.
    test('rejects non-canonical IPv4 addresses with leading zeros', () => {
        expect(validateClientId('192.168.01.1')).toBe('form_error_client_id_format');
        expect(validateClientId('010.0.0.1')).toBe('form_error_client_id_format');
        expect(validateClientId('192.168.001.1')).toBe('form_error_client_id_format');
    });

    test('rejects an IPv6 address with a non-canonical embedded IPv4 tail', () => {
        expect(validateClientId('::ffff:192.168.01.1')).toBe('form_error_client_id_format');
        expect(validateClientId('::ffff:192.168.1.1')).toBeUndefined();
    });

    // REGRESSION: netutil.ValidateHostnameLabel requires alphanumeric first
    // and last runes, so leading/trailing hyphens are rejected by the server.
    test('rejects client IDs with leading or trailing hyphens', () => {
        expect(validateClientId('-abc')).toBe('form_error_client_id_format');
        expect(validateClientId('abc-')).toBe('form_error_client_id_format');
        expect(validateClientId('-')).toBe('form_error_client_id_format');
        expect(validateClientId('---')).toBe('form_error_client_id_format');
        expect(validateClientId('a-b')).toBeUndefined();
        expect(validateClientId('a--b')).toBeUndefined();
    });

    // REGRESSION: net.ParseMAC requires exactly two hex digits per field and a
    // uniform separator; the old R_MAC accepted over-long fields and mixed
    // separators, which the server answers with 400.
    test('rejects MAC addresses the server would refuse', () => {
        expect(validateClientId('aaaa:bb:cc:dd:ee:ff')).toBe('form_error_client_id_format');
        expect(validateClientId('aa:bbb:cc:dd:ee:ff')).toBe('form_error_client_id_format');
        expect(validateClientId('aa:bb-cc:dd:ee:ff')).toBe('form_error_client_id_format');
        expect(validateClientId('aaaaa.bbbbb.cccc')).toBe('form_error_client_id_format');
        expect(validateClientId('aa:bb:cc:dd:ee:ff')).toBeUndefined();
        expect(validateClientId('aa-bb-cc-dd-ee-ff')).toBeUndefined();
        expect(validateClientId('aabb.ccdd.eeff')).toBeUndefined();
    });

    // REGRESSION: netip.ParseAddr rejects a zone ID on fe80: without at least
    // one group after the colon.
    test('rejects an fe80 address with no groups before the zone ID', () => {
        expect(validateClientId('fe80:%eth0')).toBe('form_error_client_id_format');
        expect(validateClientId('fe80::%eth0')).toBeUndefined();
        expect(validateClientId('fe80::1%eth0')).toBeUndefined();
    });

    // REGRESSION: netip.ParseAddr treats everything after the first `%` as the
    // zone ID without restricting its contents, so the server accepts this
    // value as an address with a zone ID, not as a CIDR.
    test('accepts IPv6 addresses with a zone ID the server parses as addresses', () => {
        expect(validateClientId('fe80::1%eth0/64')).toBeUndefined();
        expect(validateClientId('2001:db8::1%eth0')).toBeUndefined();
        expect(validateClientId('fe80::1%')).toBe('form_error_client_id_format');
    });
});

describe('validateIpv4', () => {
    test('accepts canonical IPv4 addresses', () => {
        expect(validateIpv4('0.0.0.0')).toBeUndefined();
        expect(validateIpv4('192.168.1.1')).toBeUndefined();
        expect(validateIpv4('255.255.255.255')).toBeUndefined();
    });

    // REGRESSION: the same netip.ParseAddr contract as validateClientId.
    test('rejects non-canonical IPv4 addresses with leading zeros', () => {
        expect(validateIpv4('192.168.01.1')).toBe('form_error_ip4_format');
        expect(validateIpv4('010.0.0.1')).toBe('form_error_ip4_format');
        expect(validateIpv4('192.168.1.01')).toBe('form_error_ip4_format');
    });
});

describe('validateIp', () => {
    test('accepts canonical IPv4 and IPv6 addresses', () => {
        expect(validateIp('192.168.1.1')).toBeUndefined();
        expect(validateIp('::1')).toBeUndefined();
    });

    test('rejects non-canonical IPv4 addresses with leading zeros', () => {
        expect(validateIp('192.168.01.1')).toBe('form_error_ip_format');
        expect(validateIp('010.0.0.1')).toBe('form_error_ip_format');
    });
});

describe('validateIpv6', () => {
    test('accepts IPv6 addresses, including zone IDs and embedded canonical IPv4', () => {
        expect(validateIpv6('2001:db8::1')).toBeUndefined();
        expect(validateIpv6('fe80::1%eth0')).toBeUndefined();
        expect(validateIpv6('::ffff:192.168.1.1')).toBeUndefined();
    });

    test('rejects an IPv6 address with a non-canonical embedded IPv4 tail', () => {
        expect(validateIpv6('::ffff:192.168.01.1')).toBe('form_error_ip6_format');
    });

    test('rejects an fe80 address with no groups before the zone ID', () => {
        expect(validateIpv6('fe80:%eth0')).toBe('form_error_ip6_format');
    });
});

describe('validateMac', () => {
    test('accepts the formats the server parses', () => {
        expect(validateMac('aa:bb:cc:dd:ee:ff')).toBeUndefined();
        expect(validateMac('AA:BB:CC:DD:EE:FF')).toBeUndefined();
        expect(validateMac('aa-bb-cc-dd-ee-ff')).toBeUndefined();
        expect(validateMac('aabb.ccdd.eeff')).toBeUndefined();
        expect(validateMac('00:00:5e:00:53:01:02:03')).toBeUndefined();
    });

    test('rejects over-long fields and mixed separators', () => {
        expect(validateMac('aaaa:bb:cc:dd:ee:ff')).toBe('form_error_mac_format');
        expect(validateMac('aa:bb-cc:dd:ee:ff')).toBe('form_error_mac_format');
        expect(validateMac('aaaaa.bbbbb.cccc')).toBe('form_error_mac_format');
    });
});

describe('isSectionFilled', () => {
    test('is false for a missing or untouched section', () => {
        expect(isSectionFilled(undefined)).toBe(false);
        expect(isSectionFilled({})).toBe(false);
        expect(isSectionFilled({ gateway_ip: '', subnet_mask: '', lease_duration: undefined })).toBe(false);
    });

    test('is true as soon as one value is set', () => {
        expect(isSectionFilled({ range_start: 'fe80::1' })).toBe(true);
        expect(isSectionFilled({ gateway_ip: '192.168.1.1' })).toBe(true);
    });

    // The backend reports the default lease duration of a DHCPv6 section that
    // has never been configured, so raw server values must be normalized with
    // `getDefaultV6Values` before the section is checked.
    test('reports a raw unconfigured DHCPv6 section as filled in', () => {
        const rawV6 = { range_start: '', lease_duration: 86400 };

        expect(isSectionFilled(rawV6)).toBe(true);
        expect(isSectionFilled(getDefaultV6Values(rawV6))).toBe(false);
    });
});

describe('getDefaultV4Values', () => {
    test('clears the lease duration of an untouched section', () => {
        expect(
            getDefaultV4Values({
                gateway_ip: '',
                subnet_mask: '',
                range_start: '',
                range_end: '',
                lease_duration: 86400,
            }),
        ).toEqual({
            gateway_ip: '',
            subnet_mask: '',
            range_start: '',
            range_end: '',
            lease_duration: undefined,
        });
    });

    test('keeps a configured section intact', () => {
        const v4 = { gateway_ip: '192.168.1.1', subnet_mask: '255.255.255.0', lease_duration: 86400 };

        expect(getDefaultV4Values(v4)).toBe(v4);
    });
});

describe('getDefaultV6Values', () => {
    // REGRESSION: the backend keeps the default lease duration of an
    // unconfigured DHCPv6 section, which used to make the section look filled
    // in, so it was sent to the backend and wiped the stored configuration.
    test('clears the lease duration of an untouched section', () => {
        const rawV6 = { range_start: '', lease_duration: 86400 };
        const normalizedV6 = { range_start: '', lease_duration: undefined };

        expect(getDefaultV6Values(rawV6)).toEqual(normalizedV6);
        expect(omitEmptySections({ v4: {}, v6: getDefaultV6Values(rawV6) })).toEqual({});
    });

    test('keeps a configured section intact', () => {
        const v6 = { range_start: 'fe80::1', lease_duration: 86400 };

        expect(getDefaultV6Values(v6)).toBe(v6);
        expect(omitEmptySections({ v4: {}, v6: getDefaultV6Values(v6) })).toEqual({ v6 });
    });
});

describe('validateGatewaySubnetMask', () => {
    // REGRESSION: AGH-171, mirror direction — both DHCP cards share one form
    // instance, so an untouched DHCPv4 section is validated when DHCPv6 is
    // saved and must not report an error there.
    test('accepts an untouched DHCPv4 section', () => {
        expect(validateGatewaySubnetMask(undefined, { v4: {} })).toBeUndefined();
        expect(
            validateGatewaySubnetMask(undefined, {
                v4: {
                    gateway_ip: '',
                    subnet_mask: '',
                    range_start: '',
                    range_end: '',
                    lease_duration: undefined,
                },
            }),
        ).toBeUndefined();
        expect(validateGatewaySubnetMask(undefined, undefined)).toBeUndefined();
    });

    test('rejects a partially filled section', () => {
        expect(validateGatewaySubnetMask(undefined, { v4: { gateway_ip: '192.168.1.1' } })).toBe(
            'gateway_or_subnet_invalid',
        );
        expect(validateGatewaySubnetMask(undefined, { v4: { subnet_mask: '255.255.255.0' } })).toBe(
            'gateway_or_subnet_invalid',
        );
    });

    test('accepts a valid gateway and subnet mask', () => {
        expect(
            validateGatewaySubnetMask(undefined, {
                v4: { gateway_ip: '192.168.1.1', subnet_mask: '255.255.255.0' },
            }),
        ).toBeUndefined();
    });

    test('rejects an invalid gateway or subnet mask', () => {
        expect(
            validateGatewaySubnetMask(undefined, {
                v4: { gateway_ip: 'not-an-ip', subnet_mask: '255.255.255.0' },
            }),
        ).toBe('gateway_or_subnet_invalid');
        expect(
            validateGatewaySubnetMask(undefined, {
                v4: { gateway_ip: '192.168.1.1', subnet_mask: '1.2.3.4' },
            }),
        ).toBe('gateway_or_subnet_invalid');
    });
});

describe('omitEmptySections', () => {
    const v4 = { gateway_ip: '192.168.1.1', subnet_mask: '255.255.255.0' };
    const v6 = { range_start: 'fe80::1', lease_duration: 86400 };

    test('keeps every filled section', () => {
        expect(omitEmptySections({ v4, v6 })).toEqual({ v4, v6 });
    });

    test('leaves out a section the user has not filled in', () => {
        expect(omitEmptySections({ v4: {}, v6 })).toEqual({ v6 });
        expect(
            omitEmptySections({
                v4: {
                    gateway_ip: '',
                    subnet_mask: '',
                    range_start: '',
                    range_end: '',
                    lease_duration: 0,
                },
                v6,
            }),
        ).toEqual({ v6 });
    });

    test('keeps only the filled section', () => {
        expect(omitEmptySections({ v4, v6: {} })).toEqual({ v4 });
    });

    test('returns no sections when nothing is filled in', () => {
        expect(omitEmptySections({ v4: {}, v6: {} })).toEqual({});
        expect(omitEmptySections({})).toEqual({});
    });
});

describe('validateRequiredIfSectionFilled', () => {
    const validateV4 = validateRequiredIfSectionFilled('v4');

    test('does not require a value while the section is empty', () => {
        expect(validateV4('', { v4: { range_start: '', range_end: '', lease_duration: undefined } })).toBeUndefined();
        expect(validateV4(undefined, {})).toBeUndefined();
        expect(validateV4(undefined, undefined)).toBeUndefined();
    });

    test('requires a value as soon as the section has any value', () => {
        expect(validateV4('', { v4: { gateway_ip: '', lease_duration: 86400 } })).toBe('form_error_required');
        expect(validateV4('  ', { v4: { gateway_ip: '', lease_duration: 86400 } })).toBe('form_error_required');
        expect(validateV4('192.168.1.1', { v4: { gateway_ip: '192.168.1.1' } })).toBeUndefined();
    });

    test('only looks at its own section', () => {
        expect(validateV4(undefined, { v6: { range_start: 'fe80::1' } })).toBeUndefined();
    });
});

describe('validateRequiredIfFilled', () => {
    const validateLease = validateRequiredIfFilled('v6', 'range_start');

    // REGRESSION: AGH-171 — DHCPv6 counts as configured only when its range
    // start is set.  The lease duration is pre-filled with a default, so it
    // must not require a range on its own and block saving DHCPv4.
    test('does not require a value when the v6 range start is empty', () => {
        expect(validateLease('', { v6: { range_start: '', lease_duration: 86400 } })).toBeUndefined();
        expect(validateLease(undefined, { v6: {} })).toBeUndefined();
        expect(validateLease(undefined, undefined)).toBeUndefined();
    });

    test('requires a value once the v6 range start is filled', () => {
        expect(validateLease('', { v6: { range_start: 'fe80::1' } })).toBe('form_error_required');
        expect(validateLease('  ', { v6: { range_start: 'fe80::1' } })).toBe('form_error_required');
        expect(validateLease('86400', { v6: { range_start: 'fe80::1' } })).toBeUndefined();
    });

    test('only looks at its own section and field', () => {
        expect(validateLease(undefined, { v4: { gateway_ip: '192.168.1.1' } })).toBeUndefined();
    });
});
