import { render, waitFor } from '@solidjs/testing-library';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
    dhcpState: {
        interfaces: {
            eth0: {
                flags: 'up|broadcast',
                gateway_ip: '192.168.1.1',
                hardware_address: '00:11:22:33:44:55',
                ipv4_addresses: ['192.168.1.2'],
                ipv6_addresses: [] as string[],
                name: 'eth0',
            },
        },
        processingConfig: false,
        v4: {
            gateway_ip: '',
            subnet_mask: '',
            range_start: '',
            range_end: '',
            lease_duration: 0,
        },
    },
}));

vi.mock('panel/stores/dhcp', () => ({
    get dhcpState() {
        return mocks.dhcpState;
    },
}));

import { DhcpV4Modal } from 'panel/components/Dhcp/blocks/DhcpV4Modal';

describe('DhcpV4Modal', () => {
    beforeEach(() => {
        mocks.dhcpState.interfaces.eth0.gateway_ip = '192.168.1.1';
        mocks.dhcpState.interfaces.eth0.ipv4_addresses = ['192.168.1.2'];
        mocks.dhcpState.v4.gateway_ip = '';
    });

    const renderModal = () =>
        render(() => (
            <DhcpV4Modal
                open={true}
                selectedInterface={() => 'eth0'}
                onClose={vi.fn()}
                onSave={vi.fn()}
            />
        ));

    it('prefills a blank gateway from the selected interface', async () => {
        const { container } = renderModal();

        const gatewayInput = container.querySelector('#v4_gateway_ip') as HTMLInputElement;

        await waitFor(() => expect(gatewayInput.value).toBe('192.168.1.1'));
    });

    it('uses the interface IPv4 when it has no reported gateway', async () => {
        mocks.dhcpState.interfaces.eth0.gateway_ip = '';

        const { container } = renderModal();
        const gatewayInput = container.querySelector('#v4_gateway_ip') as HTMLInputElement;

        await waitFor(() => expect(gatewayInput.value).toBe('192.168.1.2'));
    });

    it('preserves a saved gateway', async () => {
        mocks.dhcpState.v4.gateway_ip = '192.168.1.254';

        const { container } = renderModal();
        const gatewayInput = container.querySelector('#v4_gateway_ip') as HTMLInputElement;

        await waitFor(() => expect(gatewayInput.value).toBe('192.168.1.254'));
    });

    it('does not invent a gateway without an interface IPv4', async () => {
        mocks.dhcpState.interfaces.eth0.ipv4_addresses = [];

        const { container } = renderModal();
        const gatewayInput = container.querySelector('#v4_gateway_ip') as HTMLInputElement;

        await waitFor(() => expect(gatewayInput.value).toBe(''));
    });
});
