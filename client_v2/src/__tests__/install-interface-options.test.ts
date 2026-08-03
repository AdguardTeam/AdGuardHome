import { describe, expect, it } from 'vitest';

import { buildInterfaceOptions } from 'panel/install/Setup/helpers/InterfaceOptions';
import type { InstallInterface } from 'panel/initialState';

describe('buildInterfaceOptions', () => {
    it('includes every address of an active interface', () => {
        const interfaces: InstallInterface[] = [
            {
                flags: 'up|broadcast|multicast',
                hardware_address: '00:11:22:33:44:55',
                ip_addresses: ['192.0.2.1', '2001:db8::1'],
                mtu: 1500,
                name: 'eth0',
            },
        ];

        expect(buildInterfaceOptions(interfaces).slice(1)).toStrictEqual([
            {
                value: '192.0.2.1',
                label: 'eth0 – 192.0.2.1',
            },
            {
                value: '2001:db8::1',
                label: 'eth0 – 2001:db8::1',
            },
        ]);
    });
});
