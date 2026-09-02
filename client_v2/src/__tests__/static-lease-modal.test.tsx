import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';

import { StaticLeaseModal } from 'panel/components/Dhcp/LeasesPage/StaticLeaseModal';

const existingLeases = [{ mac: 'AA:BB:CC:DD:EE:FF', ip: '192.168.1.50', hostname: 'router' }];

const renderModal = (overrides: Record<string, unknown> = {}) => {
    const onSubmit = vi.fn();
    const onClose = vi.fn();
    render(() => (
        <StaticLeaseModal
            isOpen
            isEdit={false}
            initialData={undefined}
            processingAdding={false}
            processingUpdating={false}
            staticLeases={existingLeases}
            dhcpConfig={{ gatewayIp: '192.168.1.1', subnetMask: '255.255.255.0' }}
            onSubmit={onSubmit}
            onClose={onClose}
            {...overrides}
        />
    ));
    return { onSubmit, onClose };
};

const getInput = (id: string) => document.getElementById(id) as HTMLInputElement;

const fillLease = (mac: string, ip: string, hostname: string) => {
    fireEvent.change(getInput('static_lease_mac'), { target: { value: mac } });
    fireEvent.change(getInput('static_lease_ip'), { target: { value: ip } });
    fireEvent.change(getInput('static_lease_hostname'), { target: { value: hostname } });
};

describe('StaticLeaseModal', () => {
    beforeEach(() => vi.clearAllMocks());

    it('blocks submit when hostname duplicates an existing static lease', () => {
        const { onSubmit } = renderModal();
        fillLease('00:11:22:33:44:55', '192.168.1.100', 'router');
        fireEvent.blur(getInput('static_lease_hostname'));

        expect(screen.getByText('This hostname is already added')).toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', { name: 'Save' }));
        expect(onSubmit).not.toHaveBeenCalled();
    });

    it('allows submit when editing the lease itself with its own hostname', () => {
        const { onSubmit } = renderModal({
            isEdit: true,
            initialData: { mac: 'AA:BB:CC:DD:EE:FF', ip: '192.168.1.50', hostname: 'router' },
        });
        fireEvent.change(getInput('static_lease_ip'), { target: { value: '192.168.1.60' } });
        fireEvent.click(screen.getByRole('button', { name: 'Save' }));

        expect(onSubmit).toHaveBeenCalledWith({
            mac: 'AA:BB:CC:DD:EE:FF',
            ip: '192.168.1.60',
            hostname: 'router',
        });
    });

    it('shows an error when hostname is longer than 253 chars', () => {
        const { onSubmit } = renderModal();
        fillLease('00:11:22:33:44:55', '192.168.1.100', 'a'.repeat(254));
        fireEvent.blur(getInput('static_lease_hostname'));

        expect(
            screen.getByText('Hostname must not be longer than 253 characters'),
        ).toBeInTheDocument();
        expect(onSubmit).not.toHaveBeenCalled();
    });

    it('catches a duplicate MAC even when the casing differs from the stored one', () => {
        const { onSubmit } = renderModal({
            staticLeases: [{ mac: 'aa:bb:cc:dd:ee:ff', ip: '192.168.1.50', hostname: 'router' }],
        });
        // Backend returns MACs lowercase; the user may type them uppercase.
        fillLease('AA:BB:CC:DD:EE:FF', '192.168.1.100', 'device');
        fireEvent.click(screen.getByRole('button', { name: 'Save' }));

        expect(screen.getByText('This MAC address is already added')).toBeInTheDocument();
        expect(onSubmit).not.toHaveBeenCalled();
    });
});
