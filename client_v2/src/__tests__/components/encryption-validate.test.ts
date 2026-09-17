import { describe, it, expect } from 'vitest';

import {
    validateCertFields,
    validateKeyFields,
    validateCertKeyFields,
    validateEncryptionForm,
    validatePortField,
    validateServerSettings,
} from 'panel/components/Encryption/validate';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';

/**
 * The copy the validators return, read from the same locale file the app loads.
 * Asserting via the keys keeps the tests about *which* message is produced, so
 * rewording it does not break them.
 */
import en from 'panel/__locales/en.json';
import { copy } from 'panel/__tests__/helpers/copy';

/** Copy several cases assert, named by key so a reword cannot break them. */
const PORT_IN_USE = copy('tls_setup_error_port_in_use');
const NOT_A_CERT = copy('tls_setup_error_not_a_cert');
const NOT_A_KEY = copy('tls_setup_error_not_a_key');
const UNSAFE_PORT = copy('form_error_port_unsafe');

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

    it('flags equal HTTPS and DoT ports as a conflict', () => {
        const errs = validateEncryptionForm({
            ...valid,
            port_https: 853,
            port_dns_over_tls: 853,
        });
        expect(errs.port_https).toBe(PORT_IN_USE);
        expect(errs.port_dns_over_tls).toBe(PORT_IN_USE);
    });

    it('flags the fields whose port is already used by another AGH setting', () => {
        const errs = validateEncryptionForm(valid, {
            webUi: 443,
            plainDns: 53,
            dnscrypt: 0,
        });
        expect(errs.port_https).toBe(PORT_IN_USE);
        expect(errs.port_dns_over_tls).toBeUndefined();
    });

    it('allows DoT and DoQ to share a port (different families)', () => {
        const errs = validateEncryptionForm({
            ...valid,
            port_dns_over_tls: 853,
            port_dns_over_quic: 853,
        });
        expect(errs.port_dns_over_tls).toBeUndefined();
        expect(errs.port_dns_over_quic).toBeUndefined();
    });

    it('flags a DoQ port that collides with the plain DNS port (UDP family)', () => {
        const errs = validateEncryptionForm(
            { ...valid, port_dns_over_quic: 5353 },
            { plainDns: 5353 },
        );
        expect(errs.port_dns_over_quic).toBe(PORT_IN_USE);
        expect(errs.port_dns_over_tls).toBeUndefined();
    });

    it('ignores duplicates between external settings only', () => {
        expect(validateEncryptionForm(valid, { webUi: 53, plainDns: 53 })).toEqual({});
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

    it('rejects a private key pasted into the certificate field', () => {
        const errs = validateCertFields({
            ...valid,
            certificate_chain: '-----BEGIN PRIVATE KEY-----\nxyz\n-----END PRIVATE KEY-----',
        });
        expect(errs.certificate_chain).toBe(NOT_A_CERT);
    });

    it('asks for the whole certificate when it is not a complete PEM block', () => {
        const errs = validateCertFields({
            ...valid,
            certificate_chain: 'MIIDTCCAHgAwIBAgIJ',
        });
        expect(errs.certificate_chain).toBe(en.tls_setup_error_cert_incomplete);
    });

    it('asks for the whole certificate when the closing line is missing', () => {
        const errs = validateCertFields({
            ...valid,
            certificate_chain: '-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJAKlM4NvZ5W4r',
        });
        expect(errs.certificate_chain).toBe(en.tls_setup_error_cert_incomplete);
    });

    it('keeps the wrong-content error for a truncated private key in the certificate field', () => {
        // Pasting a key into the certificate field is the more useful verdict,
        // whether or not that key happens to be truncated.
        const errs = validateCertFields({
            ...valid,
            certificate_chain: '-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkq',
        });
        expect(errs.certificate_chain).toBe(NOT_A_CERT);
    });

    it('accepts a chain whose last block is complete', () => {
        const chain = [
            '-----BEGIN CERTIFICATE-----',
            'MIIDXTCCAkWg',
            '-----END CERTIFICATE-----',
            '-----BEGIN CERTIFICATE-----',
            'MIIDXTCCAkWh',
            '-----END CERTIFICATE-----',
        ].join('\n');
        expect(validateCertFields({ ...valid, certificate_chain: chain })).toEqual({});
    });

    it('accepts a certificate block so the backend can parse it', () => {
        expect(validateCertFields(valid).certificate_chain).toBeUndefined();
        expect(
            validateCertFields({
                ...valid,
                certificate_chain: '-----BEGIN CERTIFICATE-----\nbroken\n-----END CERTIFICATE-----',
            }).certificate_chain,
        ).toBeUndefined(); // parse errors come from the backend
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

    it('rejects a certificate pasted into the private key field', () => {
        const errs = validateKeyFields({
            ...valid,
            private_key: '-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----',
        });
        expect(errs.private_key).toBe(NOT_A_KEY);
    });

    it('rejects invalid cert path', () => {
        const errs = validateCertFields({
            ...valid,
            certificate_source: ENCRYPTION_SOURCE.PATH,
            certificate_path: 'relative/path/no/leading/slash',
            certificate_chain: '',
        });
        expect(errs.certificate_path).toBe(copy('tls_setup_error_read_cert'));
    });

    it('rejects invalid key path', () => {
        const errs = validateKeyFields({
            ...valid,
            key_source: ENCRYPTION_SOURCE.PATH,
            private_key_path: 'relative/path/no/leading/slash',
            private_key: '',
        });
        expect(errs.private_key_path).toBe(copy('tls_setup_error_read_key'));
    });

    it('asks for the whole key when it is not a complete PEM block', () => {
        expect(validateKeyFields({ ...valid, private_key: 'MIIEvQIBADANBgkq' }).private_key).toBe(
            en.tls_setup_error_key_incomplete,
        );
    });

    it('asks for the whole key when the closing line is missing', () => {
        const errs = validateKeyFields({
            ...valid,
            private_key: '-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0B',
        });
        expect(errs.private_key).toBe(en.tls_setup_error_key_incomplete);
    });

    it('keeps the wrong-content error for a truncated certificate in the key field', () => {
        const errs = validateKeyFields({
            ...valid,
            private_key: '-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWg',
        });
        expect(errs.private_key).toBe(NOT_A_KEY);
    });

    it('accepts any private key flavour whose closing line is complete', () => {
        const rsaKey =
            '-----BEGIN RSA PRIVATE KEY-----\nMIIEvQIBADANBgkq\n-----END RSA PRIVATE KEY-----';
        expect(validateKeyFields({ ...valid, private_key: rsaKey })).toEqual({});
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

describe('validatePortField', () => {
    it('accepts 0 as a disabled port', () => {
        expect(validatePortField('port_https', 0)).toBeUndefined();
        expect(validatePortField('port_dns_over_quic', 0)).toBeUndefined();
    });

    it('flags an out-of-range port', () => {
        expect(validatePortField('port_dns_over_tls', 99999)).toBeTruthy();
    });

    it('flags an unsafe HTTPS port but not the same port on DoT/DoQ', () => {
        // Browsers block a set of ports for the Web UI; the same restriction
        // does not apply to the other DNS protocols.
        expect(validatePortField('port_https', 22)).toBe(UNSAFE_PORT);
        expect(validatePortField('port_dns_over_tls', 22)).toBeUndefined();
        expect(validatePortField('port_dns_over_quic', 22)).toBeUndefined();
    });
});

describe('validateServerSettings', () => {
    const settings = {
        server_name: 'dns.example.com',
        port_https: 443,
        port_dns_over_tls: 853,
        port_dns_over_quic: 853,
    };
    const external = { webUi: 80, plainDns: 53, dnscrypt: 0 };

    it('returns no errors for a clean set of settings', () => {
        expect(validateServerSettings(settings, external)).toEqual({});
    });

    it('returns no errors for an empty set of settings', () => {
        expect(validateServerSettings({})).toEqual({});
    });

    it('treats the server name as optional but validates its format', () => {
        expect(validateServerSettings({ server_name: '' }).server_name).toBeUndefined();
        expect(validateServerSettings({ server_name: 'not a domain' }).server_name).toBe(
            copy('form_error_server_name'),
        );
    });

    it('flags a port already used by another AdGuard Home setting', () => {
        const errs = validateServerSettings({ ...settings, port_https: 80 }, external);
        expect(errs.port_https).toBe(PORT_IN_USE);
        expect(errs.port_dns_over_tls).toBeUndefined();
    });

    it('flags repeated ports between the encrypted DNS fields', () => {
        const errs = validateServerSettings({
            ...settings,
            port_https: 853,
            port_dns_over_tls: 853,
        });
        expect(errs.port_https).toBe(PORT_IN_USE);
        expect(errs.port_dns_over_tls).toBe(PORT_IN_USE);
    });

    it('flags an unsafe HTTPS port', () => {
        expect(validateServerSettings({ ...settings, port_https: 22 }).port_https).toBe(
            UNSAFE_PORT,
        );
    });

    it('covers only the settings it owns, not cert/key or plain DNS', () => {
        const errs = validateServerSettings({ server_name: '' });
        expect(errs.certificate_chain).toBeUndefined();
        expect(errs.private_key).toBeUndefined();
        expect(errs.serve_plain_dns).toBeUndefined();
    });
});
