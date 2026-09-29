import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    checkActiveDhcp: vi.fn(),
    dhcpInterfaces: vi.fn(),
    dhcpStatus: vi.fn(),
    status: vi.fn(),
    dhcpSetConfig: vi.fn(),
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
                noIcon: true,
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

describe('getDhcpStatus — statusInitialized flag', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        dhcpState.statusInitialized = false;
    });

    it('marks statusInitialized when DHCP is available', async () => {
        mocks.status.mockResolvedValue({ dhcp_available: true });
        mocks.dhcpStatus.mockResolvedValue({ leases: [], static_leases: [] });

        await getDhcpStatus();

        expect(dhcpState.statusInitialized).toBe(true);
    });

    it('marks statusInitialized when DHCP is unavailable', async () => {
        mocks.status.mockResolvedValue({ dhcp_available: false });

        await getDhcpStatus();

        expect(dhcpState.statusInitialized).toBe(true);
    });

    it('marks statusInitialized after a failed request so the loader cannot get stuck', async () => {
        mocks.status.mockRejectedValue(new Error('network'));

        await getDhcpStatus();

        expect(dhcpState.statusInitialized).toBe(true);
        expect(mocks.addErrorToast).toHaveBeenCalled();
    });
});

describe('getDhcpInterfaces — interfacesInitialized flag', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        dhcpState.interfacesInitialized = false;
    });

    it('marks interfacesInitialized after a successful request', async () => {
        mocks.dhcpInterfaces.mockResolvedValue({});

        await getDhcpInterfaces();

        expect(dhcpState.interfacesInitialized).toBe(true);
    });

    it('marks interfacesInitialized after a failed request so the loader cannot get stuck', async () => {
        mocks.dhcpInterfaces.mockRejectedValue(new Error('network'));

        await getDhcpInterfaces();

        expect(dhcpState.interfacesInitialized).toBe(true);
        expect(mocks.addErrorToast).toHaveBeenCalled();
    });
});
