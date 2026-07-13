import { describe, it, expect } from 'vitest';
import { getSubmitValues, defaultTlsValues } from 'panel/components/Encryption/blocks/helpers';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';

describe('getSubmitValues', () => {
    it('empties certificate_chain when source is PATH', () => {
        const result = getSubmitValues({
            enabled: true,
            certificate_chain: 'cert',
            certificate_path: '/path/cert.pem',
            certificate_source: ENCRYPTION_SOURCE.PATH,
            key_source: ENCRYPTION_SOURCE.CONTENT,
            private_key: 'key',
            private_key_saved: false,
        } as any);
        expect(result.certificate_chain).toBe('');
        expect(result.certificate_path).toBe('/path/cert.pem');
    });

    it('empties certificate_path when source is CONTENT', () => {
        const result = getSubmitValues({
            enabled: true,
            certificate_chain: 'cert',
            certificate_path: '/path/cert.pem',
            certificate_source: ENCRYPTION_SOURCE.CONTENT,
            key_source: ENCRYPTION_SOURCE.PATH,
            private_key: 'key',
            private_key_saved: false,
        } as any);
        expect(result.certificate_chain).toBe('cert');
        expect(result.certificate_path).toBe('');
    });

    it('empties private_key when private_key_saved is true', () => {
        const result = getSubmitValues({
            enabled: true,
            private_key: 'key',
            private_key_path: '',
            key_source: ENCRYPTION_SOURCE.SAVED,
            private_key_saved: true,
        } as any);
        expect(result.private_key).toBe('');
        expect(result.private_key_saved).toBe(true);
    });

    it('defaultTlsValues has all required fields', () => {
        expect(defaultTlsValues.enabled).toBe(false);
        expect(defaultTlsValues.serve_plain_dns).toBe(true);
        expect(defaultTlsValues.port_https).toBe(443);
        expect(defaultTlsValues.port_dns_over_tls).toBe(853);
        expect(defaultTlsValues.port_dns_over_quic).toBe(853);
        expect(defaultTlsValues.certificate_source).toBe(ENCRYPTION_SOURCE.PATH);
        expect(defaultTlsValues.private_key_saved).toBe(false);
    });
});
