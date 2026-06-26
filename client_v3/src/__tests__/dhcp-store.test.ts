import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    findActiveDhcp: vi.fn(),
    getDhcpInterfaces: vi.fn(),
    getDhcpStatus: vi.fn(),
    setDhcpConfig: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: {
        findActiveDhcp: mocks.findActiveDhcp,
        getDhcpInterfaces: mocks.getDhcpInterfaces,
        getDhcpStatus: mocks.getDhcpStatus,
        getGlobalStatus: vi.fn(),
        setDhcpConfig: mocks.setDhcpConfig,
    },
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));

import { findActiveDhcp, getDhcpInterfaces, setDhcpConfig, toggleDhcp } from 'panel/stores/dhcp';

describe('findActiveDhcp', () => {
    beforeEach(() => vi.clearAllMocks());

    it('passes { interface } not a bare string', async () => {
        mocks.findActiveDhcp.mockResolvedValue({
            v4: { other_server: { found: 'yes' }, static_ip: { static: 'yes' } },
            v6: { other_server: {} },
        });
        await findActiveDhcp('eth0');
        expect(mocks.findActiveDhcp).toHaveBeenCalledWith({ interface: 'eth0' });
    });

    it('shows dhcp_found error with retry action when another DHCP server detected', async () => {
        mocks.getDhcpInterfaces.mockResolvedValue({
            eth0: { ipv4_addresses: ['1.1.1.1'], ipv6_addresses: [] },
        });
        mocks.findActiveDhcp.mockResolvedValue({
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
        mocks.getDhcpInterfaces.mockResolvedValue({
            eth0: { ipv4_addresses: ['1.1.1.1'], ipv6_addresses: [] },
        });
        mocks.findActiveDhcp.mockResolvedValue({
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
        mocks.setDhcpConfig.mockResolvedValue(undefined);
    });

    it('shows dhcp_config_saved toast (FR-010)', async () => {
        await setDhcpConfig({
            v4: { range_start: 'a', range_end: 'b' },
            interface_name: 'eth0',
        });
        expect(mocks.addSuccessToast).toHaveBeenCalled();
    });
});

describe('toggleDhcp', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.setDhcpConfig.mockResolvedValue(undefined);
    });

    it('computes enabled from passed config, not current state (FR-011)', async () => {
        await toggleDhcp({ enabled: false, interface_name: 'eth0' });
        expect(mocks.setDhcpConfig).toHaveBeenCalledWith(
            expect.objectContaining({ enabled: true }),
        );
    });
});
