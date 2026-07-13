import { describe, it, expect } from 'vitest';

import {
    validateCertFields,
    validateKeyFields,
    validateCertKeyFields,
    validateEncryptionForm,
} from 'panel/components/Encryption/validate';
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

    it('validates cert/key fields even when encryption is disabled', () => {
        const errs = validateEncryptionForm({
            ...valid,
            enabled: false,
            serve_plain_dns: true,
            certificate_chain: '',
            private_key: '',
        });
        expect(errs.certificate_chain).toBeTruthy();
        expect(errs.private_key).toBeTruthy();
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

    it('allows empty-string ports (coerces to 0, valid as disabled)', () => {
        // Before the store loads, ports may be empty strings.
        // Number('') || 0 → 0, which is now valid (0 = disabled port).
        const errs = validateEncryptionForm({
            ...valid,
            port_https: '' as any,
        });
        expect(errs.port_https).toBeUndefined();
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

describe('validateCertFields', () => {
    it('returns no errors for a valid cert chain', () => {
        expect(validateCertFields(valid)).toEqual({});
    });

    it('returns no errors for a valid cert path', () => {
        const errs = validateCertFields({
            ...valid,
            certificate_source: ENCRYPTION_SOURCE.PATH,
            certificate_path: '/etc/ssl/cert.pem',
            certificate_chain: '',
        });
        expect(errs).toEqual({});
    });

    it('requires certificate_chain when source is CONTENT', () => {
        const errs = validateCertFields({ ...valid, certificate_chain: '' });
        expect(errs.certificate_chain).toBeTruthy();
    });

    it('requires certificate_path when source is PATH', () => {
        const errs = validateCertFields({
            ...valid,
            certificate_source: ENCRYPTION_SOURCE.PATH,
            certificate_path: '',
            certificate_chain: '',
        });
        expect(errs.certificate_path).toBeTruthy();
    });

    it('validates cert fields even when encryption is disabled', () => {
        const errs = validateCertFields({
            ...valid,
            enabled: false,
            certificate_chain: '',
        });
        expect(errs.certificate_chain).toBeTruthy();
    });
});

describe('validateKeyFields', () => {
    it('returns no errors for a valid private key', () => {
        expect(validateKeyFields(valid)).toEqual({});
    });

    it('returns no errors for a valid key path', () => {
        const errs = validateKeyFields({
            ...valid,
            key_source: ENCRYPTION_SOURCE.PATH,
            private_key_path: '/etc/ssl/key.pem',
            private_key: '',
        });
        expect(errs).toEqual({});
    });

    it('requires private_key when source is CONTENT', () => {
        const errs = validateKeyFields({ ...valid, private_key: '' });
        expect(errs.private_key).toBeTruthy();
    });

    it('requires private_key_path when source is PATH', () => {
        const errs = validateKeyFields({
            ...valid,
            key_source: ENCRYPTION_SOURCE.PATH,
            private_key_path: '',
            private_key: '',
        });
        expect(errs.private_key_path).toBeTruthy();
    });

    it('returns no errors when private_key_saved is true', () => {
        const errs = validateKeyFields({
            ...valid,
            private_key: '',
            private_key_saved: true,
        });
        expect(errs).toEqual({});
    });

    it('validates key fields even when encryption is disabled', () => {
        const errs = validateKeyFields({
            ...valid,
            enabled: false,
            private_key: '',
        });
        expect(errs.private_key).toBeTruthy();
    });

    it('rejects non-PEM cert content', () => {
        const errs = validateCertFields({
            ...valid,
            certificate_chain: 'not a pem certificate',
        });
        expect(errs.certificate_chain).toBeTruthy();
    });

    it('rejects non-PEM key content', () => {
        const errs = validateKeyFields({
            ...valid,
            private_key: 'not a pem key',
        });
        expect(errs.private_key).toBeTruthy();
    });

    it('rejects invalid cert path', () => {
        const errs = validateCertFields({
            ...valid,
            certificate_source: ENCRYPTION_SOURCE.PATH,
            certificate_path: 'relative/path/no/leading/slash',
            certificate_chain: '',
        });
        expect(errs.certificate_path).toBeTruthy();
    });

    it('rejects invalid key path', () => {
        const errs = validateKeyFields({
            ...valid,
            key_source: ENCRYPTION_SOURCE.PATH,
            private_key_path: 'relative/path/no/leading/slash',
            private_key: '',
        });
        expect(errs.private_key_path).toBeTruthy();
    });
});

describe('validateCertKeyFields', () => {
    it('returns no errors when both cert and key are valid', () => {
        expect(validateCertKeyFields(valid)).toEqual({});
    });

    it('returns only cert errors when cert is invalid but key is valid', () => {
        const errs = validateCertKeyFields({ ...valid, certificate_chain: '' });
        expect(errs.certificate_chain).toBeTruthy();
        expect(errs.private_key).toBeUndefined();
    });

    it('returns only key errors when key is invalid but cert is valid', () => {
        const errs = validateCertKeyFields({ ...valid, private_key: '' });
        expect(errs.certificate_chain).toBeUndefined();
        expect(errs.private_key).toBeTruthy();
    });

    it('returns both cert and key errors when both are invalid', () => {
        const errs = validateCertKeyFields({
            ...valid,
            certificate_chain: '',
            private_key: '',
        });
        expect(errs.certificate_chain).toBeTruthy();
        expect(errs.private_key).toBeTruthy();
    });

    it('validates cert/key fields even when encryption is disabled', () => {
        const errs = validateCertKeyFields({
            ...valid,
            enabled: false,
            certificate_chain: '',
            private_key: '',
        });
        expect(errs.certificate_chain).toBeTruthy();
        expect(errs.private_key).toBeTruthy();
    });
});
