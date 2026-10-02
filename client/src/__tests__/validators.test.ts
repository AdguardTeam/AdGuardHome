import { beforeEach, describe, expect, test, vi } from 'vitest';

vi.mock('i18next', () => ({
    default: {
        t: (key: string) => key,
    },
}));

import { validateIpForGatewaySubnetMask } from '../helpers/validators';

describe('validateIpForGatewaySubnetMask', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    const nestedForm = {
        v4: {
            gateway_ip: '192.168.1.1',
            subnet_mask: '255.255.255.0',
            range_start: '10.0.0.100',
            range_end: '10.0.0.200',
            lease_duration: 86400,
        },
    };

    test('rejects an out-of-subnet range when gateway fields live under v4', () => {
        expect(validateIpForGatewaySubnetMask('10.0.0.100', nestedForm)).toBe('subnet_error');
    });

    test('accepts an in-subnet range when gateway fields live under v4', () => {
        const inSubnet = {
            v4: {
                ...nestedForm.v4,
                range_start: '192.168.1.100',
                range_end: '192.168.1.200',
            },
        };
        expect(validateIpForGatewaySubnetMask('192.168.1.100', inSubnet)).toBeUndefined();
    });

    test('skips subnet checks when nested gateway fields are missing', () => {
        expect(validateIpForGatewaySubnetMask('10.0.0.100', { v4: {} })).toBeUndefined();
    });
});
