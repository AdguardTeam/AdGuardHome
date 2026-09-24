import { describe, expect, test, vi } from 'vitest';

import {
    validateClientId,
    validateIpForGatewaySubnetMask,
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
});

describe('validateRequiredIfSectionFilled', () => {
    const validateV4 = validateRequiredIfSectionFilled('v4');

    test('does not require a value while the section is empty', () => {
        expect(
            validateV4('', { v4: { range_start: '', range_end: '', lease_duration: undefined } }),
        ).toBeUndefined();
        expect(validateV4(undefined, {})).toBeUndefined();
        expect(validateV4(undefined, undefined)).toBeUndefined();
    });

    test('requires a value as soon as the section has any value', () => {
        expect(validateV4('', { v4: { gateway_ip: '', lease_duration: 86400 } })).toBe(
            'form_error_required',
        );
        expect(validateV4('  ', { v4: { gateway_ip: '', lease_duration: 86400 } })).toBe(
            'form_error_required',
        );
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
        expect(validateLease('  ', { v6: { range_start: 'fe80::1' } })).toBe(
            'form_error_required',
        );
        expect(validateLease('86400', { v6: { range_start: 'fe80::1' } })).toBeUndefined();
    });

    test('only looks at its own section and field', () => {
        expect(validateLease(undefined, { v4: { gateway_ip: '192.168.1.1' } })).toBeUndefined();
    });
});
