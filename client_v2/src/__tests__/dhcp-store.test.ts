import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    checkActiveDhcp: vi.fn(),
    dhcpInterfaces: vi.fn(),
    dhcpStatus: vi.fn(),
    dhcpSetConfig: vi.fn(),
    status: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
        checkActiveDhcp: mocks.checkActiveDhcp,
        dhcpInterfaces: mocks.dhcpInterfaces,
        dhcpStatus: mocks.dhcpStatus,
        status: mocks.status,
        dhcpSetConfig: mocks.dhcpSetConfig,
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));

import {
    dhcpState,
    findActiveDhcp,
    getDhcpInterfaces,
    getDhcpStatus,
    setDhcpConfig,
    toggleDhcp,
} from 'panel/stores/dhcp';

describe('findActiveDhcp', () => {
    beforeEach(() => vi.clearAllMocks());

    it('passes { interface } not a bare string', async () => {
        mocks.checkActiveDhcp.mockResolvedValue({
            v4: { other_server: { found: 'yes' }, static_ip: { static: 'yes' } },
            v6: { other_server: {} },
        });
        await findActiveDhcp('eth0');
        expect(mocks.checkActiveDhcp).toHaveBeenCalledWith({ interface: 'eth0' });
    });

    it('shows dhcp_found error with retry action when another DHCP server detected', async () => {
        mocks.dhcpInterfaces.mockResolvedValue({
            eth0: { ipv4_addresses: ['1.1.1.1'], ipv6_addresses: [] },
        });
        mocks.checkActiveDhcp.mockResolvedValue({
            v4: {
                other_server: { found: 'yes' },
                static_ip: { static: 'yes', ip: 'x' },
            },
            v6: { other_server: {} },
        });
        await getDhcpInterfaces();
        await findActiveDhcp('eth0');
        expect(mocks.addErrorToast).toHaveBeenCalledWith(
            expect.objectContaining({
                action: expect.objectContaining({ text: expect.any(String) }),
            }),
        );
    });

    it('shows dhcp_not_found success toast when clean', async () => {
        mocks.dhcpInterfaces.mockResolvedValue({
            eth0: { ipv4_addresses: ['1.1.1.1'], ipv6_addresses: [] },
        });
        mocks.checkActiveDhcp.mockResolvedValue({
            v4: {
                other_server: { found: 'no' },
                static_ip: { static: 'yes', ip: '1.1.1.1' },
            },
            v6: { other_server: { found: 'no' } },
        });
        await getDhcpInterfaces();
        await findActiveDhcp('eth0');
        expect(mocks.addSuccessToast).toHaveBeenCalled();
    });
});

describe('setDhcpConfig', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.dhcpSetConfig.mockResolvedValue(undefined);
    });

    it('shows dhcp_config_saved toast', async () => {
        await setDhcpConfig({
            v4: {
                gateway_ip: '192.168.1.1',
                subnet_mask: '255.255.255.0',
                range_start: '192.168.1.100',
                range_end: '192.168.1.200',
                lease_duration: 86400,
            },
            interface_name: 'eth0',
        });
        expect(mocks.addSuccessToast).toHaveBeenCalled();
    });

    it('returns success and stores custom options', async () => {
        const options = ['66 text pxe.example.org', '67 text bootx64.efi'];
        const saved = await setDhcpConfig({
            v4: {
                gateway_ip: '192.168.1.1',
                subnet_mask: '255.255.255.0',
                range_start: '192.168.1.100',
                range_end: '192.168.1.200',
                lease_duration: 86400,
                options,
            },
            interface_name: 'eth0',
        });

        expect(saved).toBe(true);
        expect(dhcpState.v4.options).toEqual(options);
    });

    it('returns failure when saving custom options fails', async () => {
        mocks.dhcpSetConfig.mockRejectedValueOnce(new Error('invalid option'));

        const saved = await setDhcpConfig({
            v4: {
                gateway_ip: '192.168.1.1',
                subnet_mask: '255.255.255.0',
                range_start: '192.168.1.100',
                range_end: '192.168.1.200',
                lease_duration: 86400,
                options: ['66 unknown pxe.example.org'],
            },
            interface_name: 'eth0',
        });

        expect(saved).toBe(false);
        expect(mocks.addErrorToast).toHaveBeenCalled();
    });
});

describe('getDhcpStatus', () => {
    beforeEach(() => vi.clearAllMocks());

    it('hydrates custom options returned by the API', async () => {
        mocks.status.mockResolvedValue({ dhcp_available: true });
        mocks.dhcpStatus.mockResolvedValue({
            enabled: false,
            interface_name: 'eth0',
            v4: {
                gateway_ip: '192.168.1.1',
                subnet_mask: '255.255.255.0',
                range_start: '192.168.1.100',
                range_end: '192.168.1.200',
                lease_duration: 86400,
                options: ['6 ips 192.168.1.2,192.168.1.3'],
            },
            v6: { range_start: '', lease_duration: 0 },
            leases: [],
            static_leases: [],
        });

        await getDhcpStatus();

        expect(dhcpState.v4.options).toEqual(['6 ips 192.168.1.2,192.168.1.3']);
    });
});

describe('toggleDhcp', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.dhcpSetConfig.mockResolvedValue(undefined);
    });

    it('computes enabled from passed config, not current state', async () => {
        await toggleDhcp({ enabled: false, interface_name: 'eth0' });
        expect(mocks.dhcpSetConfig).toHaveBeenCalledWith(
            expect.objectContaining({ enabled: true }),
        );
    });
});
