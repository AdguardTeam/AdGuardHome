import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';

const mocks = vi.hoisted(() => ({
    dhcpState: {
        processingConfig: false,
        v4: {
            gateway_ip: '',
            subnet_mask: '',
            range_start: '',
            range_end: '',
            lease_duration: 0,
        },
        interfaces: {
            eth0: {
                ipv4_addresses: ['192.168.1.1'],
                ipv6_addresses: ['fe80::1'],
                gateway_ip: '192.168.1.1',
            },
        },
    } as Record<string, unknown>,
}));

vi.mock('panel/stores/dhcp', () => ({
    get dhcpState() {
        return mocks.dhcpState;
    },
}));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

import { DhcpV4Modal } from 'panel/components/Dhcp/blocks/DhcpV4Modal/DhcpV4Modal';

const field = (id: string) => document.getElementById(id) as HTMLInputElement;
const saveButton = () => screen.getByTestId('config-dialog-save');

const renderModal = () => {
    const onSave = vi.fn();
    render(() => (
        <DhcpV4Modal
            open
            selectedInterface={() => 'eth0'}
            onClose={() => {}}
            onSave={onSave}
        />
    ));
    return onSave;
};

beforeEach(() => vi.clearAllMocks());

describe('DhcpV4Modal — dual-stack interface', () => {
    // REGRESSION: AGH-171 — saving DHCPv4 on an interface that also has an
    // IPv6 address must not depend on any DHCPv6 value.  The payload carries
    // the v4 section only, so no v6 field can block or leak into the save.
    it('saves the IPv4 section only, leaving DHCPv6 unconfigured', async () => {
        const user = userEvent.setup();
        const onSave = renderModal();

        await user.type(field('v4_gateway_ip'), '192.168.1.1');
        await user.type(field('v4_subnet_mask'), '255.255.255.0');
        await user.type(field('v4_range_start'), '192.168.1.100');
        await user.type(field('v4_range_end'), '192.168.1.200');
        await user.type(field('v4_lease_duration'), '86400');
        await user.click(saveButton());

        expect(onSave).toHaveBeenCalledWith({
            gateway_ip: '192.168.1.1',
            subnet_mask: '255.255.255.0',
            range_start: '192.168.1.100',
            range_end: '192.168.1.200',
            lease_duration: 86400,
        });
    });
});
