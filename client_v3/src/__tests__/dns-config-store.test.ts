import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    getDnsConfig: vi.fn(),
    addErrorToast: vi.fn(),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: { getDnsConfig: mocks.getDnsConfig },
}));
vi.mock('panel/stores/toasts', () => ({ addErrorToast: mocks.addErrorToast }));

import { getDnsConfig, dnsConfigState } from 'panel/stores/dnsConfig';

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
