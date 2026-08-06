import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    dnsInfo: vi.fn(),
    dnsConfig: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    dnsInfo: mocks.dnsInfo,
    dnsConfig: mocks.dnsConfig,
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

    it('defaults null blocking IPs and empty upstream_mode', async () => {
        mocks.dnsInfo.mockResolvedValue({
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

    it('toggles resolve_clients and persists', async () => {
        mocks.dnsConfig.mockResolvedValue({});

        const before = dnsConfigState.resolve_clients;
        await toggleResolveClients();
        expect(dnsConfigState.resolve_clients).toBe(!before);

        // Toggle back
        await toggleResolveClients();
        expect(dnsConfigState.resolve_clients).toBe(before);
    });

    it('calls dnsConfig with inverted value', async () => {
        mocks.dnsConfig.mockResolvedValue({});

        const before = dnsConfigState.resolve_clients;
        await toggleResolveClients();
        expect(mocks.dnsConfig).toHaveBeenCalledWith({
            resolve_clients: !before,
        });
    });
});
