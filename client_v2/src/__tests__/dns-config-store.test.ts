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

import {
    getDnsConfig,
    dnsConfigState,
    togglePrivatePtrResolvers,
    toggleResolveClients,
} from 'panel/stores/dnsConfig';

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

describe('togglePrivatePtrResolvers', () => {
    beforeEach(() => vi.clearAllMocks());

    const loadConfig = async (config: {
        use_private_ptr_resolvers: boolean;
        local_ptr_upstreams: string[];
    }) => {
        mocks.dnsInfo.mockResolvedValue(config);
        await getDnsConfig();
    };

    it('sends the effective servers when enabling', async () => {
        mocks.dnsConfig.mockResolvedValue({});
        await loadConfig({
            use_private_ptr_resolvers: false,
            local_ptr_upstreams: ['1.1.1.1', '# comment'],
        });

        await togglePrivatePtrResolvers();

        // The backend validates the request body on its own, so the servers
        // have to travel with the flag.
        expect(mocks.dnsConfig).toHaveBeenCalledWith({
            use_private_ptr_resolvers: true,
            local_ptr_upstreams: ['1.1.1.1', '# comment'],
        });
    });

    it('sends only the flag when no servers are configured', async () => {
        mocks.dnsConfig.mockResolvedValue({});
        await loadConfig({ use_private_ptr_resolvers: false, local_ptr_upstreams: [] });

        await togglePrivatePtrResolvers();

        expect(mocks.dnsConfig).toHaveBeenCalledWith({ use_private_ptr_resolvers: true });
    });

    it('does not send the servers when disabling', async () => {
        mocks.dnsConfig.mockResolvedValue({});
        await loadConfig({
            use_private_ptr_resolvers: true,
            local_ptr_upstreams: ['1.1.1.1'],
        });

        await togglePrivatePtrResolvers();

        expect(mocks.dnsConfig).toHaveBeenCalledWith({ use_private_ptr_resolvers: false });
    });

    it('drops a toggle while another save is in flight', async () => {
        // Never settles, so `processingSetConfig` stays true.
        mocks.dnsConfig.mockReturnValue(new Promise(() => {}));
        await loadConfig({ use_private_ptr_resolvers: false, local_ptr_upstreams: [] });

        togglePrivatePtrResolvers();
        await togglePrivatePtrResolvers();

        expect(mocks.dnsConfig).toHaveBeenCalledTimes(1);
    });
});
