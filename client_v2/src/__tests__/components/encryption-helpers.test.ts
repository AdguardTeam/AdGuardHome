import { describe, it, expect } from 'vitest';
import {
    getSubmitValues,
    getStepValidationValues,
    defaultTlsValues,
} from 'panel/components/Encryption/blocks/helpers';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';

const contentValues = {
    ...defaultTlsValues,
    certificate_source: ENCRYPTION_SOURCE.CONTENT,
    key_source: ENCRYPTION_SOURCE.CONTENT,
};

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

    it('states private_key_saved explicitly for the content source', () => {
        const fresh = getSubmitValues({
            enabled: true,
            private_key: 'key',
            private_key_path: '',
            key_source: ENCRYPTION_SOURCE.CONTENT,
            private_key_saved: false,
        } as any);
        expect(fresh.private_key).toBe('key');
        expect(fresh.private_key_saved).toBe(false);

        const saved = getSubmitValues({
            enabled: true,
            private_key: 'key',
            private_key_path: '',
            key_source: ENCRYPTION_SOURCE.CONTENT,
            private_key_saved: true,
        } as any);
        expect(saved.private_key).toBe('');
        expect(saved.private_key_saved).toBe(true);
    });

    it('clears private_key_saved when the key comes from a file path', () => {
        const result = getSubmitValues({
            enabled: true,
            private_key: '',
            private_key_path: '/path/key.pem',
            key_source: ENCRYPTION_SOURCE.PATH,
            private_key_saved: true,
        } as any);
        expect(result.private_key_path).toBe('/path/key.pem');
        expect(result.private_key_saved).toBe(false);
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

describe('getStepValidationValues', () => {
    it('builds a cert-only, port-free payload for step 1', () => {
        const p = getStepValidationValues(1, {
            ...contentValues,
            certificate_chain: 'CERT',
            private_key: 'KEY',
        });
        expect(p).toMatchObject({
            enabled: false,
            serve_plain_dns: true,
            server_name: '',
            private_key: '',
            private_key_path: '',
            private_key_saved: false,
        });
        expect(p.certificate_chain).toBe('CERT');
    });

    it('keeps cert + key and drops the server name for step 2', () => {
        const p = getStepValidationValues(2, {
            ...contentValues,
            certificate_chain: 'CERT',
            server_name: 'example.com',
            private_key: 'KEY',
        });
        expect(p).toMatchObject({ enabled: false, server_name: '' });
        expect(p.certificate_chain).toBe('CERT');
        expect(p.private_key).toBe('KEY');
    });

    it('enables encryption for the step 3 pre-flight', () => {
        expect(getStepValidationValues(3, defaultTlsValues).enabled).toBe(true);
    });

    it('keeps the configured server name on step 3', () => {
        const p = getStepValidationValues(3, {
            ...contentValues,
            server_name: 'example.com',
        });
        expect(p.server_name).toBe('example.com');
        expect(p.serve_plain_dns).toBe(true);
    });

    it('respects path sources and clears the opposite field', () => {
        const p = getStepValidationValues(2, {
            ...contentValues,
            certificate_source: ENCRYPTION_SOURCE.PATH,
            certificate_path: '/etc/ssl/cert.pem',
            certificate_chain: 'CERT',
            key_source: ENCRYPTION_SOURCE.PATH,
            private_key_path: '/etc/ssl/key.pem',
            private_key: 'KEY',
        });
        expect(p.certificate_chain).toBe('');
        expect(p.certificate_path).toBe('/etc/ssl/cert.pem');
        expect(p.private_key).toBe('');
        expect(p.private_key_path).toBe('/etc/ssl/key.pem');
    });
});
