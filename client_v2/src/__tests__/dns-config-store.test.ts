import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    getDnsConfig: vi.fn(),
    setDnsConfig: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: {
        getDnsConfig: mocks.getDnsConfig,
        setDnsConfig: mocks.setDnsConfig,
    },
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));
vi.mock('panel/common/intl', () => ({
    default: { getMessage: vi.fn((key: string) => key) },
}));

import { getDnsConfig, dnsConfigState, toggleResolveClients } from 'panel/stores/dnsConfig';

describe('getDnsConfig', () => {
    beforeEach(() => vi.clearAllMocks());

    it('defaults null blocking IPs and empty upstream_mode (FR-012)', async () => {
        mocks.getDnsConfig.mockResolvedValue({
            blocking_ipv4: null,
            blocking_ipv6: null,
            upstream_mode: '',
        });
        await getDnsConfig();
        expect(dnsConfigState.blocking_ipv4).toBe('0.0.0.0');
        expect(dnsConfigState.blocking_ipv6).toBe('::');
        expect(dnsConfigState.upstream_mode).toBe('load_balance');
    });
});

describe('toggleResolveClients', () => {
    beforeEach(() => vi.clearAllMocks());

    it('toggles resolve_clients and persists (FR-032)', async () => {
        mocks.setDnsConfig.mockResolvedValue({});

        const before = dnsConfigState.resolve_clients;
        await toggleResolveClients();
        expect(dnsConfigState.resolve_clients).toBe(!before);

        // Toggle back
        await toggleResolveClients();
        expect(dnsConfigState.resolve_clients).toBe(before);
    });

    it('calls apiClient.setDnsConfig with inverted value (FR-032)', async () => {
        mocks.setDnsConfig.mockResolvedValue({});

        const before = dnsConfigState.resolve_clients;
        await toggleResolveClients();
        expect(mocks.setDnsConfig).toHaveBeenCalledWith({
            resolve_clients: !before,
        });
    });
});
