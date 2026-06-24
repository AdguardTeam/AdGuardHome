import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    setTlsConfig: vi.fn(),
    validateTlsConfig: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: {
        setTlsConfig: mocks.setTlsConfig,
        validateTlsConfig: mocks.validateTlsConfig,
        getGlobalStatus: vi.fn(),
    },
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));
vi.mock('panel/stores/dashboard', () => ({ getDnsStatus: vi.fn() }));

import {
    setTlsConfig,
    validateTlsConfig,
    resetValidationStatus,
    encryptionState,
} from 'panel/stores/encryption';

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

    it('clears validation status fields when resetValidationStatus is called', async () => {
        mocks.validateTlsConfig.mockResolvedValue({
            valid_chain: true,
            valid_cert: true,
            valid_key: true,
            valid_pair: true,
            subject: 'CN=example.com',
            issuer: 'CN=example.com',
            key_type: 'RSA',
            not_after: '2027-01-01T00:00:00Z',
            not_before: '2026-01-01T00:00:00Z',
            dns_names: ['example.com'],
            warning_validation: 'self-signed certificate',
            certificate_chain: '',
            private_key: '',
        });

        await validateTlsConfig({
            enabled: true,
            certificate_chain: 'x',
            private_key: 'y',
        });

        expect(encryptionState.valid_cert).toBe(true);
        expect(encryptionState.warning_validation).toBe('self-signed certificate');

        resetValidationStatus();

        expect(encryptionState.valid_cert).toBe(false);
        expect(encryptionState.valid_chain).toBe(false);
        expect(encryptionState.valid_key).toBe(false);
        expect(encryptionState.valid_pair).toBe(false);
        expect(encryptionState.warning_validation).toBe('');
        expect(encryptionState.subject).toBe('');
        expect(encryptionState.issuer).toBe('');
        expect(encryptionState.key_type).toBe('');
        expect(encryptionState.dns_names).toBeNull();
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
