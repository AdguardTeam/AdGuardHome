import { describe, it, expect } from 'vitest';

import { validateEncryptionForm } from 'panel/components/Encryption/validate';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';

const valid = {
    enabled: true,
    serve_plain_dns: true,
    server_name: 'dns.example.com',
    force_https: false,
    port_https: 443,
    port_dns_over_tls: 853,
    port_dns_over_quic: 853,
    certificate_chain: '-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----',
    private_key: '-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----',
    certificate_path: '',
    private_key_path: '',
    certificate_source: ENCRYPTION_SOURCE.CONTENT,
    key_source: ENCRYPTION_SOURCE.CONTENT,
    private_key_saved: false,
};

describe('validateEncryptionForm', () => {
    it('returns no errors for a fully valid enabled form', () => {
        expect(validateEncryptionForm(valid)).toEqual({});
    });

    it('requires certificate_chain when enabled and source is content', () => {
        const errs = validateEncryptionForm({ ...valid, certificate_chain: '' });
        expect(errs.certificate_chain).toBeTruthy();
    });

    it('requires certificate_path when enabled and source is path', () => {
        const errs = validateEncryptionForm({
            ...valid,
            certificate_source: ENCRYPTION_SOURCE.PATH,
            certificate_path: '',
            certificate_chain: '',
        });
        expect(errs.certificate_path).toBeTruthy();
    });

    it('requires private_key unless private_key_saved is true', () => {
        const errs = validateEncryptionForm({ ...valid, private_key: '' });
        expect(errs.private_key).toBeTruthy();

        const ok = validateEncryptionForm({
            ...valid,
            private_key: '',
            private_key_saved: true,
        });
        expect(ok.private_key).toBeUndefined();
    });

    it('does NOT require cert/key when encryption is disabled', () => {
        const errs = validateEncryptionForm({
            ...valid,
            enabled: false,
            serve_plain_dns: true,
            certificate_chain: '',
            private_key: '',
        });
        expect(errs.certificate_chain).toBeUndefined();
        expect(errs.private_key).toBeUndefined();
    });

    it('flags an out-of-range https port', () => {
        const errs = validateEncryptionForm({ ...valid, port_https: 99999 });
        expect(errs.port_https).toBeTruthy();
    });

    it('flags an unsafe https port', () => {
        const errs = validateEncryptionForm({ ...valid, port_https: 22 });
        expect(errs.port_https).toBeTruthy();
    });

    it('allows port 0 for DoT/DoQ (disabled)', () => {
        const errs = validateEncryptionForm({
            ...valid,
            port_dns_over_tls: 0,
            port_dns_over_quic: 0,
        });
        expect(errs.port_dns_over_tls).toBeUndefined();
        expect(errs.port_dns_over_quic).toBeUndefined();
    });

    it('flags equal https and DoT ports', () => {
        const errs = validateEncryptionForm({
            ...valid,
            port_https: 853,
            port_dns_over_tls: 853,
        });
        expect(errs.port_https).toBeTruthy();
        expect(errs.port_dns_over_tls).toBeTruthy();
    });

    it('flags empty-string ports as invalid (coerces to 0)', () => {
        // Before the store loads, ports may be empty strings.
        // Number('') || 0 → 0, which fails validatePort (0 < 80).
        const errs = validateEncryptionForm({
            ...valid,
            port_https: '' as any,
        });
        expect(errs.port_https).toBeTruthy();
    });

    it('requires plain DNS when encryption is disabled', () => {
        const errs = validateEncryptionForm({
            ...valid,
            enabled: false,
            serve_plain_dns: false,
            certificate_chain: '',
            private_key: '',
        });
        expect(errs.serve_plain_dns).toBeTruthy();
    });

    it('flags an invalid server name', () => {
        const errs = validateEncryptionForm({
            ...valid,
            server_name: 'not valid!',
        });
        expect(errs.server_name).toBeTruthy();
    });
});
