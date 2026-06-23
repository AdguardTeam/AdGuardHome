import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    setTlsConfig: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: { setTlsConfig: mocks.setTlsConfig, getGlobalStatus: vi.fn() },
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));
vi.mock('panel/stores/dashboard', () => ({ getDnsStatus: vi.fn() }));

import { setTlsConfig } from 'panel/stores/encryption';

describe('setTlsConfig', () => {
    beforeEach(() => vi.clearAllMocks());

    it('defaults empty ports to 0 (FR-013)', async () => {
        mocks.setTlsConfig.mockImplementation(async (v: any) => ({
            ...v,
            certificate_chain: '',
            private_key: '',
        }));
        await setTlsConfig({
            certificate_chain: '',
            private_key: '',
            port_https: '',
            port_dns_over_tls: '',
            port_dns_over_quic: '',
        });
        const sent = mocks.setTlsConfig.mock.calls[0][0];
        expect(sent.port_https).toBe(0);
        expect(sent.port_dns_over_tls).toBe(0);
        expect(sent.port_dns_over_quic).toBe(0);
    });

    it('reloads when enabled+force_https on http: origin (FR-014)', async () => {
        const reloadFn = vi.fn();
        Object.defineProperty(window, 'location', {
            value: { protocol: 'http:', reload: reloadFn },
            writable: true,
        });
        mocks.setTlsConfig.mockImplementation(async (v: any) => ({
            ...v,
            certificate_chain: '',
            private_key: '',
        }));
        await setTlsConfig({
            certificate_chain: '',
            private_key: '',
            enabled: true,
            force_https: true,
            port_https: 443,
        });
        expect(reloadFn).toHaveBeenCalled();
    });
});
